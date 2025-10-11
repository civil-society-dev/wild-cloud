package services

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/wild-cloud/wild-central/daemon/internal/storage"
)

// ServiceManifest defines a service deployment configuration
// Matches the simple app manifest pattern
type ServiceManifest struct {
	Name             string                      `yaml:"name" json:"name"`
	Description      string                      `yaml:"description" json:"description"`
	Namespace        string                      `yaml:"namespace" json:"namespace"`
	Category         string                      `yaml:"category,omitempty" json:"category,omitempty"`
	Dependencies     []string                    `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	ConfigReferences []string                    `yaml:"configReferences,omitempty" json:"configReferences,omitempty"`
	ServiceConfig    map[string]ConfigDefinition `yaml:"serviceConfig,omitempty" json:"serviceConfig,omitempty"`
}

// ConfigDefinition defines config that should be prompted during service setup
type ConfigDefinition struct {
	Path    string `yaml:"path" json:"path"`               // Config path to set
	Prompt  string `yaml:"prompt" json:"prompt"`           // User prompt text
	Default string `yaml:"default" json:"default"`         // Default value (supports templates)
	Type    string `yaml:"type,omitempty" json:"type,omitempty"` // Value type: string|int|bool (default: string)
}

// LoadManifest reads and parses a service manifest from a service directory
func LoadManifest(serviceDir string) (*ServiceManifest, error) {
	manifestPath := filepath.Join(serviceDir, "wild-manifest.yaml")

	if !storage.FileExists(manifestPath) {
		return nil, fmt.Errorf("manifest not found: %s", manifestPath)
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest ServiceManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Validate required fields
	if manifest.Name == "" {
		return nil, fmt.Errorf("manifest missing name")
	}
	if manifest.Namespace == "" {
		return nil, fmt.Errorf("manifest missing namespace")
	}

	return &manifest, nil
}

// LoadAllManifests loads manifests for all services in a directory
func LoadAllManifests(servicesDir string) (map[string]*ServiceManifest, error) {
	manifests := make(map[string]*ServiceManifest)

	entries, err := os.ReadDir(servicesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read services directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		serviceDir := filepath.Join(servicesDir, entry.Name())
		manifest, err := LoadManifest(serviceDir)
		if err != nil {
			// Skip services without manifests (they might not be migrated yet)
			continue
		}

		manifests[manifest.Name] = manifest
	}

	return manifests, nil
}

// GetDeploymentName returns the primary deployment name for this service
// Uses name as the deployment name by default
func (m *ServiceManifest) GetDeploymentName() string {
	// For now, assume deployment name matches service name
	// This can be made configurable if needed
	return m.Name
}

// GetRequiredConfig returns all config paths that must be set
func (m *ServiceManifest) GetRequiredConfig() []string {
	var required []string

	// Add all service config paths (these will be prompted)
	for _, cfg := range m.ServiceConfig {
		required = append(required, cfg.Path)
	}

	return required
}

// GetAllConfigPaths returns all config paths (references + service config)
func (m *ServiceManifest) GetAllConfigPaths() []string {
	var paths []string

	// Config references (must already exist)
	paths = append(paths, m.ConfigReferences...)

	// Service config (will be prompted)
	for _, cfg := range m.ServiceConfig {
		paths = append(paths, cfg.Path)
	}

	return paths
}
