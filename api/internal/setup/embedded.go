package setup

import (
	"embed"
	"fmt"
	"io/fs"
	"path"

	"gopkg.in/yaml.v3"
)

// Embed all setup directories into the binary
//
//go:embed cluster-services/* cluster-nodes/* dnsmasq/* operator/* README.md
var setupFS embed.FS

// Legacy alias for backward compatibility
var clusterServices = setupFS

// ServiceManifest represents the wild-manifest.yaml structure
type ServiceManifest struct {
	Name             string                      `yaml:"name"`
	Description      string                      `yaml:"description"`
	Version          string                      `yaml:"version"`
	Category         string                      `yaml:"category"`
	Namespace        string                      `yaml:"namespace"`
	DeploymentName   string                      `yaml:"deploymentName,omitempty"` // Optional: defaults to Name
	Dependencies     []string                    `yaml:"dependencies,omitempty"`
	ConfigReferences []string                    `yaml:"configReferences,omitempty"`
	ServiceConfig    map[string]ConfigDefinition `yaml:"serviceConfig,omitempty"`
}

// ConfigDefinition defines config that should be prompted during service setup
type ConfigDefinition struct {
	Path    string `yaml:"path"`
	Prompt  string `yaml:"prompt"`
	Default string `yaml:"default"`
	Type    string `yaml:"type,omitempty"`
}

// ListServices returns all available cluster services
func ListServices() ([]string, error) {
	entries, err := clusterServices.ReadDir("cluster-services")
	if err != nil {
		return nil, fmt.Errorf("reading cluster services: %w", err)
	}

	var services []string
	for _, entry := range entries {
		if entry.IsDir() {
			services = append(services, entry.Name())
		}
	}
	return services, nil
}

// GetServiceFile returns a specific file from a service
func GetServiceFile(serviceName, filePath string) ([]byte, error) {
	fullPath := path.Join("cluster-services", serviceName, filePath)
	data, err := clusterServices.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("reading service file %s: %w", fullPath, err)
	}
	return data, nil
}

// GetServiceFS returns the filesystem for a specific service
func GetServiceFS(serviceName string) (fs.FS, error) {
	subFS, err := fs.Sub(clusterServices, path.Join("cluster-services", serviceName))
	if err != nil {
		return nil, fmt.Errorf("accessing service %s: %w", serviceName, err)
	}
	return subFS, nil
}

// GetManifest returns the parsed wild-manifest.yaml for a service
func GetManifest(serviceName string) (*ServiceManifest, error) {
	data, err := GetServiceFile(serviceName, "wild-manifest.yaml")
	if err != nil {
		return nil, err
	}

	var manifest ServiceManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parsing manifest for %s: %w", serviceName, err)
	}

	return &manifest, nil
}

// GetKustomizeTemplate returns all files from the kustomize.template directory
func GetKustomizeTemplate(serviceName string) (fs.FS, error) {
	serviceFS, err := GetServiceFS(serviceName)
	if err != nil {
		return nil, err
	}

	templateFS, err := fs.Sub(serviceFS, "kustomize.template")
	if err != nil {
		return nil, fmt.Errorf("accessing kustomize.template for %s: %w", serviceName, err)
	}

	return templateFS, nil
}

// ServiceExists checks if a service exists in the embedded files
func ServiceExists(serviceName string) bool {
	services, err := ListServices()
	if err != nil {
		return false
	}

	for _, s := range services {
		if s == serviceName {
			return true
		}
	}
	return false
}

// GetClusterNodesFS returns the filesystem for cluster-nodes setup files
func GetClusterNodesFS() (fs.FS, error) {
	subFS, err := fs.Sub(setupFS, "cluster-nodes")
	if err != nil {
		return nil, fmt.Errorf("accessing cluster-nodes: %w", err)
	}
	return subFS, nil
}

// GetClusterNodesFile returns a specific file from cluster-nodes
func GetClusterNodesFile(filePath string) ([]byte, error) {
	fullPath := path.Join("cluster-nodes", filePath)
	data, err := setupFS.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("reading cluster-nodes file %s: %w", filePath, err)
	}
	return data, nil
}

// GetDNSMasqFS returns the filesystem for dnsmasq setup files
func GetDNSMasqFS() (fs.FS, error) {
	subFS, err := fs.Sub(setupFS, "dnsmasq")
	if err != nil {
		return nil, fmt.Errorf("accessing dnsmasq: %w", err)
	}
	return subFS, nil
}

// GetDNSMasqFile returns a specific file from dnsmasq
func GetDNSMasqFile(filePath string) ([]byte, error) {
	fullPath := path.Join("dnsmasq", filePath)
	data, err := setupFS.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("reading dnsmasq file %s: %w", filePath, err)
	}
	return data, nil
}

// GetOperatorFS returns the filesystem for operator setup files
func GetOperatorFS() (fs.FS, error) {
	subFS, err := fs.Sub(setupFS, "operator")
	if err != nil {
		return nil, fmt.Errorf("accessing operator: %w", err)
	}
	return subFS, nil
}

// GetOperatorFile returns a specific file from operator
func GetOperatorFile(filePath string) ([]byte, error) {
	fullPath := path.Join("operator", filePath)
	data, err := setupFS.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("reading operator file %s: %w", filePath, err)
	}
	return data, nil
}

// GetReadme returns the setup README.md content
func GetReadme() ([]byte, error) {
	data, err := setupFS.ReadFile("README.md")
	if err != nil {
		return nil, fmt.Errorf("reading README.md: %w", err)
	}
	return data, nil
}
