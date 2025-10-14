package apps

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/wild-cloud/wild-central/daemon/internal/secrets"
	"github.com/wild-cloud/wild-central/daemon/internal/storage"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
)

// Manager handles application lifecycle operations
type Manager struct {
	dataDir string
	appsDir string // Path to apps directory in repo
}

// NewManager creates a new apps manager
func NewManager(dataDir, appsDir string) *Manager {
	return &Manager{
		dataDir: dataDir,
		appsDir: appsDir,
	}
}

// App represents an application
type App struct {
	Name         string            `json:"name" yaml:"name"`
	Description  string            `json:"description" yaml:"description"`
	Version      string            `json:"version" yaml:"version"`
	Category     string            `json:"category" yaml:"category"`
	Dependencies []string          `json:"dependencies" yaml:"dependencies"`
	Config       map[string]string `json:"config,omitempty" yaml:"config,omitempty"`
}

// DeployedApp represents a deployed application instance
type DeployedApp struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Version   string `json:"version"`
	Namespace string `json:"namespace"`
	URL       string `json:"url,omitempty"`
}

// ListAvailable lists all available apps from the apps directory
func (m *Manager) ListAvailable() ([]App, error) {
	if m.appsDir == "" {
		return []App{}, fmt.Errorf("apps directory not configured")
	}

	// Read apps directory
	entries, err := os.ReadDir(m.appsDir)
	if err != nil {
		return []App{}, fmt.Errorf("failed to read apps directory: %w", err)
	}

	apps := []App{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check for manifest.yaml
		appFile := filepath.Join(m.appsDir, entry.Name(), "manifest.yaml")
		if !storage.FileExists(appFile) {
			continue
		}

		// Parse manifest.yaml
		data, err := os.ReadFile(appFile)
		if err != nil {
			continue
		}

		var app App
		if err := yaml.Unmarshal(data, &app); err != nil {
			continue
		}

		app.Name = entry.Name() // Use directory name as app name
		apps = append(apps, app)
	}

	return apps, nil
}

// Get returns details for a specific available app
func (m *Manager) Get(appName string) (*App, error) {
	appFile := filepath.Join(m.appsDir, appName, "manifest.yaml")

	if !storage.FileExists(appFile) {
		return nil, fmt.Errorf("app %s not found", appName)
	}

	data, err := os.ReadFile(appFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read app file: %w", err)
	}

	var app App
	if err := yaml.Unmarshal(data, &app); err != nil {
		return nil, fmt.Errorf("failed to parse app file: %w", err)
	}

	app.Name = appName
	return &app, nil
}

// ListDeployed lists deployed apps for an instance
func (m *Manager) ListDeployed(instanceName string) ([]DeployedApp, error) {
	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)
	instancePath := filepath.Join(m.dataDir, "instances", instanceName)
	appsDir := filepath.Join(instancePath, "apps")

	apps := []DeployedApp{}

	// Check if apps directory exists
	if !storage.FileExists(appsDir) {
		return apps, nil
	}

	// List all app directories
	entries, err := os.ReadDir(appsDir)
	if err != nil {
		return apps, fmt.Errorf("failed to read apps directory: %w", err)
	}

	// For each app directory, check if it's deployed in the cluster
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		appName := entry.Name()

		// Initialize app with basic info
		app := DeployedApp{
			Name:      appName,
			Namespace: appName,
			Status:    "added", // Default status: added but not deployed
		}

		// Try to get version from manifest
		manifestPath := filepath.Join(appsDir, appName, "manifest.yaml")
		if storage.FileExists(manifestPath) {
			manifestData, _ := os.ReadFile(manifestPath)
			var manifest struct {
				Version string `yaml:"version"`
			}
			if yaml.Unmarshal(manifestData, &manifest) == nil {
				app.Version = manifest.Version
			}
		}

		// Check if namespace exists in cluster
		checkCmd := exec.Command("kubectl", "get", "namespace", appName, "-o", "json")
		tools.WithKubeconfig(checkCmd, kubeconfigPath)
		output, err := checkCmd.CombinedOutput()

		if err == nil {
			// Namespace exists - parse status
			var ns struct {
				Status struct {
					Phase string `json:"phase"`
				} `json:"status"`
			}
			if yaml.Unmarshal(output, &ns) == nil && ns.Status.Phase == "Active" {
				// Namespace is active - app is deployed
				app.Status = "deployed"
			}
		}

		apps = append(apps, app)
	}

	return apps, nil
}

