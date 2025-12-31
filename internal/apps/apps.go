package apps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/wild-cloud/wild-central/daemon/internal/secrets"
	"github.com/wild-cloud/wild-central/daemon/internal/storage"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
	"github.com/wild-cloud/wild-central/daemon/internal/utilities"
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
	Name           string                 `json:"name" yaml:"name"`
	Description    string                 `json:"description" yaml:"description"`
	Version        string                 `json:"version" yaml:"version"`
	Category       string                 `json:"category,omitempty" yaml:"category,omitempty"`
	Icon           string                 `json:"icon,omitempty" yaml:"icon,omitempty"`
	Requires       []AppDependency        `json:"requires,omitempty" yaml:"requires,omitempty"`
	Dependencies   []string               `json:"dependencies" yaml:"dependencies"` // Deprecated: kept for backward compatibility
	Config         map[string]string      `json:"config,omitempty" yaml:"config,omitempty"`
	DefaultConfig  map[string]interface{} `json:"defaultConfig,omitempty" yaml:"defaultConfig,omitempty"`
	DefaultSecrets []string               `json:"defaultSecrets,omitempty" yaml:"defaultSecrets,omitempty"`
}

// DeployedApp represents a deployed application instance
type DeployedApp struct {
	Name      string `json:"name"`
	Is        string `json:"is,omitempty"` // The original app type
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

		var manifest AppManifest
		if err := yaml.Unmarshal(data, &manifest); err != nil {
			continue
		}

		// Convert manifest to App struct
		app := App{
			Name:          entry.Name(), // Use directory name as app name
			Description:   manifest.Description,
			Version:       manifest.Version,
			Category:      manifest.Category,
			Icon:          manifest.Icon,
			DefaultConfig: manifest.DefaultConfig,
			Requires:      manifest.Requires, // Include full dependency information
		}

		// Extract secret keys from DefaultSecrets
		if len(manifest.DefaultSecrets) > 0 {
			app.DefaultSecrets = make([]string, len(manifest.DefaultSecrets))
			for i, secret := range manifest.DefaultSecrets {
				app.DefaultSecrets[i] = secret.Key
			}
		}

		// Extract dependencies from Requires field (for backward compatibility)
		if len(manifest.Requires) > 0 {
			app.Dependencies = make([]string, len(manifest.Requires))
			for i, dep := range manifest.Requires {
				app.Dependencies[i] = dep.Name
			}
		}

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

	var manifest AppManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse app file: %w", err)
	}

	// Convert manifest to App struct
	app := &App{
		Name:          appName,
		Description:   manifest.Description,
		Version:       manifest.Version,
		Category:      manifest.Category,
		Icon:          manifest.Icon,
		DefaultConfig: manifest.DefaultConfig,
		Requires:      manifest.Requires, // Include full dependency information
	}

	// Extract secret keys from DefaultSecrets
	if len(manifest.DefaultSecrets) > 0 {
		app.DefaultSecrets = make([]string, len(manifest.DefaultSecrets))
		for i, secret := range manifest.DefaultSecrets {
			app.DefaultSecrets[i] = secret.Key
		}
	}

	// Extract dependencies from Requires field (for backward compatibility)
	if len(manifest.Requires) > 0 {
		app.Dependencies = make([]string, len(manifest.Requires))
		for i, dep := range manifest.Requires {
			app.Dependencies[i] = dep.Name
		}
	}

	return app, nil
}

// ListDeployed lists deployed apps for an instance
func (m *Manager) ListDeployed(instanceName string) ([]DeployedApp, error) {
	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)
	instancePath := tools.GetInstancePath(m.dataDir, instanceName)
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

		// Try to get version and 'is' from manifest
		manifestPath := filepath.Join(appsDir, appName, "manifest.yaml")
		if storage.FileExists(manifestPath) {
			manifestData, _ := os.ReadFile(manifestPath)
			var manifest struct {
				Version string `yaml:"version"`
				Is      string `yaml:"is"`
			}
			if yaml.Unmarshal(manifestData, &manifest) == nil {
				app.Version = manifest.Version
				app.Is = manifest.Is
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

				// Get ingress URL if available
				// Try Traefik IngressRoute first
				ingressCmd := exec.Command("kubectl", "get", "ingressroute", "-n", appName, "-o", "json")
				tools.WithKubeconfig(ingressCmd, kubeconfigPath)
				ingressOutput, err := ingressCmd.CombinedOutput()

				if err == nil {
					var ingressList struct {
						Items []struct {
							Spec struct {
								Routes []struct {
									Match string `json:"match"`
								} `json:"routes"`
							} `json:"spec"`
						} `json:"items"`
					}
					if json.Unmarshal(ingressOutput, &ingressList) == nil && len(ingressList.Items) > 0 {
						// Extract host from the first route match (format: Host(`example.com`))
						if len(ingressList.Items[0].Spec.Routes) > 0 {
							match := ingressList.Items[0].Spec.Routes[0].Match
							// Parse Host(`domain.com`) format
							if strings.Contains(match, "Host(`") {
								start := strings.Index(match, "Host(`") + 6
								end := strings.Index(match[start:], "`")
								if end > 0 {
									host := match[start : start+end]
									app.URL = "https://" + host
								}
							}
						}
					}
				}

				// If no IngressRoute, try standard Ingress
				if app.URL == "" {
					ingressCmd := exec.Command("kubectl", "get", "ingress", "-n", appName, "-o", "json")
					tools.WithKubeconfig(ingressCmd, kubeconfigPath)
					ingressOutput, err := ingressCmd.CombinedOutput()

					if err == nil {
						var ingressList struct {
							Items []struct {
								Spec struct {
									Rules []struct {
										Host string `json:"host"`
									} `json:"rules"`
								} `json:"spec"`
							} `json:"items"`
						}
						if json.Unmarshal(ingressOutput, &ingressList) == nil && len(ingressList.Items) > 0 {
							if len(ingressList.Items[0].Spec.Rules) > 0 {
								host := ingressList.Items[0].Spec.Rules[0].Host
								if host != "" {
									app.URL = "https://" + host
								}
							}
						}
					}
				}
			}
		}

		apps = append(apps, app)
	}

	return apps, nil
}

