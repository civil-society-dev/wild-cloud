package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/wild-cloud/wild-central/daemon/internal/operations"
	"github.com/wild-cloud/wild-central/daemon/internal/storage"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
)

// Manager handles base service operations
type Manager struct {
	dataDir     string
	servicesDir string                      // Path to services directory
	manifests   map[string]*ServiceManifest // Cached service manifests
}

// NewManager creates a new services manager
func NewManager(dataDir, servicesDir string) *Manager {
	m := &Manager{
		dataDir:     dataDir,
		servicesDir: servicesDir,
	}

	// Load all service manifests
	manifests, err := LoadAllManifests(servicesDir)
	if err != nil {
		// Log error but continue - services without manifests will fall back to hardcoded map
		fmt.Printf("Warning: failed to load service manifests: %v\n", err)
		manifests = make(map[string]*ServiceManifest)
	}
	m.manifests = manifests

	return m
}

// Service represents a base service
type Service struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Version     string   `json:"version"`
	Namespace   string   `json:"namespace"`
	Dependencies []string `json:"dependencies,omitempty"`
}

// Base services in Wild Cloud (kept for reference/validation)
var BaseServices = []string{
	"metallb",      // Load balancer
	"traefik",      // Ingress controller
	"cert-manager", // Certificate management
	"longhorn",     // Storage
}

// serviceDeployments maps service directory names to their actual namespace and deployment name
var serviceDeployments = map[string]struct {
	namespace      string
	deploymentName string
}{
	"cert-manager":           {"cert-manager", "cert-manager"},
	"coredns":                {"kube-system", "coredns"},
	"docker-registry":        {"docker-registry", "docker-registry"},
	"externaldns":            {"externaldns", "external-dns"},
	"kubernetes-dashboard":   {"kubernetes-dashboard", "kubernetes-dashboard"},
	"longhorn":               {"longhorn-system", "longhorn-ui"},
	"metallb":                {"metallb-system", "controller"},
	"nfs":                    {"nfs-system", "nfs-server"},
	"node-feature-discovery": {"node-feature-discovery", "node-feature-discovery-master"},
	"nvidia-device-plugin":   {"nvidia-device-plugin", "nvidia-device-plugin-daemonset"},
	"smtp":                   {"smtp-system", "smtp"},
	"traefik":                {"traefik", "traefik"},
	"utils":                  {"utils-system", "utils"},
}

// checkServiceStatus checks if a service is deployed
func (m *Manager) checkServiceStatus(instanceName, serviceName string) string {
	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)

	// If kubeconfig doesn't exist, cluster isn't bootstrapped
	if !storage.FileExists(kubeconfigPath) {
		return "not-deployed"
	}

	kubectl := tools.NewKubectl(kubeconfigPath)

	// Special case: NFS doesn't have a deployment, check for StorageClass instead
	if serviceName == "nfs" {
		cmd := exec.Command("kubectl", "--kubeconfig", kubeconfigPath, "get", "storageclass", "nfs", "-o", "name")
		if err := cmd.Run(); err == nil {
			return "deployed"
		}
		return "not-deployed"
	}

	var namespace, deploymentName string

	// Check hardcoded map first for deployment name (has correct names)
	if deployment, ok := serviceDeployments[serviceName]; ok {
		namespace = deployment.namespace
		deploymentName = deployment.deploymentName
	} else if manifest, ok := m.manifests[serviceName]; ok {
		// Fall back to manifest if not in hardcoded map
		namespace = manifest.Namespace
		deploymentName = manifest.GetDeploymentName()
	} else {
		// Service not found anywhere, assume not deployed
		return "not-deployed"
	}

	if kubectl.DeploymentExists(deploymentName, namespace) {
		return "deployed"
	}

	return "not-deployed"
}

// List returns all base services and their status
func (m *Manager) List(instanceName string) ([]Service, error) {
	services := []Service{}

	// Discover services from the services directory
	entries, err := os.ReadDir(m.servicesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read services directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue // Skip non-directories like README.md
		}

		name := entry.Name()

		// Get service info from manifest if available
		var namespace, description, version string
		var dependencies []string

		if manifest, ok := m.manifests[name]; ok {
			namespace = manifest.Namespace
			description = manifest.Description
			version = manifest.Category // Using category as version for now
			dependencies = manifest.Dependencies
		} else {
			// Fall back to hardcoded map
			namespace = name + "-system" // default
			if deployment, ok := serviceDeployments[name]; ok {
				namespace = deployment.namespace
			}
		}

		service := Service{
			Name:         name,
			Status:       m.checkServiceStatus(instanceName, name),
			Namespace:    namespace,
			Description:  description,
			Version:      version,
			Dependencies: dependencies,
		}

		services = append(services, service)
	}

	return services, nil
}

