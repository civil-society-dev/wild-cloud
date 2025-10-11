package config

import (
	"fmt"
	"path/filepath"

	"github.com/wild-cloud/wild-central/daemon/internal/storage"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
)

// Manager handles configuration file operations with idempotency
type Manager struct {
	yq *tools.YQ
}

// NewManager creates a new config manager
func NewManager() *Manager {
	return &Manager{
		yq: tools.NewYQ(),
	}
}

// EnsureInstanceConfig ensures an instance config file exists with proper structure
func (m *Manager) EnsureInstanceConfig(instancePath string) error {
	configPath := filepath.Join(instancePath, "config.yaml")

	// Check if config already exists
	if storage.FileExists(configPath) {
		// Validate existing config
		if err := m.yq.Validate(configPath); err != nil {
			return fmt.Errorf("invalid config file: %w", err)
		}
		return nil
	}

	// Create minimal config structure
	initialConfig := `# Wild Cloud Instance Configuration
baseDomain: ""
domain: ""
internalDomain: ""
dhcpRange: ""
backup:
  root: ""
nfs:
  host: ""
  mediaPath: ""
cluster:
  name: ""
  loadBalancerIp: ""
  ipAddressPool: ""
  hostnamePrefix: ""
  certManager:
    cloudflare:
      domain: ""
      zoneID: ""
  externalDns:
    ownerId: ""
  nodes:
    talos:
      version: ""
      schematicId: ""
    control:
      vip: ""
    activeNodes: []
`

	// Ensure instance directory exists
	if err := storage.EnsureDir(instancePath, 0755); err != nil {
		return err
	}

	// Write config with proper permissions
	if err := storage.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		return err
	}

	return nil
}

// GetConfigValue retrieves a value from a config file
func (m *Manager) GetConfigValue(configPath, key string) (string, error) {
	if !storage.FileExists(configPath) {
		return "", fmt.Errorf("config file not found: %s", configPath)
	}

	value, err := m.yq.Get(configPath, fmt.Sprintf(".%s", key))
	if err != nil {
		return "", fmt.Errorf("getting config value %s: %w", key, err)
	}

	return value, nil
}

// SetConfigValue sets a value in a config file
func (m *Manager) SetConfigValue(configPath, key, value string) error {
	if !storage.FileExists(configPath) {
		return fmt.Errorf("config file not found: %s", configPath)
	}

	// Acquire lock before modifying
	lockPath := configPath + ".lock"
	return storage.WithLock(lockPath, func() error {
		return m.yq.Set(configPath, fmt.Sprintf(".%s", key), value)
	})
}

// EnsureConfigValue sets a value only if it's not already set (idempotent)
func (m *Manager) EnsureConfigValue(configPath, key, value string) error {
	if !storage.FileExists(configPath) {
		return fmt.Errorf("config file not found: %s", configPath)
	}

	// Check if value already set
	currentValue, err := m.GetConfigValue(configPath, key)
	if err == nil && currentValue != "" && currentValue != "null" {
		// Value already set, skip
		return nil
	}

	// Set the value
	return m.SetConfigValue(configPath, key, value)
}

// ValidateConfig validates a config file
func (m *Manager) ValidateConfig(configPath string) error {
	if !storage.FileExists(configPath) {
		return fmt.Errorf("config file not found: %s", configPath)
	}

	return m.yq.Validate(configPath)
}

// CopyConfig copies a config file to a new location
func (m *Manager) CopyConfig(srcPath, dstPath string) error {
	if !storage.FileExists(srcPath) {
		return fmt.Errorf("source config file not found: %s", srcPath)
	}

	// Read source
	content, err := storage.ReadFile(srcPath)
	if err != nil {
		return err
	}

	// Ensure destination directory exists
	if err := storage.EnsureDir(filepath.Dir(dstPath), 0755); err != nil {
		return err
	}

	// Write destination
	return storage.WriteFile(dstPath, content, 0644)
}

// GetInstanceConfigPath returns the path to an instance's config file
func GetInstanceConfigPath(dataDir, instanceName string) string {
	return filepath.Join(dataDir, "instances", instanceName, "config.yaml")
}

// GetInstanceSecretsPath returns the path to an instance's secrets file
func GetInstanceSecretsPath(dataDir, instanceName string) string {
	return filepath.Join(dataDir, "instances", instanceName, "secrets.yaml")
}

// GetInstancePath returns the path to an instance directory
func GetInstancePath(dataDir, instanceName string) string {
	return filepath.Join(dataDir, "instances", instanceName)
}