// processConfigTemplates recursively processes gomplate templates in config values
// This function expects config values at the root context (e.g., {{ .cloud.domain }})
func processConfigTemplates(config map[string]interface{}, configFile string, gomplate *tools.Gomplate) (map[string]interface{}, error) {
	processed := make(map[string]interface{})

	for key, value := range config {
		switch v := value.(type) {
		case string:
			// Process string templates
			if strings.Contains(v, "{{") {
				// Use gomplate with config at root context
				args := []string{
					"-i", v,
					"-c", fmt.Sprintf(".=%s", configFile),
				}
				compiled, err := gomplate.Exec(args...)
				if err != nil {
					return nil, fmt.Errorf("failed to compile template for key %s: %w", key, err)
				}
				processed[key] = strings.TrimSpace(compiled)
			} else {
				processed[key] = v
			}
		case map[string]interface{}:
			// Recursively process nested maps
			nestedProcessed, err := processConfigTemplates(v, configFile, gomplate)
			if err != nil {
				return nil, err
			}
			processed[key] = nestedProcessed
		case map[interface{}]interface{}:
			// Convert to string map and process
			stringMap := make(map[string]interface{})
			for k, val := range v {
				if strKey, ok := k.(string); ok {
					stringMap[strKey] = val
				}
			}
			nestedProcessed, err := processConfigTemplates(stringMap, configFile, gomplate)
			if err != nil {
				return nil, err
			}
			processed[key] = nestedProcessed
		default:
			// Keep other types as-is
			processed[key] = value
		}
	}

	return processed, nil
}

// processSecretTemplate processes a gomplate template for secret defaults
// This function uses named contexts for config and secrets (e.g., {{ .config.apps.loomio.db.user }}, {{ .secrets.apps.loomio.dbPassword }})
func processSecretTemplate(template string, appName string, configFile, secretsFile string, gomplate *tools.Gomplate) (string, error) {
	// Create merged context file with app-specific config under "app" key
	mergedContextFile := filepath.Join(filepath.Dir(configFile), fmt.Sprintf(".merged-secrets.%s.tmp.yaml", appName))
	defer os.Remove(mergedContextFile)

	// Load root config
	rootData, err := os.ReadFile(configFile)
	if err != nil {
		return "", fmt.Errorf("failed to read config file: %w", err)
	}

	var rootConfig map[string]interface{}
	if err := yaml.Unmarshal(rootData, &rootConfig); err != nil {
		return "", fmt.Errorf("failed to parse config: %w", err)
	}

	// Extract app-specific config and add it under "app" key
	if apps, ok := rootConfig["apps"].(map[string]interface{}); ok {
		if appConfig, ok := apps[appName].(map[string]interface{}); ok {
			rootConfig["app"] = appConfig
		}
	}

	// Load secrets and extract app-specific secrets under "secrets" key
	secretsData, err := os.ReadFile(secretsFile)
	if err != nil {
		return "", fmt.Errorf("failed to read secrets file: %w", err)
	}

	var allSecrets map[string]interface{}
	if err := yaml.Unmarshal(secretsData, &allSecrets); err != nil {
		return "", fmt.Errorf("failed to parse secrets: %w", err)
	}

	// Extract app-specific secrets and add them under "secrets" key
	if apps, ok := allSecrets["apps"].(map[string]interface{}); ok {
		if appSecrets, ok := apps[appName].(map[string]interface{}); ok {
			rootConfig["secrets"] = appSecrets
		}
	}

	// Write merged config
	mergedYAML, err := yaml.Marshal(rootConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal merged config: %w", err)
	}
	if err := storage.WriteFile(mergedContextFile, mergedYAML, 0644); err != nil {
		return "", fmt.Errorf("failed to write merged config: %w", err)
	}

	args := []string{
		"-i", template,
		"-c", fmt.Sprintf(".=%s", mergedContextFile),
	}

	compiled, err := gomplate.Exec(args...)
	if err != nil {
		return "", fmt.Errorf("failed to process template: %w", err)
	}

	return strings.TrimSpace(compiled), nil
}

// setNestedConfig recursively sets nested configuration values using yq
func setNestedConfig(yq *tools.YQ, configFile, basePath string, value interface{}) error {
	switch v := value.(type) {
	case map[string]interface{}:
		// Handle nested map - recursively set each field
		for k, nested := range v {
			nestedPath := fmt.Sprintf("%s.%s", basePath, k)
			if err := setNestedConfig(yq, configFile, nestedPath, nested); err != nil {
				return err
			}
		}
	case map[interface{}]interface{}:
		// YAML unmarshaling sometimes produces this type
		for k, nested := range v {
			key := fmt.Sprintf("%v", k)
			nestedPath := fmt.Sprintf("%s.%s", basePath, key)
			if err := setNestedConfig(yq, configFile, nestedPath, nested); err != nil {
				return err
			}
		}
	default:
		// Primitive value - set it directly
		return yq.Set(configFile, basePath, fmt.Sprintf("%v", value))
	}
	return nil
}