// Get returns a specific service
func (m *Manager) Get(instanceName, serviceName string) (*Service, error) {
	// Get the correct namespace from the map
	namespace := serviceName + "-system" // default
	if deployment, ok := serviceDeployments[serviceName]; ok {
		namespace = deployment.namespace
	}

	service := &Service{
		Name:      serviceName,
		Status:    m.checkServiceStatus(instanceName, serviceName),
		Namespace: namespace,
	}

	return service, nil
}

// Install orchestrates the complete service installation lifecycle
func (m *Manager) Install(instanceName, serviceName string, fetch, deploy bool, opID string, broadcaster *operations.Broadcaster) error {
	// Phase 1: Fetch (if requested or files don't exist)
	if fetch || !m.serviceFilesExist(instanceName, serviceName) {
		if err := m.Fetch(instanceName, serviceName); err != nil {
			return fmt.Errorf("fetch failed: %w", err)
		}
	}

	// Phase 2: Validate Configuration
	// Configuration happens via API before calling install
	// Validate all required config is set
	if err := m.validateConfig(instanceName, serviceName); err != nil {
		return fmt.Errorf("configuration incomplete: %w", err)
	}

	// Phase 3: Compile templates
	if err := m.Compile(instanceName, serviceName); err != nil {
		return fmt.Errorf("template compilation failed: %w", err)
	}

	// Phase 4: Deploy (if requested)
	if deploy {
		if err := m.Deploy(instanceName, serviceName, opID, broadcaster); err != nil {
			return fmt.Errorf("deployment failed: %w", err)
		}
	}

	return nil
}

// InstallAll installs all base services
func (m *Manager) InstallAll(instanceName string, fetch, deploy bool, opID string, broadcaster *operations.Broadcaster) error {
	for _, serviceName := range BaseServices {
		if err := m.Install(instanceName, serviceName, fetch, deploy, opID, broadcaster); err != nil {
			return fmt.Errorf("failed to install %s: %w", serviceName, err)
		}
	}

	return nil
}

// Delete removes a service
func (m *Manager) Delete(instanceName, serviceName string) error {
	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)

	serviceDir := filepath.Join(m.servicesDir, serviceName)
	manifestsFile := filepath.Join(serviceDir, "manifests.yaml")

	if !storage.FileExists(manifestsFile) {
		return fmt.Errorf("service %s not found", serviceName)
	}

	cmd := exec.Command("kubectl", "delete", "-f", manifestsFile)
	tools.WithKubeconfig(cmd, kubeconfigPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete service: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// GetStatus returns detailed status for a service
func (m *Manager) GetStatus(instanceName, serviceName string) (*Service, error) {
	// Get the correct namespace from the map
	namespace := serviceName + "-system" // default
	if deployment, ok := serviceDeployments[serviceName]; ok {
		namespace = deployment.namespace
	}

	service := &Service{
		Name:      serviceName,
		Namespace: namespace,
		Status:    m.checkServiceStatus(instanceName, serviceName),
	}

	return service, nil
}

// GetManifest returns the manifest for a service
func (m *Manager) GetManifest(serviceName string) (*ServiceManifest, error) {
	if manifest, ok := m.manifests[serviceName]; ok {
		return manifest, nil
	}
	return nil, fmt.Errorf("service %s not found or has no manifest", serviceName)
}

// GetServiceConfig returns the service configuration fields from the manifest
func (m *Manager) GetServiceConfig(serviceName string) (map[string]ConfigDefinition, error) {
	manifest, err := m.GetManifest(serviceName)
	if err != nil {
		return nil, err
	}
	return manifest.ServiceConfig, nil
}

// GetConfigReferences returns the config references from the manifest
func (m *Manager) GetConfigReferences(serviceName string) ([]string, error) {
	manifest, err := m.GetManifest(serviceName)
	if err != nil {
		return nil, err
	}
	return manifest.ConfigReferences, nil
}

// Fetch copies service files from directory to instance
func (m *Manager) Fetch(instanceName, serviceName string) error {
	// 1. Validate service exists in directory
	sourceDir := filepath.Join(m.servicesDir, serviceName)
	if !dirExists(sourceDir) {
		return fmt.Errorf("service %s not found in directory", serviceName)
	}

	// 2. Create instance service directory
	instanceDir := filepath.Join(m.dataDir, "instances", instanceName,
		"setup", "cluster-services", serviceName)
	if err := os.MkdirAll(instanceDir, 0755); err != nil {
		return fmt.Errorf("failed to create service directory: %w", err)
	}

	// 3. Copy files:
	//    - README.md (if exists, optional)
	//    - install.sh (if exists, optional)
	//    - kustomize.template/* (if exists, optional)

	// Copy README.md
	copyFileIfExists(filepath.Join(sourceDir, "README.md"),
		filepath.Join(instanceDir, "README.md"))

	// Copy install.sh (optional)
	installSh := filepath.Join(sourceDir, "install.sh")
	if fileExists(installSh) {
		if err := copyFile(installSh, filepath.Join(instanceDir, "install.sh")); err != nil {
			return fmt.Errorf("failed to copy install.sh: %w", err)
		}
		// Make install.sh executable
		os.Chmod(filepath.Join(instanceDir, "install.sh"), 0755)
	}

	// Copy kustomize.template directory if it exists
	templateDir := filepath.Join(sourceDir, "kustomize.template")
	if dirExists(templateDir) {
		destTemplateDir := filepath.Join(instanceDir, "kustomize.template")
		if err := copyDir(templateDir, destTemplateDir); err != nil {
			return fmt.Errorf("failed to copy templates: %w", err)
		}
	}

	return nil
}

// serviceFilesExist checks if service files exist in the instance
func (m *Manager) serviceFilesExist(instanceName, serviceName string) bool {
	serviceDir := filepath.Join(m.dataDir, "instances", instanceName,
		"setup", "cluster-services", serviceName)
	installSh := filepath.Join(serviceDir, "install.sh")
	return fileExists(installSh)
}

// Helper functions for file operations

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, input, 0644)
}

