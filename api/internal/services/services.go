package services

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/wild-cloud/wild-central/daemon/internal/operations"
	"github.com/wild-cloud/wild-central/daemon/internal/setup"
	"github.com/wild-cloud/wild-central/daemon/internal/storage"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
)

// Manager handles base service operations
type Manager struct {
	dataDir   string
	manifests map[string]*ServiceManifest // Cached service manifests
}

// NewManager creates a new services manager
// Note: Service definitions are now loaded from embedded setup files
func NewManager(dataDir string) *Manager {
	m := &Manager{
		dataDir: dataDir,
	}

	// Load all service manifests from embedded files
	manifests := make(map[string]*ServiceManifest)
	services, err := setup.ListServices()
	if err == nil {
		for _, serviceName := range services {
			manifest, err := setup.GetManifest(serviceName)
			if err == nil {
				// Convert setup.ServiceManifest to services.ServiceManifest
				// Convert setup.ConfigDefinition map to services.ConfigDefinition map
				serviceConfig := make(map[string]ConfigDefinition)
				for key, cfg := range manifest.ServiceConfig {
					serviceConfig[key] = ConfigDefinition{
						Path:    cfg.Path,
						Prompt:  cfg.Prompt,
						Default: cfg.Default,
						Type:    cfg.Type,
					}
				}

				manifests[serviceName] = &ServiceManifest{
					Name:             manifest.Name,
					Description:      manifest.Description,
					Version:          manifest.Version,
					Namespace:        manifest.Namespace,
					DeploymentName:   manifest.DeploymentName,
					StorageClassName: manifest.StorageClassName,
					Category:         manifest.Category,
					Dependencies:     manifest.Dependencies,
					ConfigReferences: manifest.ConfigReferences,
					ServiceConfig:    serviceConfig,
				}
			}
		}
	} else {
		fmt.Printf("Warning: failed to load service manifests from embedded files: %v\n", err)
	}
	m.manifests = manifests

	return m
}

// Service represents a base service
type Service struct {
	Name             string                  `json:"name"`
	Description      string                  `json:"description"`
	Status           string                  `json:"status"` // Overall status (for backward compatibility)
	Version          string                  `json:"version"`
	Namespace        string                  `json:"namespace"`
	StorageClassName string                  `json:"storageClassName,omitempty"`
	Dependencies     []string                `json:"dependencies,omitempty"`
	HasConfig        bool                    `json:"hasConfig"` // Whether service has configurable fields
	Lifecycle        *ServiceLifecycleStatus `json:"lifecycle,omitempty"` // Enhanced lifecycle state
}

// ServiceLifecycleStatus provides detailed state information for each lifecycle phase
type ServiceLifecycleStatus struct {
	Templates     TemplateState     `json:"templates"`
	Configuration ConfigurationState `json:"configuration"`
	Deployment    DeploymentState    `json:"deployment"`
}

// TemplateState represents the fetch/template phase state
type TemplateState struct {
	State          string `json:"state"`          // "not_fetched", "cached", "up_to_date", "update_available"
	Version        string `json:"version"`        // Currently installed version
	LatestVersion  string `json:"latestVersion"`  // Latest available version
	UpdateAvailable bool   `json:"updateAvailable"`
}

// ConfigurationState represents the compile phase state
type ConfigurationState struct {
	State        string  `json:"state"`        // "compiled", "needs_recompile", "not_configured"
	Reason       string  `json:"reason,omitempty"` // "config_changed", "templates_changed", null
	LastCompiled *string `json:"lastCompiled,omitempty"` // ISO timestamp
}

// DeploymentState represents the deploy phase state
type DeploymentState struct {
	State   string               `json:"state"`   // "deployed", "not_deployed", "degraded", "out_of_sync"
	Healthy bool                 `json:"healthy"`
	Replicas *tools.DeploymentInfo `json:"replicas,omitempty"`
}

// Base services in Wild Cloud (kept for reference/validation)
var BaseServices = []string{
	"metallb",      // Load balancer
	"traefik",      // Ingress controller
	"cert-manager", // Certificate management
	"longhorn",     // Storage
}

