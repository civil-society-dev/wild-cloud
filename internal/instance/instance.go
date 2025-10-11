package instance

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wild-cloud/wild-central/daemon/internal/config"
	"github.com/wild-cloud/wild-central/daemon/internal/context"
	"github.com/wild-cloud/wild-central/daemon/internal/secrets"
	"github.com/wild-cloud/wild-central/daemon/internal/storage"
)

// Manager handles instance lifecycle operations
type Manager struct {
	dataDir       string
	configMgr     *config.Manager
	secretsMgr    *secrets.Manager
	contextMgr    *context.Manager
}

// NewManager creates a new instance manager
func NewManager(dataDir string) *Manager {
	return &Manager{
		dataDir:       dataDir,
		configMgr:     config.NewManager(),
		secretsMgr:    secrets.NewManager(),
		contextMgr:    context.NewManager(dataDir),
	}
}

// Instance represents a Wild Cloud instance
type Instance struct {
	Name       string
	Path       string
	ConfigPath string
	SecretsPath string
}

// GetInstancePath returns the path to an instance directory
func (m *Manager) GetInstancePath(name string) string {
	return filepath.Join(m.dataDir, "instances", name)
}

// GetInstanceConfigPath returns the path to an instance's config file
func (m *Manager) GetInstanceConfigPath(name string) string {
	return filepath.Join(m.GetInstancePath(name), "config.yaml")
}

// GetInstanceSecretsPath returns the path to an instance's secrets file
func (m *Manager) GetInstanceSecretsPath(name string) string {
	return filepath.Join(m.GetInstancePath(name), "secrets.yaml")
}

// InstanceExists checks if an instance exists
func (m *Manager) InstanceExists(name string) bool {
	return storage.FileExists(m.GetInstancePath(name))
}

// CreateInstance creates a new Wild Cloud instance with initial structure
func (m *Manager) CreateInstance(name string) error {
	if name == "" {
		return fmt.Errorf("instance name cannot be empty")
	}

	instancePath := m.GetInstancePath(name)

	// Check if instance already exists (idempotency - just return success)
	if m.InstanceExists(name) {
		return nil
	}

	// Acquire lock for instance creation
	lockPath := filepath.Join(m.dataDir, "instances", ".lock")
	return storage.WithLock(lockPath, func() error {
		// Create instance directory
		if err := storage.EnsureDir(instancePath, 0755); err != nil {
			return fmt.Errorf("creating instance directory: %w", err)
		}

		// Create config file
		if err := m.configMgr.EnsureInstanceConfig(instancePath); err != nil {
			return fmt.Errorf("creating config file: %w", err)
		}

		// Create secrets file
		if err := m.secretsMgr.EnsureSecretsFile(instancePath); err != nil {
			return fmt.Errorf("creating secrets file: %w", err)
		}

		// Create subdirectories
		subdirs := []string{"talos", "k8s", "logs", "backups"}
		for _, subdir := range subdirs {
			subdirPath := filepath.Join(instancePath, subdir)
			if err := storage.EnsureDir(subdirPath, 0755); err != nil {
				return fmt.Errorf("creating subdirectory %s: %w", subdir, err)
			}
		}

		return nil
	})
}

// DeleteInstance removes a Wild Cloud instance
func (m *Manager) DeleteInstance(name string) error {
	if name == "" {
		return fmt.Errorf("instance name cannot be empty")
	}

	instancePath := m.GetInstancePath(name)

	// Check if instance exists
	if !m.InstanceExists(name) {
		return fmt.Errorf("instance %s does not exist", name)
	}

	// Clear context if this is the current instance
	currentContext, err := m.contextMgr.GetCurrentContext()
	if err == nil && currentContext == name {
		if err := m.contextMgr.ClearCurrentContext(); err != nil {
			return fmt.Errorf("clearing current context: %w", err)
		}
	}

	// Acquire lock for instance deletion
	lockPath := filepath.Join(m.dataDir, "instances", ".lock")
	return storage.WithLock(lockPath, func() error {
		// Remove instance directory
		if err := os.RemoveAll(instancePath); err != nil {
			return fmt.Errorf("removing instance directory: %w", err)
		}

		return nil
	})
}

// ListInstances returns a list of all instance names
func (m *Manager) ListInstances() ([]string, error) {
	instancesDir := filepath.Join(m.dataDir, "instances")

	// Ensure instances directory exists
	if !storage.FileExists(instancesDir) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(instancesDir)
	if err != nil {
		return nil, fmt.Errorf("reading instances directory: %w", err)
	}

	var instances []string
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != ".lock" {
			instances = append(instances, entry.Name())
		}
	}

	return instances, nil
}

// GetInstance retrieves instance information
func (m *Manager) GetInstance(name string) (*Instance, error) {
	if !m.InstanceExists(name) {
		return nil, fmt.Errorf("instance %s does not exist", name)
	}

	return &Instance{
		Name:        name,
		Path:        m.GetInstancePath(name),
		ConfigPath:  m.GetInstanceConfigPath(name),
		SecretsPath: m.GetInstanceSecretsPath(name),
	}, nil
}

// GetCurrentInstance returns the current context instance
func (m *Manager) GetCurrentInstance() (*Instance, error) {
	name, err := m.contextMgr.GetCurrentContext()
	if err != nil {
		return nil, err
	}

	return m.GetInstance(name)
}

// SetCurrentInstance sets the current instance context
func (m *Manager) SetCurrentInstance(name string) error {
	if !m.InstanceExists(name) {
		return fmt.Errorf("instance %s does not exist", name)
	}

	return m.contextMgr.SetCurrentContext(name)
}

// ValidateInstance checks if an instance has valid structure
func (m *Manager) ValidateInstance(name string) error {
	if !m.InstanceExists(name) {
		return fmt.Errorf("instance %s does not exist", name)
	}

	instance, err := m.GetInstance(name)
	if err != nil {
		return err
	}

	// Check config file exists and is valid
	if !storage.FileExists(instance.ConfigPath) {
		return fmt.Errorf("config file missing for instance %s", name)
	}

	if err := m.configMgr.ValidateConfig(instance.ConfigPath); err != nil {
		return fmt.Errorf("invalid config for instance %s: %w", name, err)
	}

	// Check secrets file exists with proper permissions
	if !storage.FileExists(instance.SecretsPath) {
		return fmt.Errorf("secrets file missing for instance %s", name)
	}

	// Verify secrets file permissions
	info, err := os.Stat(instance.SecretsPath)
	if err != nil {
		return fmt.Errorf("checking secrets file permissions: %w", err)
	}

	if info.Mode().Perm() != 0600 {
		return fmt.Errorf("secrets file has incorrect permissions (expected 0600, got %04o)", info.Mode().Perm())
	}

	return nil
}

// InitializeInstance performs initial setup for a newly created instance
func (m *Manager) InitializeInstance(name string, initialConfig map[string]string) error {
	if !m.InstanceExists(name) {
		return fmt.Errorf("instance %s does not exist", name)
	}

	instance, err := m.GetInstance(name)
	if err != nil {
		return err
	}

	// Set initial config values
	for key, value := range initialConfig {
		if err := m.configMgr.SetConfigValue(instance.ConfigPath, key, value); err != nil {
			return fmt.Errorf("setting config value %s: %w", key, err)
		}
	}

	return nil
}