// Add adds an app to the instance configuration
func (m *Manager) Add(instanceName, appName string, config map[string]string) error {
	// 1. Verify app exists
	manifestPath := filepath.Join(m.appsDir, appName, "manifest.yaml")
	if !storage.FileExists(manifestPath) {
		return fmt.Errorf("app %s not found at %s", appName, manifestPath)
	}

	instancePath := filepath.Join(m.dataDir, "instances", instanceName)
	configFile := filepath.Join(instancePath, "config.yaml")
	secretsFile := filepath.Join(instancePath, "secrets.yaml")
	appDestDir := filepath.Join(instancePath, "apps", appName)

	// Check instance config exists
	if !storage.FileExists(configFile) {
		return fmt.Errorf("instance config not found: %s", instanceName)
	}

	// 2. Process manifest with gomplate
	tempManifest := filepath.Join(os.TempDir(), fmt.Sprintf("manifest-%s.yaml", appName))
	defer os.Remove(tempManifest)

	gomplate := tools.NewGomplate()
	context := map[string]string{
		".":       configFile,
		"secrets": secretsFile,
	}
	if err := gomplate.RenderWithContext(manifestPath, tempManifest, context); err != nil {
		return fmt.Errorf("failed to process manifest: %w", err)
	}

	// Parse processed manifest
	manifestData, err := os.ReadFile(tempManifest)
	if err != nil {
		return fmt.Errorf("failed to read processed manifest: %w", err)
	}

	var manifest struct {
		DefaultConfig   map[string]interface{} `yaml:"defaultConfig"`
		RequiredSecrets []string               `yaml:"requiredSecrets"`
	}
	if err := yaml.Unmarshal(manifestData, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	// 3. Update configuration
	yq := tools.NewYQ()
	configLock := configFile + ".lock"

	if err := storage.WithLock(configLock, func() error {
		// Ensure apps section exists
		expr := fmt.Sprintf(".apps.%s = .apps.%s // {}", appName, appName)
		if _, err := yq.Exec("-i", expr, configFile); err != nil {
			return fmt.Errorf("failed to ensure apps section: %w", err)
		}

		// Merge defaultConfig (preserves existing values)
		if len(manifest.DefaultConfig) > 0 {
			for key, value := range manifest.DefaultConfig {
				keyPath := fmt.Sprintf(".apps.%s.%s", appName, key)
				// Only set if not already present
				existing, _ := yq.Get(configFile, keyPath)
				if existing == "" || existing == "null" {
					if err := yq.Set(configFile, keyPath, fmt.Sprintf("%v", value)); err != nil {
						return fmt.Errorf("failed to set config %s: %w", key, err)
					}
				}
			}
		}

		// Apply user-provided config overrides
		for key, value := range config {
			keyPath := fmt.Sprintf(".apps.%s.%s", appName, key)
			if err := yq.Set(configFile, keyPath, value); err != nil {
				return fmt.Errorf("failed to set config %s: %w", key, err)
			}
		}

		return nil
	}); err != nil {
		return err
	}

	// 4. Generate required secrets
	secretsMgr := secrets.NewManager()
	for _, secretKey := range manifest.RequiredSecrets {
		if _, err := secretsMgr.EnsureSecret(secretsFile, secretKey, secrets.DefaultSecretLength); err != nil {
			return fmt.Errorf("failed to ensure secret %s: %w", secretKey, err)
		}
	}

	// 5. Copy and compile app files
	if err := storage.EnsureDir(appDestDir, 0755); err != nil {
		return fmt.Errorf("failed to create app directory: %w", err)
	}

	// Copy source app directory
	sourceAppDir := filepath.Join(m.appsDir, appName)
	entries, err := os.ReadDir(sourceAppDir)
	if err != nil {
		return fmt.Errorf("failed to read app directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// TODO: Handle subdirectories if needed
			continue
		}

		sourcePath := filepath.Join(sourceAppDir, entry.Name())
		destPath := filepath.Join(appDestDir, entry.Name())

		// Process with gomplate
		if err := gomplate.RenderWithContext(sourcePath, destPath, context); err != nil {
			return fmt.Errorf("failed to compile %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// Deploy deploys an app to the cluster
func (m *Manager) Deploy(instanceName, appName string) error {
	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)
	instancePath := filepath.Join(m.dataDir, "instances", instanceName)
	secretsFile := filepath.Join(instancePath, "secrets.yaml")

	// Get compiled app manifests from instance directory
	appDir := filepath.Join(instancePath, "apps", appName)
	if !storage.FileExists(appDir) {
		return fmt.Errorf("app %s not found in instance (run 'wild app add %s' first)", appName, appName)
	}

	// Create namespace if it doesn't exist
	namespaceCmd := exec.Command("kubectl", "create", "namespace", appName, "--dry-run=client", "-o", "yaml")
	tools.WithKubeconfig(namespaceCmd, kubeconfigPath)
	namespaceYaml, _ := namespaceCmd.CombinedOutput()

	applyNsCmd := exec.Command("kubectl", "apply", "-f", "-")
	applyNsCmd.Stdin = bytes.NewReader(namespaceYaml)
	tools.WithKubeconfig(applyNsCmd, kubeconfigPath)
	_, _ = applyNsCmd.CombinedOutput() // Ignore errors - namespace might already exist

	// Create Kubernetes secrets from secrets.yaml
	if storage.FileExists(secretsFile) {
		yq := tools.NewYQ()
		appSecretsPath := fmt.Sprintf(".apps.%s", appName)
		appSecretsJson, err := yq.Get(secretsFile, fmt.Sprintf("%s | @json", appSecretsPath))
		if err == nil && appSecretsJson != "" && appSecretsJson != "null" {
			// Delete existing secret if it exists (to update it)
			deleteCmd := exec.Command("kubectl", "delete", "secret", fmt.Sprintf("%s-secrets", appName), "-n", appName, "--ignore-not-found")
			tools.WithKubeconfig(deleteCmd, kubeconfigPath)
			_, _ = deleteCmd.CombinedOutput()

			// Create secret from literals
			createSecretCmd := exec.Command("kubectl", "create", "secret", "generic", fmt.Sprintf("%s-secrets", appName), "-n", appName)

			// Parse secrets and add as literals
			var appSecrets map[string]string
			if err := yaml.Unmarshal([]byte(appSecretsJson), &appSecrets); err == nil {
				for key, value := range appSecrets {
					secretKey := fmt.Sprintf("apps.%s.%s", appName, key)
					createSecretCmd.Args = append(createSecretCmd.Args, fmt.Sprintf("--from-literal=%s=%s", secretKey, value))
				}
			}

			tools.WithKubeconfig(createSecretCmd, kubeconfigPath)
			if output, err := createSecretCmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to create secret: %w\nOutput: %s", err, string(output))
			}
		}
	}

	// Apply manifests with kubectl using kustomize
	cmd := exec.Command("kubectl", "apply", "-k", appDir)
	tools.WithKubeconfig(cmd, kubeconfigPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to deploy app: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// Delete removes an app from the cluster and configuration
func (m *Manager) Delete(instanceName, appName string) error {
	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)
	instancePath := filepath.Join(m.dataDir, "instances", instanceName)
	configFile := filepath.Join(instancePath, "config.yaml")
	secretsFile := filepath.Join(instancePath, "secrets.yaml")

	// Get compiled app manifests from instance directory
	appDir := filepath.Join(instancePath, "apps", appName)
	if !storage.FileExists(appDir) {
		return fmt.Errorf("app %s not found in instance", appName)
	}

	// Delete namespace (this deletes ALL resources including deployments, services, secrets, etc.)
	deleteNsCmd := exec.Command("kubectl", "delete", "namespace", appName, "--ignore-not-found")
	tools.WithKubeconfig(deleteNsCmd, kubeconfigPath)
	output, err := deleteNsCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete namespace: %w\nOutput: %s", err, string(output))
	}

	// Wait for namespace deletion to complete (timeout after 60s)
	waitCmd := exec.Command("kubectl", "wait", "--for=delete", "namespace", appName, "--timeout=60s")
	tools.WithKubeconfig(waitCmd, kubeconfigPath)
	_, _ = waitCmd.CombinedOutput() // Ignore errors - namespace might not exist

	// Delete local app configuration directory
	if err := os.RemoveAll(appDir); err != nil {
		return fmt.Errorf("failed to delete local app directory: %w", err)
	}

	// Remove app config from config.yaml
	yq := tools.NewYQ()
	configLock := configFile + ".lock"
	if storage.FileExists(configFile) {
		if err := storage.WithLock(configLock, func() error {
			return yq.Delete(configFile, fmt.Sprintf(".apps.%s", appName))
		}); err != nil {
			return fmt.Errorf("failed to remove app config: %w", err)
		}
	}

	// Remove app secrets from secrets.yaml
	secretsMgr := secrets.NewManager()
	if storage.FileExists(secretsFile) {
		if err := secretsMgr.DeleteSecret(secretsFile, fmt.Sprintf("apps.%s", appName)); err != nil {
			// Don't fail if secret doesn't exist
			if !strings.Contains(err.Error(), "not found") {
				return fmt.Errorf("failed to remove app secrets: %w", err)
			}
		}
	}

	return nil
}

// GetStatus returns the status of a deployed app
func (m *Manager) GetStatus(instanceName, appName string) (*DeployedApp, error) {
	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)
	instancePath := filepath.Join(m.dataDir, "instances", instanceName)
	appDir := filepath.Join(instancePath, "apps", appName)

	app := &DeployedApp{
		Name:      appName,
		Status:    "not-added",
		Namespace: appName,
	}

	// Check if app was added to instance
	if !storage.FileExists(appDir) {
		return app, nil
	}

	app.Status = "not-deployed"

	// Get version from manifest
	manifestPath := filepath.Join(appDir, "manifest.yaml")
	if storage.FileExists(manifestPath) {
		manifestData, _ := os.ReadFile(manifestPath)
		var manifest struct {
			Version string `yaml:"version"`
		}
		if yaml.Unmarshal(manifestData, &manifest) == nil {
			app.Version = manifest.Version
		}
	}

	// Check if namespace exists
	checkNsCmd := exec.Command("kubectl", "get", "namespace", appName, "-o", "json")
	tools.WithKubeconfig(checkNsCmd, kubeconfigPath)
	nsOutput, err := checkNsCmd.CombinedOutput()
	if err != nil {
		// Namespace doesn't exist - not deployed
		return app, nil
	}

	// Parse namespace to check if it's active
	var ns struct {
		Status struct {
			Phase string `json:"phase"`
		} `json:"status"`
	}
	if err := yaml.Unmarshal(nsOutput, &ns); err != nil || ns.Status.Phase != "Active" {
		return app, nil
	}

	// Namespace exists - check pod status
	podsCmd := exec.Command("kubectl", "get", "pods", "-n", appName, "-o", "json")
	tools.WithKubeconfig(podsCmd, kubeconfigPath)
	podsOutput, err := podsCmd.CombinedOutput()
	if err != nil {
		app.Status = "error"
		return app, nil
	}

	// Parse pods
	var podList struct {
		Items []struct {
			Status struct {
				Phase             string `json:"phase"`
				ContainerStatuses []struct {
					Ready bool `json:"ready"`
				} `json:"containerStatuses"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := yaml.Unmarshal(podsOutput, &podList); err != nil {
		app.Status = "error"
		return app, nil
	}

	if len(podList.Items) == 0 {
		app.Status = "no-pods"
		return app, nil
	}

	// Check pod status
	allRunning := true
	allReady := true
	for _, pod := range podList.Items {
		if pod.Status.Phase != "Running" {
			allRunning = false
		}
		for _, cs := range pod.Status.ContainerStatuses {
			if !cs.Ready {
				allReady = false
			}
		}
	}

	if allRunning && allReady {
		app.Status = "running"
	} else if allRunning {
		app.Status = "starting"
	} else {
		app.Status = "unhealthy"
	}

	return app, nil
}