// checkServiceStatus checks the deployment status of a service
// Returns: "not-deployed", "deployed", "degraded", or "progressing"
func (m *Manager) checkServiceStatus(instanceName, serviceName string) string {
	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)

	// If kubeconfig doesn't exist, cluster isn't bootstrapped
	if !storage.FileExists(kubeconfigPath) {
		return "not-deployed"
	}

	kubectl := tools.NewKubectl(kubeconfigPath)

	// Special case: NFS doesn't have a deployment, check for StorageClass instead
	if serviceName == "nfs" {
		cmd := exec.Command("kubectl", "get", "storageclass", "nfs", "-o", "name")
		tools.WithKubeconfig(cmd, kubeconfigPath)
		if err := cmd.Run(); err == nil {
			return "deployed"
		}
		return "not-deployed"
	}

	manifest, ok := m.manifests[serviceName]
	if !ok {
		return "not-deployed"
	}

	namespace := manifest.Namespace
	deploymentName := manifest.GetDeploymentName()

	// Try to get deployment first, then try daemonset
	deploymentInfo, err := kubectl.GetDeployment(deploymentName, namespace)
	if err != nil {
		// If deployment not found, try daemonset
		deploymentInfo, err = kubectl.GetDaemonSet(deploymentName, namespace)
		if err != nil {
			return "not-deployed"
		}
	}

	// Determine status based on replica/pod counts
	// For DaemonSets: Desired=0 is valid when no nodes match the selector
	if deploymentInfo.Desired == 0 {
		// Check if there are any current pods - if yes, it's deployed but scaled to zero
		// If no current pods and no desired pods, still considered "deployed" (waiting for matching nodes)
		return "deployed"
	}

	if deploymentInfo.Ready == deploymentInfo.Desired && deploymentInfo.Desired > 0 {
		return "deployed"
	} else if deploymentInfo.Ready < deploymentInfo.Desired {
		if deploymentInfo.Current > deploymentInfo.Desired {
			return "progressing"
		}
		return "degraded"
	}

	return "deployed"
}

// getServiceLifecycleStatus returns detailed lifecycle state for a service
func (m *Manager) getServiceLifecycleStatus(instanceName, serviceName string) *ServiceLifecycleStatus {
	return &ServiceLifecycleStatus{
		Templates:     m.checkTemplateState(instanceName, serviceName),
		Configuration: m.checkConfigurationState(instanceName, serviceName),
		Deployment:    m.checkDeploymentState(instanceName, serviceName),
	}
}

// checkTemplateState determines if templates are fetched and if updates are available
func (m *Manager) checkTemplateState(instanceName, serviceName string) TemplateState {
	manifest, ok := m.manifests[serviceName]
	if !ok {
		return TemplateState{State: "not_fetched"}
	}

	// Check if service files exist in instance directory
	instanceServiceDir := filepath.Join(tools.GetInstancePath(m.dataDir, instanceName), "setup", "cluster-services", serviceName)
	if !storage.FileExists(instanceServiceDir) {
		return TemplateState{
			State:          "not_fetched",
			LatestVersion:  manifest.Version,
			UpdateAvailable: false,
		}
	}

	// Read instance manifest to get installed version
	instanceManifestPath := filepath.Join(instanceServiceDir, "wild-manifest.yaml")

	// If manifest file doesn't exist, service needs update to get latest version
	if !storage.FileExists(instanceManifestPath) {
		return TemplateState{
			State:          "update_available",
			LatestVersion:  manifest.Version,
			UpdateAvailable: true,
		}
	}

	var instanceManifest ServiceManifest
	if data, err := os.ReadFile(instanceManifestPath); err == nil {
		if err := yaml.Unmarshal(data, &instanceManifest); err == nil {
			// Compare versions
			if instanceManifest.Version != manifest.Version {
				return TemplateState{
					State:          "update_available",
					Version:        instanceManifest.Version,
					LatestVersion:  manifest.Version,
					UpdateAvailable: true,
				}
			}
			// Versions match
			return TemplateState{
				State:          "up_to_date",
				Version:        instanceManifest.Version,
				LatestVersion:  manifest.Version,
				UpdateAvailable: false,
			}
		}
	}

	// Can't read version, assume cached
	return TemplateState{
		State:          "cached",
		Version:        manifest.Version,
		LatestVersion:  manifest.Version,
		UpdateAvailable: false,
	}
}