// Add adds an app to the instance configuration
func (m *Manager) Add(instanceName, appName string, config map[string]interface{}, requiredAppMappings map[string]string) error {
	// 1. Verify app exists
	manifestPath := filepath.Join(m.appsDir, appName, "manifest.yaml")
	if !storage.FileExists(manifestPath) {
		return fmt.Errorf("app %s not found at %s", appName, manifestPath)
	}

	instancePath := tools.GetInstancePath(m.dataDir, instanceName)
	configFile := tools.GetInstanceConfigPath(m.dataDir, instanceName)
	secretsFile := tools.GetInstanceSecretsPath(m.dataDir, instanceName)
	appDestDir := filepath.Join(instancePath, "apps", appName)

	// Check instance config exists
	if !storage.FileExists(configFile) {
		return fmt.Errorf("instance config not found: %s", instanceName)
	}

	// Create app directory structure
	if err := storage.EnsureDir(appDestDir, 0755); err != nil {
		return fmt.Errorf("failed to create app directory: %w", err)
	}

	// Create .package directory for source preservation
	packageDir := filepath.Join(appDestDir, ".package")
	if err := storage.EnsureDir(packageDir, 0755); err != nil {
		return fmt.Errorf("failed to create package directory: %w", err)
	}

	// 2. Parse manifest
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest AppManifest
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

		// Process config in order from manifest YAML to handle {{ .app.X }} references correctly
		// Use the source manifest since the destination hasn't been copied yet
		sourceManifestPath := filepath.Join(m.appsDir, appName, "manifest.yaml")
		if err := processConfigInOrder(sourceManifestPath, appName, configFile); err != nil {
			return fmt.Errorf("failed to process config in order: %w", err)
		}

		// Apply user-provided config overrides (process templates first)
		if len(config) > 0 {
			gomplate := tools.NewGomplate()
			processedConfig, err := processUserConfig(config, appName, configFile, gomplate)
			if err != nil {
				return fmt.Errorf("failed to process user config: %w", err)
			}

			for key, value := range processedConfig {
				keyPath := fmt.Sprintf(".apps.%s.%s", appName, key)
				// Use setNestedConfig to handle both simple values and nested objects
				if err := setNestedConfig(yq, configFile, keyPath, value); err != nil {
					return fmt.Errorf("failed to set config %s: %w", key, err)
				}
			}
		}

		return nil
	}); err != nil {
		return err
	}

	// 4. Generate required secrets
	// Process secrets sequentially so later ones can reference earlier ones
	secretsMgr := secrets.NewManager()
	gomplate := tools.NewGomplate()

	for _, secretDef := range manifest.DefaultSecrets {
		// Build the full secret path: apps.<appName>.<key>
		secretPath := fmt.Sprintf("apps.%s.%s", appName, secretDef.Key)

		// Check if secret already exists
		existingSecret, _ := secretsMgr.GetSecret(secretsFile, secretPath)
		if existingSecret != "" && existingSecret != "null" {
			// Secret already exists, don't overwrite
			continue
		}

		var secretValue string

		if secretDef.Default != "" {
			// Process the default value using gomplate
			if strings.Contains(secretDef.Default, "{{") {
				compiled, err := processSecretTemplate(secretDef.Default, appName, configFile, secretsFile, gomplate)
				if err != nil {
					return fmt.Errorf("failed to compile secret template for %s: %w", secretDef.Key, err)
				}
				secretValue = compiled
			} else {
				secretValue = secretDef.Default
			}
		} else {
			// No default specified, generate random secret
			var err error
			secretValue, err = secrets.GenerateSecret(secrets.DefaultSecretLength)
			if err != nil {
				return fmt.Errorf("failed to generate secret for %s: %w", secretDef.Key, err)
			}
		}

		// Set the secret value with full path
		if err := secretsMgr.SetSecret(secretsFile, secretPath, secretValue); err != nil {
			return fmt.Errorf("failed to set secret %s: %w", secretPath, err)
		}
	}

	// 5. Copy source files to .package directory first
	sourceAppDir := filepath.Join(m.appsDir, appName)
	entries, err := os.ReadDir(sourceAppDir)
	if err != nil {
		return fmt.Errorf("failed to read app directory: %w", err)
	}

	// Copy all source files to .package directory
	for _, entry := range entries {
		sourcePath := filepath.Join(sourceAppDir, entry.Name())
		packagePath := filepath.Join(packageDir, entry.Name())

		if entry.IsDir() {
			// TODO: Handle subdirectories if needed
			continue
		}

		// Copy file to .package
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", entry.Name(), err)
		}
		if err := storage.WriteFile(packagePath, data, 0644); err != nil {
			return fmt.Errorf("failed to write to package %s: %w", entry.Name(), err)
		}
	}

	// Update manifest with source information and required app mappings
	manifestDestPath := filepath.Join(appDestDir, "manifest.yaml")
	if manifest.Source == "" {
		// Construct source URI based on appsDir location
		sourceAppDir := filepath.Join(m.appsDir, appName)
		absPath, err := filepath.Abs(sourceAppDir)
		if err != nil {
			absPath = sourceAppDir
		}
		manifest.Source = fmt.Sprintf("file://%s", absPath)
	}

	// Note: The 'is' field should already be set in the manifest from the directory
	// It tracks the original app type and is required in all manifests

	// Update Requires field with installedAs values from requiredAppMappings
	if len(requiredAppMappings) > 0 && len(manifest.Requires) > 0 {
		for i := range manifest.Requires {
			// Use alias as key, or name if no alias
			key := manifest.Requires[i].Name
			if manifest.Requires[i].Alias != "" {
				key = manifest.Requires[i].Alias
			}

			// Set installedAs from the mapping
			if installedAs, ok := requiredAppMappings[key]; ok {
				manifest.Requires[i].InstalledAs = installedAs
			}
		}
	}

	// Save updated manifest
	manifestYAML, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	if err := storage.WriteFile(manifestDestPath, manifestYAML, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	// 6. Compile app files from .package to app directory
	gomplate = tools.NewGomplate()

	// Extract app's config section to temp files for gomplate

	// Create temp config file with just app's section
	tempConfigFile := filepath.Join(appDestDir, ".config.tmp.yaml")
	appConfigYAML, _ := yq.Get(configFile, fmt.Sprintf(".apps.%s", appName))
	if appConfigYAML != "" && appConfigYAML != "null" {
		if err := storage.WriteFile(tempConfigFile, []byte(appConfigYAML), 0644); err != nil {
			return fmt.Errorf("failed to write temp config: %w", err)
		}
		defer os.Remove(tempConfigFile)
	}

	// Create temp secrets file with just app's section
	tempSecretsFile := filepath.Join(appDestDir, ".secrets.tmp.yaml")
	appSecretsYAML, err := yq.Get(secretsFile, fmt.Sprintf(".apps.%s", appName))
	if err == nil && appSecretsYAML != "" && appSecretsYAML != "null" {
		if err := storage.WriteFile(tempSecretsFile, []byte(appSecretsYAML), 0644); err != nil {
			return fmt.Errorf("failed to write temp secrets: %w", err)
		}
		defer os.Remove(tempSecretsFile)
	} else {
		// Write empty secrets file to avoid gomplate errors
		if err := storage.WriteFile(tempSecretsFile, []byte("apps: {}"), 0644); err != nil {
			return fmt.Errorf("failed to write empty temp secrets: %w", err)
		}
		defer os.Remove(tempSecretsFile)
	}

	// Create context for template processing - now using temp files with app's config at root
	context := map[string]string{
		".":       tempConfigFile,
		"secrets": tempSecretsFile,
	}

	// Re-read entries from .package directory for compilation
	packageEntries, err := os.ReadDir(packageDir)
	if err != nil {
		return fmt.Errorf("failed to read package directory: %w", err)
	}

	for _, entry := range packageEntries {
		if entry.IsDir() {
			// TODO: Handle subdirectories if needed
			continue
		}

		sourcePath := filepath.Join(packageDir, entry.Name())
		destPath := filepath.Join(appDestDir, entry.Name())

		// Skip processing manifest.yaml - already saved with source info
		if entry.Name() == "manifest.yaml" {
			continue
		}

		// Process other files with gomplate
		if err := gomplate.RenderWithContext(sourcePath, destPath, context); err != nil {
			return fmt.Errorf("failed to compile %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// Deploy deploys an app to the cluster
func (m *Manager) Deploy(instanceName, appName string) error {
	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)
	instancePath := tools.GetInstancePath(m.dataDir, instanceName)
	secretsFile := tools.GetInstanceSecretsPath(m.dataDir, instanceName)

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

	// Check if this app needs postgres-secrets (for db-init jobs)
	// Look for db-init-job.yaml that references postgres-secrets
	dbInitJobPath := filepath.Join(appDir, "db-init-job.yaml")
	if storage.FileExists(dbInitJobPath) {
		dbInitContent, _ := os.ReadFile(dbInitJobPath)
		if bytes.Contains(dbInitContent, []byte("postgres-secrets")) {
			// Copy postgres-secrets from postgres namespace to app namespace
			getSecretCmd := exec.Command("kubectl", "get", "secret", "postgres-secrets",
				"-n", "postgres", "-o", "yaml")
			tools.WithKubeconfig(getSecretCmd, kubeconfigPath)
			secretYaml, err := getSecretCmd.CombinedOutput()
			if err == nil {
				// Replace namespace in the YAML
				secretYaml = bytes.ReplaceAll(secretYaml, []byte("namespace: postgres"),
					[]byte(fmt.Sprintf("namespace: %s", appName)))

				// Apply the secret to the app namespace
				applySecretCmd := exec.Command("kubectl", "apply", "-f", "-")
				applySecretCmd.Stdin = bytes.NewReader(secretYaml)
				tools.WithKubeconfig(applySecretCmd, kubeconfigPath)
				_, _ = applySecretCmd.CombinedOutput() // Ignore errors if secret already exists
			}
		}
	}

	// Check if this app has an ingress with TLS configuration
	// and copy wildcard certificates from cert-manager namespace
	ingressPath := filepath.Join(appDir, "ingress.yaml")
	if storage.FileExists(ingressPath) {
		ingressContent, _ := os.ReadFile(ingressPath)
		// Check for wildcard TLS secrets that need to be copied from cert-manager
		wildcardSecrets := []string{"wildcard-wild-cloud-tls", "wildcard-internal-wild-cloud-tls"}
		for _, secretName := range wildcardSecrets {
			if bytes.Contains(ingressContent, []byte(secretName)) {
				// Copy the wildcard certificate from cert-manager namespace
				if err := utilities.CopySecretBetweenNamespaces(kubeconfigPath, secretName, "cert-manager", appName); err != nil {
					// Log error but don't fail deployment - TLS will use default cert
					// This is non-fatal as the app will still work, just without proper TLS
					fmt.Printf("Warning: Failed to copy TLS secret %s: %v\n", secretName, err)
				}
			}
		}
	}

	// Load app manifest to check for requiredSecrets
	manifestPath := filepath.Join(appDir, "manifest.yaml")
	var manifest AppManifest
	if storage.FileExists(manifestPath) {
		manifestData, err := os.ReadFile(manifestPath)
		if err == nil {
			yaml.Unmarshal(manifestData, &manifest)
		}
	}

	// Create Kubernetes secrets from secrets.yaml
	if storage.FileExists(secretsFile) {
		secretsMgr := secrets.NewManager()

		// Delete existing secret if it exists (to update it)
		deleteCmd := exec.Command("kubectl", "delete", "secret", fmt.Sprintf("%s-secrets", appName), "-n", appName, "--ignore-not-found")
		tools.WithKubeconfig(deleteCmd, kubeconfigPath)
		_, _ = deleteCmd.CombinedOutput()

		// Create secret from literals
		createSecretCmd := exec.Command("kubectl", "create", "secret", "generic", fmt.Sprintf("%s-secrets", appName), "-n", appName)

		// First, add app's own secrets from defaultSecrets
		if len(manifest.DefaultSecrets) > 0 {
			for _, secretDef := range manifest.DefaultSecrets {
				// Get the secret value from secrets.yaml
				secretPath := fmt.Sprintf("apps.%s.%s", appName, secretDef.Key)
				secretValue, err := secretsMgr.GetSecret(secretsFile, secretPath)
				if err == nil && secretValue != "" && secretValue != "null" {
					// Add to Kubernetes secret with just the key name (not the full path)
					createSecretCmd.Args = append(createSecretCmd.Args, fmt.Sprintf("--from-literal=%s=%s", secretDef.Key, secretValue))
				}
			}
		}

		// Add required secrets from dependencies
		if len(manifest.RequiredSecrets) > 0 {
			for _, requiredSecret := range manifest.RequiredSecrets {
				// requiredSecret is in format "app-ref.key" (e.g., "postgres.password")
				// The value should be looked up from secrets.yaml at apps.<actual-app-name>.<key>
				// And added to the Kubernetes secret with the key as "app-ref.key"

				// Get the secret value from secrets.yaml
				secretPath := fmt.Sprintf("apps.%s", requiredSecret)
				secretValue, err := secretsMgr.GetSecret(secretsFile, secretPath)
				if err != nil || secretValue == "" || secretValue == "null" {
					// Try without "apps." prefix for backwards compatibility
					secretValue, _ = secretsMgr.GetSecret(secretsFile, requiredSecret)
				}

				if secretValue != "" && secretValue != "null" {
					// Add to Kubernetes secret with the requiredSecret as the key
					createSecretCmd.Args = append(createSecretCmd.Args,
						fmt.Sprintf("--from-literal=%s=%s", requiredSecret, secretValue))
				}
			}
		}

		// Only create secret if we have any secrets to add
		if len(createSecretCmd.Args) > 5 { // base command has 5 args
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
	instancePath := tools.GetInstancePath(m.dataDir, instanceName)
	configFile := tools.GetInstanceConfigPath(m.dataDir, instanceName)
	secretsFile := tools.GetInstanceSecretsPath(m.dataDir, instanceName)

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
	instancePath := tools.GetInstancePath(m.dataDir, instanceName)
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

// GetEnhanced returns enhanced app information with runtime status
func (m *Manager) GetEnhanced(instanceName, appName string) (*EnhancedApp, error) {
	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)
	instancePath := tools.GetInstancePath(m.dataDir, instanceName)
	configFile := tools.GetInstanceConfigPath(m.dataDir, instanceName)
	appDir := filepath.Join(instancePath, "apps", appName)

	enhanced := &EnhancedApp{
		Name:      appName,
		Status:    "not-added",
		Namespace: appName,
	}

	// Check if app was added to instance
	if !storage.FileExists(appDir) {
		return enhanced, nil
	}

	enhanced.Status = "not-deployed"

	// Load manifest
	manifestPath := filepath.Join(appDir, "manifest.yaml")
	if storage.FileExists(manifestPath) {
		manifestData, _ := os.ReadFile(manifestPath)
		var manifest AppManifest
		if yaml.Unmarshal(manifestData, &manifest) == nil {
			enhanced.Version = manifest.Version
			enhanced.Description = manifest.Description
			enhanced.Icon = manifest.Icon
			enhanced.Manifest = &manifest
		}
	}

	// Note: README content is now served via dedicated /readme endpoint
	// No need to populate readme/documentation fields here

	// Load config
	yq := tools.NewYQ()
	configJSON, err := yq.Get(configFile, fmt.Sprintf(".apps.%s | @json", appName))
	if err == nil && configJSON != "" && configJSON != "null" {
		var config map[string]interface{}
		if json.Unmarshal([]byte(configJSON), &config) == nil {
			enhanced.Config = config
		}
	}

	// Check if namespace exists
	checkNsCmd := exec.Command("kubectl", "get", "namespace", appName, "-o", "json")
	tools.WithKubeconfig(checkNsCmd, kubeconfigPath)
	nsOutput, err := checkNsCmd.CombinedOutput()
	if err != nil {
		// Namespace doesn't exist - not deployed
		return enhanced, nil
	}

	// Parse namespace to check if it's active
	var ns struct {
		Status struct {
			Phase string `json:"phase"`
		} `json:"status"`
	}
	if err := json.Unmarshal(nsOutput, &ns); err != nil || ns.Status.Phase != "Active" {
		return enhanced, nil
	}

	enhanced.Status = "deployed"

	// Get URL (ingress)
	enhanced.URL = m.getAppURL(kubeconfigPath, appName)

	// Get runtime status
	runtime, err := m.getRuntimeStatus(kubeconfigPath, appName)
	if err == nil {
		enhanced.Runtime = runtime

		// Update status based on runtime
		if runtime.Pods != nil && len(runtime.Pods) > 0 {
			allRunning := true
			allReady := true
			for _, pod := range runtime.Pods {
				if pod.Status != "Running" {
					allRunning = false
				}
				// Check ready ratio
				parts := strings.Split(pod.Ready, "/")
				if len(parts) == 2 && parts[0] != parts[1] {
					allReady = false
				}
			}

			if allRunning && allReady {
				enhanced.Status = "running"
			} else if allRunning {
				enhanced.Status = "starting"
			} else {
				enhanced.Status = "unhealthy"
			}
		}
	}

	return enhanced, nil
}

// GetEnhancedStatus returns just the runtime status for an app
func (m *Manager) GetEnhancedStatus(instanceName, appName string) (*RuntimeStatus, error) {
	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)

	// Check if namespace exists
	checkNsCmd := exec.Command("kubectl", "get", "namespace", appName, "-o", "json")
	tools.WithKubeconfig(checkNsCmd, kubeconfigPath)
	if err := checkNsCmd.Run(); err != nil {
		return nil, fmt.Errorf("namespace not found or not deployed")
	}

	return m.getRuntimeStatus(kubeconfigPath, appName)
}

// GetAppManifest reads and parses the manifest.yaml for an app from the apps directory
func (m *Manager) GetAppManifest(appName string) (*AppManifest, error) {
	if m.appsDir == "" {
		return nil, fmt.Errorf("apps directory not configured")
	}

	manifestPath := filepath.Join(m.appsDir, appName, "manifest.yaml")
	if !storage.FileExists(manifestPath) {
		return nil, fmt.Errorf("manifest not found for app %s", appName)
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest AppManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &manifest, nil
}

// getRuntimeStatus fetches runtime information from kubernetes
func (m *Manager) getRuntimeStatus(kubeconfigPath, namespace string) (*RuntimeStatus, error) {
	kubectl := tools.NewKubectl(kubeconfigPath)

	runtime := &RuntimeStatus{}

	// Get pods (with detailed info for app status display)
	pods, err := kubectl.GetPods(namespace, true)
	if err == nil {
		runtime.Pods = pods
	}

	// Get replicas
	replicas, err := kubectl.GetReplicas(namespace)
	if err == nil && (replicas.Desired > 0 || replicas.Current > 0) {
		runtime.Replicas = replicas
	}

	// Get resources
	resources, err := kubectl.GetResources(namespace)
	if err == nil {
		runtime.Resources = resources
	}

	// Get recent events (last 10)
	events, err := kubectl.GetRecentEvents(namespace, 10)
	if err == nil {
		runtime.RecentEvents = events
	}

	return runtime, nil
}

// Update updates an app from its source package
func (m *Manager) Update(instanceName, appName string) error {
	instancePath := tools.GetInstancePath(m.dataDir, instanceName)
	appDestDir := filepath.Join(instancePath, "apps", appName)
	packageDir := filepath.Join(appDestDir, ".package")

	// Check if .package exists (if not, it's a custom app)
	if !storage.FileExists(packageDir) {
		return fmt.Errorf("app %s is custom or not installed (no package source)", appName)
	}

	// Read manifest to get source
	manifestPath := filepath.Join(appDestDir, "manifest.yaml")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest AppManifest
	if err := yaml.Unmarshal(manifestData, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Parse source URI (e.g., "file:///path/to/wild-directory/postgres")
	sourceParts := strings.Split(manifest.Source, "://")
	if len(sourceParts) != 2 {
		return fmt.Errorf("invalid source format: %s", manifest.Source)
	}

	var sourceAppDir string
	switch sourceParts[0] {
	case "file":
		// Local filesystem path
		sourceAppDir = sourceParts[1]
	case "git+https", "git+http", "git+ssh":
		// Git repository - not yet implemented
		return fmt.Errorf("git source not yet supported: %s", manifest.Source)
	default:
		return fmt.Errorf("unsupported source protocol: %s", sourceParts[0])
	}

	// Copy new version to temp directory
	tempDir := filepath.Join(appDestDir, ".package.new")
	if err := storage.EnsureDir(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir) // Clean up temp dir

	// Copy from source
	entries, err := os.ReadDir(sourceAppDir)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		sourcePath := filepath.Join(sourceAppDir, entry.Name())
		tempPath := filepath.Join(tempDir, entry.Name())

		data, err := os.ReadFile(sourcePath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", entry.Name(), err)
		}
		if err := storage.WriteFile(tempPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", entry.Name(), err)
		}
	}

	// Replace .package with new version
	if err := os.RemoveAll(packageDir); err != nil {
		return fmt.Errorf("failed to remove old package: %w", err)
	}
	if err := os.Rename(tempDir, packageDir); err != nil {
		return fmt.Errorf("failed to update package: %w", err)
	}

	// Re-compile from .package to app dir
	configFile := tools.GetInstanceConfigPath(m.dataDir, instanceName)
	secretsFile := tools.GetInstanceSecretsPath(m.dataDir, instanceName)
	gomplate := tools.NewGomplate()
	yq := tools.NewYQ()

	// Extract app's config section to temp files
	tempConfigFile := filepath.Join(appDestDir, ".config.tmp.yaml")
	appConfigYAML, err := yq.Get(configFile, fmt.Sprintf(".apps.%s", appName))
	if err == nil && appConfigYAML != "" && appConfigYAML != "null" {
		if err := storage.WriteFile(tempConfigFile, []byte(appConfigYAML), 0644); err != nil {
			return fmt.Errorf("failed to write temp config: %w", err)
		}
		defer os.Remove(tempConfigFile)
	}

	tempSecretsFile := filepath.Join(appDestDir, ".secrets.tmp.yaml")
	appSecretsYAML, err := yq.Get(secretsFile, fmt.Sprintf(".apps.%s", appName))
	if err == nil && appSecretsYAML != "" && appSecretsYAML != "null" {
		if err := storage.WriteFile(tempSecretsFile, []byte(appSecretsYAML), 0644); err != nil {
			return fmt.Errorf("failed to write temp secrets: %w", err)
		}
		defer os.Remove(tempSecretsFile)
	}

	// Create context for template processing
	context := map[string]string{
		".":       tempConfigFile,
		"secrets": tempSecretsFile,
	}

	// Re-read entries from .package directory for compilation
	packageEntries, err := os.ReadDir(packageDir)
	if err != nil {
		return fmt.Errorf("failed to read package directory: %w", err)
	}

	for _, entry := range packageEntries {
		if entry.IsDir() || entry.Name() == "manifest.yaml" {
			continue
		}

		sourcePath := filepath.Join(packageDir, entry.Name())
		destPath := filepath.Join(appDestDir, entry.Name())

		// Process files with gomplate
		if err := gomplate.RenderWithContext(sourcePath, destPath, context); err != nil {
			return fmt.Errorf("failed to compile %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// Eject converts an app from package-managed to custom
func (m *Manager) Eject(instanceName, appName string) error {
	instancePath := tools.GetInstancePath(m.dataDir, instanceName)
	appDestDir := filepath.Join(instancePath, "apps", appName)
	packageDir := filepath.Join(appDestDir, ".package")

	// Remove .package directory
	if err := os.RemoveAll(packageDir); err != nil {
		return fmt.Errorf("failed to remove package directory: %w", err)
	}

	// Update manifest to remove source
	manifestPath := filepath.Join(appDestDir, "manifest.yaml")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest AppManifest
	if err := yaml.Unmarshal(manifestData, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Remove source field
	manifest.Source = ""

	// Save updated manifest
	manifestYAML, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	if err := storage.WriteFile(manifestPath, manifestYAML, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// UpdateConfig updates an app's configuration and recompiles if needed
func (m *Manager) UpdateConfig(instanceName, appName string, config map[string]interface{}) error {
	instancePath := tools.GetInstancePath(m.dataDir, instanceName)
	configFile := tools.GetInstanceConfigPath(m.dataDir, instanceName)
	secretsFile := tools.GetInstanceSecretsPath(m.dataDir, instanceName)
	appDestDir := filepath.Join(instancePath, "apps", appName)
	packageDir := filepath.Join(appDestDir, ".package")

	// Update config
	yq := tools.NewYQ()
	configLock := configFile + ".lock"

	if err := storage.WithLock(configLock, func() error {
		for key, value := range config {
			keyPath := fmt.Sprintf(".apps.%s.%s", appName, key)
			// Use setNestedConfig to handle both simple values and nested objects
			if err := setNestedConfig(yq, configFile, keyPath, value); err != nil {
				return fmt.Errorf("failed to set config %s: %w", key, err)
			}
		}
		return nil
	}); err != nil {
		return err
	}

	// Re-compile if app has .package
	if storage.FileExists(packageDir) {
		gomplate := tools.NewGomplate()
		yq := tools.NewYQ()

		// Extract app's config section to temp files
		tempConfigFile := filepath.Join(appDestDir, ".config.tmp.yaml")
		appConfigYAML, err := yq.Get(configFile, fmt.Sprintf(".apps.%s", appName))
		if err == nil && appConfigYAML != "" && appConfigYAML != "null" {
			if err := storage.WriteFile(tempConfigFile, []byte(appConfigYAML), 0644); err != nil {
				return fmt.Errorf("failed to write temp config: %w", err)
			}
			defer os.Remove(tempConfigFile)
		}

		tempSecretsFile := filepath.Join(appDestDir, ".secrets.tmp.yaml")
		appSecretsYAML, err := yq.Get(secretsFile, fmt.Sprintf(".apps.%s", appName))
		if err == nil && appSecretsYAML != "" && appSecretsYAML != "null" {
			if err := storage.WriteFile(tempSecretsFile, []byte(appSecretsYAML), 0644); err != nil {
				return fmt.Errorf("failed to write temp secrets: %w", err)
			}
			defer os.Remove(tempSecretsFile)
		}

		// Create context for template processing
		context := map[string]string{
			".":       tempConfigFile,
			"secrets": tempSecretsFile,
		}

		// Re-read entries from .package directory for compilation
		packageEntries, err := os.ReadDir(packageDir)
		if err != nil {
			return fmt.Errorf("failed to read package directory: %w", err)
		}

		for _, entry := range packageEntries {
			if entry.IsDir() || entry.Name() == "manifest.yaml" {
				continue
			}

			sourcePath := filepath.Join(packageDir, entry.Name())
			destPath := filepath.Join(appDestDir, entry.Name())

			// Process files with gomplate
			if err := gomplate.RenderWithContext(sourcePath, destPath, context); err != nil {
				return fmt.Errorf("failed to compile %s: %w", entry.Name(), err)
			}
		}
	}

	return nil
}

// getAppURL extracts the ingress URL for an app
func (m *Manager) getAppURL(kubeconfigPath, appName string) string {
	// Try Traefik IngressRoute first
	ingressCmd := exec.Command("kubectl", "get", "ingressroute", "-n", appName, "-o", "json")
	tools.WithKubeconfig(ingressCmd, kubeconfigPath)
	ingressOutput, err := ingressCmd.CombinedOutput()

	if err == nil {
		var ingressList struct {
			Items []struct {
				Spec struct {
					Routes []struct {
						Match string `json:"match"`
					} `json:"routes"`
				} `json:"spec"`
			} `json:"items"`
		}
		if json.Unmarshal(ingressOutput, &ingressList) == nil && len(ingressList.Items) > 0 {
			if len(ingressList.Items[0].Spec.Routes) > 0 {
				match := ingressList.Items[0].Spec.Routes[0].Match
				// Parse Host(`domain.com`) format
				if strings.Contains(match, "Host(`") {
					start := strings.Index(match, "Host(`") + 6
					end := strings.Index(match[start:], "`")
					if end > 0 {
						host := match[start : start+end]
						return "https://" + host
					}
				}
			}
		}
	}

	// If no IngressRoute, try standard Ingress
	ingressCmd = exec.Command("kubectl", "get", "ingress", "-n", appName, "-o", "json")
	tools.WithKubeconfig(ingressCmd, kubeconfigPath)
	ingressOutput, err = ingressCmd.CombinedOutput()

	if err == nil {
		var ingressList struct {
			Items []struct {
				Spec struct {
					Rules []struct {
						Host string `json:"host"`
					} `json:"rules"`
				} `json:"spec"`
			} `json:"items"`
		}
		if json.Unmarshal(ingressOutput, &ingressList) == nil && len(ingressList.Items) > 0 {
			if len(ingressList.Items[0].Spec.Rules) > 0 {
				host := ingressList.Items[0].Spec.Rules[0].Host
				if host != "" {
					return "https://" + host
				}
			}
		}
	}

	return ""
}

// processUserConfig processes user-provided config values, compiling any templates
// Reuses existing processValueNode logic by converting to YAML and back
func processUserConfig(config map[string]interface{}, appName, configFile string, gomplate *tools.Gomplate) (map[string]interface{}, error) {
	// Convert map to YAML bytes
	configYAML, err := yaml.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	// Parse into yaml.Node to use existing processValueNode
	var node yaml.Node
	if err := yaml.Unmarshal(configYAML, &node); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Process using existing template compilation logic
	// Note: processValueNode expects the root mapping node's content
	if node.Kind != yaml.DocumentNode || len(node.Content) == 0 {
		return config, nil
	}

	processed, err := processValueNode(node.Content[0], appName, configFile, nil, gomplate)
	if err != nil {
		return nil, err
	}

	// Convert result back to map
	result, ok := processed.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected result type from processValueNode: %T", processed)
	}

	return result, nil
}

// processConfigInOrder processes config keys in the order they appear in the manifest YAML
// processValueNode recursively processes a yaml.Node value, compiling templates in all scalar (leaf) nodes
func processValueNode(node *yaml.Node, appName, configFile string, appContext map[string]interface{}, gomplate *tools.Gomplate) (interface{}, error) {
	switch node.Kind {
	case yaml.ScalarNode:
		value := node.Value
		// Process templates if value contains {{
		if strings.Contains(value, "{{") {
			// Create merged context file
			mergedContextFile := filepath.Join(filepath.Dir(configFile), fmt.Sprintf(".merged.%s.tmp.yaml", appName))
			defer os.Remove(mergedContextFile)

			// Load root config
			rootData, err := os.ReadFile(configFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read config: %w", err)
			}

			var rootConfig map[string]interface{}
			if err := yaml.Unmarshal(rootData, &rootConfig); err != nil {
				return nil, fmt.Errorf("failed to parse config: %w", err)
			}

			// Merge app context under the "app" key
			rootConfig["app"] = appContext

			// Write merged config
			mergedYAML, err := yaml.Marshal(rootConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal merged config: %w", err)
			}
			if err := storage.WriteFile(mergedContextFile, mergedYAML, 0644); err != nil {
				return nil, fmt.Errorf("failed to write merged config: %w", err)
			}

			// Process template with merged context
			args := []string{
				"-i", value,
				"-c", fmt.Sprintf(".=%s", mergedContextFile),
			}
			compiled, err := gomplate.Exec(args...)
			if err != nil {
				return nil, fmt.Errorf("failed to compile template: %w", err)
			}
			return strings.TrimSpace(compiled), nil
		}
		return value, nil

	case yaml.SequenceNode:
		// Handle arrays - process each element
		var arr []interface{}
		for _, item := range node.Content {
			processed, err := processValueNode(item, appName, configFile, appContext, gomplate)
			if err != nil {
				return nil, err
			}
			arr = append(arr, processed)
		}
		return arr, nil

	case yaml.MappingNode:
		// Handle nested objects - recursively process each key-value pair
		result := make(map[string]interface{})
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			key := keyNode.Value
			processed, err := processValueNode(valueNode, appName, configFile, appContext, gomplate)
			if err != nil {
				return nil, fmt.Errorf("failed to process nested key %s: %w", key, err)
			}
			result[key] = processed
		}
		return result, nil

	default:
		return node.Value, nil
	}
}

func processConfigInOrder(manifestPath string, appName string, configFile string) error {
	// Read the manifest file directly to preserve order
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	// Parse YAML preserving order
	var node yaml.Node
	if err := yaml.Unmarshal(manifestData, &node); err != nil {
		return fmt.Errorf("failed to parse manifest YAML: %w", err)
	}

	// Find defaultConfig node
	var defaultConfigNode *yaml.Node
	if len(node.Content) > 0 && node.Content[0].Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content[0].Content); i += 2 {
			if node.Content[0].Content[i].Value == "defaultConfig" {
				defaultConfigNode = node.Content[0].Content[i+1]
				break
			}
		}
	}

	if defaultConfigNode == nil || defaultConfigNode.Kind != yaml.MappingNode {
		return nil // No defaultConfig to process
	}

	yq := tools.NewYQ()
	gomplate := tools.NewGomplate()

	// Build up app context as we process values
	appContext := make(map[string]interface{})

	// Process each config key in order
	for i := 0; i < len(defaultConfigNode.Content); i += 2 {
		keyNode := defaultConfigNode.Content[i]
		valueNode := defaultConfigNode.Content[i+1]

		key := keyNode.Value
		keyPath := fmt.Sprintf(".apps.%s.%s", appName, key)

		// Check if already exists
		existing, _ := yq.Get(configFile, keyPath)
		if existing != "" && existing != "null" {
			// Parse the existing value and add to context for later references
			var existingValue interface{}
			if err := yaml.Unmarshal([]byte(existing), &existingValue); err == nil {
				appContext[key] = existingValue
			} else {
				appContext[key] = existing
			}
			continue // Skip existing values
		}

		// Recursively process the value node, compiling templates in all leaf nodes
		value, err := processValueNode(valueNode, appName, configFile, appContext, gomplate)
		if err != nil {
			return fmt.Errorf("failed to process config key %s: %w", key, err)
		}

		// Add processed value to context for future references
		appContext[key] = value

		// Set the config value in the actual config file
		if err := setNestedConfig(yq, configFile, keyPath, value); err != nil {
			return fmt.Errorf("failed to set config %s: %w", key, err)
		}
	}

	return nil
}