func copyFileIfExists(src, dst string) error {
	if !fileExists(src) {
		return nil
	}
	return copyFile(src, dst)
}

func copyDir(src, dst string) error {
	// Create destination directory
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// Compile processes gomplate templates into final Kubernetes manifests
func (m *Manager) Compile(instanceName, serviceName string) error {
	instanceDir := filepath.Join(m.dataDir, "instances", instanceName)
	serviceDir := filepath.Join(instanceDir, "setup", "cluster-services", serviceName)
	templateDir := filepath.Join(serviceDir, "kustomize.template")
	outputDir := filepath.Join(serviceDir, "kustomize")

	// 1. Check if templates exist
	if !dirExists(templateDir) {
		// No templates to compile - this is OK for some services
		return nil
	}

	// 2. Load config and secrets files
	configFile := filepath.Join(instanceDir, "config.yaml")
	secretsFile := filepath.Join(instanceDir, "secrets.yaml")

	if !fileExists(configFile) {
		return fmt.Errorf("config.yaml not found for instance %s", instanceName)
	}

	// 3. Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 4. Process templates with gomplate
	// Build gomplate command
	gomplateArgs := []string{
		"-c", fmt.Sprintf(".=%s", configFile),
	}

	// Add secrets context if file exists
	if fileExists(secretsFile) {
		gomplateArgs = append(gomplateArgs, "-c", fmt.Sprintf("secrets=%s", secretsFile))
	}

	// Process each template file recursively
	err := filepath.Walk(templateDir, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Calculate relative path and destination
		relPath, _ := filepath.Rel(templateDir, srcPath)
		dstPath := filepath.Join(outputDir, relPath)

		// Create destination directory
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return err
		}

		// Run gomplate on this file
		args := append(gomplateArgs, "-f", srcPath, "-o", dstPath)
		cmd := exec.Command("gomplate", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("gomplate failed for %s: %w\nOutput: %s", relPath, err, output)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("template compilation failed: %w", err)
	}

	return nil
}

// Deploy executes the service-specific install.sh script
// opID and broadcaster are optional - if provided, output will be streamed to SSE clients
func (m *Manager) Deploy(instanceName, serviceName, opID string, broadcaster *operations.Broadcaster) error {
	fmt.Printf("[DEBUG] Deploy() called for service=%s instance=%s opID=%s\n", serviceName, instanceName, opID)

	instanceDir := filepath.Join(m.dataDir, "instances", instanceName)
	serviceDir := filepath.Join(instanceDir, "setup", "cluster-services", serviceName)
	installScript := filepath.Join(serviceDir, "install.sh")

	// 1. Check if install.sh exists
	if !fileExists(installScript) {
		// No install.sh means nothing to deploy - this is valid for documentation-only services
		msg := fmt.Sprintf("ℹ️  Service %s has no install.sh - nothing to deploy\n", serviceName)
		if broadcaster != nil && opID != "" {
			broadcaster.Publish(opID, []byte(msg))
		}
		return nil
	}
	fmt.Printf("[DEBUG] Found install script: %s\n", installScript)

	// 2. Set up environment
	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)
	if !fileExists(kubeconfigPath) {
		return fmt.Errorf("kubeconfig not found - cluster may not be bootstrapped")
	}
	fmt.Printf("[DEBUG] Using kubeconfig: %s\n", kubeconfigPath)

	// Build environment - append to existing environment
	// This ensures kubectl and other tools are available
	env := os.Environ()
	env = append(env,
		fmt.Sprintf("WILD_INSTANCE=%s", instanceName),
		fmt.Sprintf("WILD_CENTRAL_DATA=%s", m.dataDir),
		fmt.Sprintf("KUBECONFIG=%s", kubeconfigPath),
	)
	fmt.Printf("[DEBUG] Environment configured: WILD_INSTANCE=%s, KUBECONFIG=%s\n", instanceName, kubeconfigPath)

	// 3. Set up output streaming
	var outputWriter *broadcastWriter
	if opID != "" {
		// Create log directory
		logDir := filepath.Join(instanceDir, "operations", opID)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}

		// Create log file
		logFile, err := os.Create(filepath.Join(logDir, "output.log"))
		if err != nil {
			return fmt.Errorf("failed to create log file: %w", err)
		}
		defer logFile.Close()

		// Create broadcast writer
		outputWriter = newBroadcastWriter(logFile, broadcaster, opID)

		// Send initial heartbeat message to SSE stream
		if broadcaster != nil {
			initialMsg := fmt.Sprintf("🚀 Starting deployment of %s...\n", serviceName)
			broadcaster.Publish(opID, []byte(initialMsg))
			fmt.Printf("[DEBUG] Sent initial SSE message for opID=%s\n", opID)
		}
	}

	// 4. Execute install.sh
	fmt.Printf("[DEBUG] Executing: /bin/bash %s\n", installScript)
	cmd := exec.Command("/bin/bash", installScript)
	cmd.Dir = serviceDir
	cmd.Env = env

	if outputWriter != nil {
		// Stream output to file and SSE clients
		cmd.Stdout = outputWriter
		cmd.Stderr = outputWriter
		fmt.Printf("[DEBUG] Starting command execution for opID=%s\n", opID)
		err := cmd.Run()
		fmt.Printf("[DEBUG] Command completed for opID=%s, err=%v\n", opID, err)
		if broadcaster != nil {
			outputWriter.Flush()            // Flush any remaining buffered data
			broadcaster.Close(opID)         // Close all SSE clients
		}
		return err
	} else {
		// Fallback: capture output for logging (backward compatibility)
		output, err := cmd.CombinedOutput()
		fmt.Printf("=== Deploy %s output ===\n%s\n=== End output ===\n", serviceName, output)
		if err != nil {
			return fmt.Errorf("deployment failed: %w\nOutput: %s", err, output)
		}
		return nil
	}
}

// validateConfig checks that all required config is set for a service
func (m *Manager) validateConfig(instanceName, serviceName string) error {
	manifest, err := m.GetManifest(serviceName)
	if err != nil {
		return err // Service has no manifest
	}

	// Load instance config
	instanceDir := filepath.Join(m.dataDir, "instances", instanceName)
	configFile := filepath.Join(instanceDir, "config.yaml")

	configData, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(configData, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Check all required paths exist
	missing := []string{}
	allPaths := append(manifest.ConfigReferences, manifest.GetRequiredConfig()...)

	for _, path := range allPaths {
		if getNestedValue(config, path) == nil {
			missing = append(missing, path)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration: %v", missing)
	}

	return nil
}

// getNestedValue retrieves a value from nested map using dot notation
func getNestedValue(data map[string]interface{}, path string) interface{} {
	keys := strings.Split(path, ".")
	current := data

	for i, key := range keys {
		if i == len(keys)-1 {
			return current[key]
		}

		if next, ok := current[key].(map[string]interface{}); ok {
			current = next
		} else {
			return nil
		}
	}

	return nil
}