// checkConfigurationState determines if service needs recompilation
func (m *Manager) checkConfigurationState(instanceName, serviceName string) ConfigurationState {
	instanceServiceDir := filepath.Join(tools.GetInstancePath(m.dataDir, instanceName), "setup", "cluster-services", serviceName)

	// Check if kustomize directory exists (compiled manifests)
	kustomizeDir := filepath.Join(instanceServiceDir, "kustomize")
	if !storage.FileExists(kustomizeDir) {
		return ConfigurationState{
			State: "not_configured",
			Reason: "",
		}
	}

	// Check if kustomize.template exists
	templateDir := filepath.Join(instanceServiceDir, "kustomize.template")
	if !storage.FileExists(templateDir) {
		// No templates = always compiled
		return ConfigurationState{
			State: "compiled",
		}
	}

	// Get modification times to determine if recompile needed
	templateModTime := getDirectoryModTime(templateDir)
	kustomizeModTime := getDirectoryModTime(kustomizeDir)

	configPath := filepath.Join(tools.GetInstancePath(m.dataDir, instanceName), "config.yaml")
	configModTime := getFileModTime(configPath)

	// If templates or config changed after last compile, needs recompile
	if templateModTime.After(kustomizeModTime) {
		lastCompiled := kustomizeModTime.Format(time.RFC3339)
		return ConfigurationState{
			State:        "needs_recompile",
			Reason:       "templates_changed",
			LastCompiled: &lastCompiled,
		}
	}

	if configModTime.After(kustomizeModTime) {
		lastCompiled := kustomizeModTime.Format(time.RFC3339)
		return ConfigurationState{
			State:        "needs_recompile",
			Reason:       "config_changed",
			LastCompiled: &lastCompiled,
		}
	}

	// Up to date
	lastCompiled := kustomizeModTime.Format(time.RFC3339)
	return ConfigurationState{
		State:        "compiled",
		LastCompiled: &lastCompiled,
	}
}

// checkDeploymentState determines cluster deployment status
func (m *Manager) checkDeploymentState(instanceName, serviceName string) DeploymentState {
	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)

	// If kubeconfig doesn't exist, cluster isn't bootstrapped
	if !storage.FileExists(kubeconfigPath) {
		return DeploymentState{
			State:   "not_deployed",
			Healthy: false,
		}
	}

	kubectl := tools.NewKubectl(kubeconfigPath)
	manifest, ok := m.manifests[serviceName]
	if !ok {
		return DeploymentState{
			State:   "not_deployed",
			Healthy: false,
		}
	}

	namespace := manifest.Namespace
	deploymentName := manifest.GetDeploymentName()

	// If no deployment name specified, this is a configuration-only service (e.g., NFS StorageClass)
	// Check for StorageClass or namespace existence instead of workload
	// Note: Check the raw DeploymentName field, not GetDeploymentName(), because Get falls back to Name
	if manifest.DeploymentName == "" {
		// If storageClassName specified, check for that
		if manifest.StorageClassName != "" {
			_, err := kubectl.GetStorageClass(manifest.StorageClassName)
			if err != nil {
				return DeploymentState{
					State:   "not_deployed",
					Healthy: false,
				}
			}
			// Configuration service is deployed if storage class exists
			return DeploymentState{
				State:   "deployed",
				Healthy: true,
			}
		}

		// Otherwise check namespace existence
		namespaceInfo, err := kubectl.GetNamespace(namespace)
		if err != nil || namespaceInfo.Status.Phase != "Active" {
			return DeploymentState{
				State:   "not_deployed",
				Healthy: false,
			}
		}
		// Configuration service is deployed if namespace exists and is active
		return DeploymentState{
			State:   "deployed",
			Healthy: true,
		}
	}

	// Try to get deployment first, then try daemonset
	deploymentInfo, err := kubectl.GetDeployment(deploymentName, namespace)
	if err != nil {
		// If deployment not found, try daemonset
		deploymentInfo, err = kubectl.GetDaemonSet(deploymentName, namespace)
		if err != nil {
			return DeploymentState{
				State:   "not_deployed",
				Healthy: false,
			}
		}
	}

	// Check if compiled manifests are newer than the last deployment
	instanceServiceDir := filepath.Join(tools.GetInstancePath(m.dataDir, instanceName), "setup", "cluster-services", serviceName)
	kustomizeDir := filepath.Join(instanceServiceDir, "kustomize")
	lastDeployFile := filepath.Join(instanceServiceDir, ".last-deploy")

	if storage.FileExists(kustomizeDir) && storage.FileExists(lastDeployFile) {
		kustomizeModTime := getDirectoryModTime(kustomizeDir)
		lastDeployTime := getFileModTime(lastDeployFile)

		// If kustomize files are newer than last deploy, manifests need to be redeployed
		if !kustomizeModTime.IsZero() && !lastDeployTime.IsZero() {
			if kustomizeModTime.After(lastDeployTime) {
				return DeploymentState{
					State:   "needs_redeploy",
					Healthy: deploymentInfo.Ready == deploymentInfo.Desired,
					Replicas: deploymentInfo,
				}
			}
		}
	}

	// Determine health
	healthy := deploymentInfo.Ready == deploymentInfo.Desired && deploymentInfo.Desired > 0
	if deploymentInfo.Desired == 0 {
		// Scaled to zero or DaemonSet with no matching nodes is still "deployed"
		healthy = true
	}

	// Determine state
	state := "deployed"
	if !healthy {
		if deploymentInfo.Ready < deploymentInfo.Desired {
			if deploymentInfo.Current > deploymentInfo.Desired {
				state = "progressing"
			} else {
				state = "degraded"
			}
		}
	}

	return DeploymentState{
		State:   state,
		Healthy: healthy,
		Replicas: deploymentInfo,
	}
}

// Helper functions for file/directory modification times
func getFileModTime(path string) time.Time {
	if info, err := os.Stat(path); err == nil {
		return info.ModTime()
	}
	return time.Time{}
}

func getDirectoryModTime(path string) time.Time {
	var latestModTime time.Time
	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			if info.ModTime().After(latestModTime) {
				latestModTime = info.ModTime()
			}
		}
		return nil
	})
	return latestModTime
}

// List returns all base services and their status
func (m *Manager) List(instanceName string) ([]Service, error) {
	services := []Service{}

	// Discover services from embedded setup files
	serviceNames, err := setup.ListServices()
	if err != nil {
		return nil, fmt.Errorf("failed to list services from embedded files: %w", err)
	}

	for _, name := range serviceNames {
		// Skip SMTP - it's now managed as cloud configuration, not a deployable service
		if name == "smtp" {
			continue
		}

		// Get service info from manifest if available
		var namespace, description, version string
		var dependencies []string
		var hasConfig bool

		if manifest, ok := m.manifests[name]; ok {
			namespace = manifest.Namespace
			description = manifest.Description
			version = manifest.Version
			dependencies = manifest.Dependencies
			hasConfig = len(manifest.ServiceConfig) > 0
		} else {
			// Service not in manifests, skip
			continue
		}

		// Get lifecycle status
		lifecycleStatus := m.getServiceLifecycleStatus(instanceName, name)

		service := Service{
			Name:         name,
			Status:       m.checkServiceStatus(instanceName, name),
			Namespace:    namespace,
			Description:  description,
			Version:      version,
			Dependencies: dependencies,
			HasConfig:    hasConfig,
			Lifecycle:    lifecycleStatus,
		}

		services = append(services, service)
	}

	return services, nil
}

// Get returns a specific service
func (m *Manager) Get(instanceName, serviceName string) (*Service, error) {
	manifest, ok := m.manifests[serviceName]
	if !ok {
		return nil, fmt.Errorf("service not found: %s", serviceName)
	}

	// Get lifecycle status
	lifecycleStatus := m.getServiceLifecycleStatus(instanceName, serviceName)

	service := &Service{
		Name:             serviceName,
		Status:           m.checkServiceStatus(instanceName, serviceName),
		Namespace:        manifest.Namespace,
		StorageClassName: manifest.StorageClassName,
		Lifecycle:        lifecycleStatus,
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

	// Check if service exists in embedded files
	if !setup.ServiceExists(serviceName) {
		return fmt.Errorf("service %s not found", serviceName)
	}

	// Get kustomize directory from instance
	instanceServiceDir := filepath.Join(tools.GetInstancePath(m.dataDir, instanceName), "setup", "cluster-services", serviceName)
	kustomizeDir := filepath.Join(instanceServiceDir, "kustomize")
	kustomizationFile := filepath.Join(kustomizeDir, "kustomization.yaml")

	// Check if kustomize directory exists (service is installed)
	if !storage.FileExists(kustomizeDir) {
		return fmt.Errorf("service not installed - kustomize directory not found")
	}

	// Verify kustomization.yaml exists
	if !storage.FileExists(kustomizationFile) {
		return fmt.Errorf("kustomization.yaml not found - cannot delete service")
	}

	// Delete using kustomize
	cmd := exec.Command("kubectl", "delete", "-k", kustomizeDir)
	tools.WithKubeconfig(cmd, kubeconfigPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete service: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// GetStatus returns detailed status for a service
func (m *Manager) GetStatus(instanceName, serviceName string) (*Service, error) {
	manifest, ok := m.manifests[serviceName]
	if !ok {
		return nil, fmt.Errorf("service not found: %s", serviceName)
	}

	// Get lifecycle status
	lifecycleStatus := m.getServiceLifecycleStatus(instanceName, serviceName)

	service := &Service{
		Name:      serviceName,
		Namespace: manifest.Namespace,
		Status:    m.checkServiceStatus(instanceName, serviceName),
		Lifecycle: lifecycleStatus,
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

// Fetch extracts service files from embedded setup to instance
func (m *Manager) Fetch(instanceName, serviceName string) error {
	// 1. Validate service exists in embedded files
	if !setup.ServiceExists(serviceName) {
		return fmt.Errorf("service %s not found in embedded files", serviceName)
	}

	// 2. Create instance service directory
	instanceDir := filepath.Join(tools.GetInstancePath(m.dataDir, instanceName),
		"setup", "cluster-services", serviceName)
	if err := os.MkdirAll(instanceDir, 0755); err != nil {
		return fmt.Errorf("failed to create service directory: %w", err)
	}

	// 3. Extract files from embedded setup:
	//    - README.md (if exists, optional)
	//    - install.sh (if exists, optional)
	//    - wild-manifest.yaml
	//    - kustomize.template/* (if exists, optional)

	// Extract README.md if it exists
	if readmeData, err := setup.GetServiceFile(serviceName, "README.md"); err == nil {
		_ = os.WriteFile(filepath.Join(instanceDir, "README.md"), readmeData, 0644)
	}

	// Extract install.sh if it exists
	if installData, err := setup.GetServiceFile(serviceName, "install.sh"); err == nil {
		installPath := filepath.Join(instanceDir, "install.sh")
		if err := os.WriteFile(installPath, installData, 0755); err != nil {
			return fmt.Errorf("failed to write install.sh: %w", err)
		}
	}

	// Extract wild-manifest.yaml
	if manifestData, err := setup.GetServiceFile(serviceName, "wild-manifest.yaml"); err == nil {
		if err := os.WriteFile(filepath.Join(instanceDir, "wild-manifest.yaml"), manifestData, 0644); err != nil {
			return fmt.Errorf("failed to write wild-manifest.yaml: %w", err)
		}
	}

	// Extract kustomize.template directory
	templateFS, err := setup.GetKustomizeTemplate(serviceName)
	if err == nil {
		destTemplateDir := filepath.Join(instanceDir, "kustomize.template")
		if err := extractFS(templateFS, destTemplateDir); err != nil {
			return fmt.Errorf("failed to extract templates: %w", err)
		}
	}

	return nil
}

// serviceFilesExist checks if service files exist in the instance
func (m *Manager) serviceFilesExist(instanceName, serviceName string) bool {
	serviceDir := filepath.Join(tools.GetInstancePath(m.dataDir, instanceName),
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

// extractFS extracts files from an fs.FS to a destination directory
func extractFS(fsys fs.FS, dst string) error {
	return fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Create destination path
		dstPath := filepath.Join(dst, path)

		if d.IsDir() {
			// Create directory
			return os.MkdirAll(dstPath, 0755)
		}

		// Read file from embedded FS
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}

		// Write file to destination
		return os.WriteFile(dstPath, data, 0644)
	})
}

// Compile processes gomplate templates into final Kubernetes manifests
func (m *Manager) Compile(instanceName, serviceName string) error {
	instanceDir := tools.GetInstancePath(m.dataDir, instanceName)
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

	// Touch all output files to update their modification times
	// This ensures lifecycle detection recognizes the compile happened
	now := time.Now()
	filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			os.Chtimes(path, now, now)
		}
		return nil
	})

	return nil
}

// Deploy executes the service-specific install.sh script
// opID and broadcaster are optional - if provided, output will be streamed to SSE clients
func (m *Manager) Deploy(instanceName, serviceName, opID string, broadcaster *operations.Broadcaster) error {
	fmt.Printf("[DEBUG] Deploy() called for service=%s instance=%s opID=%s\n", serviceName, instanceName, opID)

	instanceDir := tools.GetInstancePath(m.dataDir, instanceName)
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
		fmt.Sprintf("WILD_API_DATA_DIR=%s", m.dataDir),
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

	var err error
	if outputWriter != nil {
		// Stream output to file and SSE clients
		cmd.Stdout = outputWriter
		cmd.Stderr = outputWriter
		fmt.Printf("[DEBUG] Starting command execution for opID=%s\n", opID)
		err = cmd.Run()
		fmt.Printf("[DEBUG] Command completed for opID=%s, err=%v\n", opID, err)
		if broadcaster != nil {
			outputWriter.Flush()    // Flush any remaining buffered data
			broadcaster.Close(opID) // Close all SSE clients
		}
	} else {
		// Fallback: capture output for logging (backward compatibility)
		output, cmdErr := cmd.CombinedOutput()
		fmt.Printf("=== Deploy %s output ===\n%s\n=== End output ===\n", serviceName, output)
		if cmdErr != nil {
			return fmt.Errorf("deployment failed: %w\nOutput: %s", cmdErr, output)
		}
	}

	// If deployment succeeded, touch .last-deploy file to track deployment time
	if err == nil {
		lastDeployFile := filepath.Join(serviceDir, ".last-deploy")
		now := time.Now()
		if touchErr := os.Chtimes(lastDeployFile, now, now); touchErr != nil {
			// If file doesn't exist, create it
			if os.IsNotExist(touchErr) {
				if file, createErr := os.Create(lastDeployFile); createErr == nil {
					file.Close()
				}
			}
		}
	}

	return err
}

// validateConfig checks that all required config is set for a service
func (m *Manager) validateConfig(instanceName, serviceName string) error {
	manifest, err := m.GetManifest(serviceName)
	if err != nil {
		return err // Service has no manifest
	}

	// Load instance config
	configFile := tools.GetInstanceConfigPath(m.dataDir, instanceName)

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

// CleanFiles removes cached and compiled service files from the instance directory
func (m *Manager) CleanFiles(instanceName, serviceName string) error {
	// Construct service path: {dataDir}/instances/{instance}/setup/cluster-services/{service}
	servicePath := filepath.Join(m.dataDir, "instances", instanceName, "setup", "cluster-services", serviceName)

	// Check if service directory exists
	if _, err := os.Stat(servicePath); os.IsNotExist(err) {
		// Service doesn't exist, nothing to clean
		return nil
	}

	// Remove the entire service directory
	if err := os.RemoveAll(servicePath); err != nil {
		return fmt.Errorf("failed to remove service files: %w", err)
	}

	return nil
}
