package context

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/wild-cloud/wild-central/daemon/internal/storage"
)

// Manager handles current instance context tracking
type Manager struct {
	dataDir string
}

// NewManager creates a new context manager
func NewManager(dataDir string) *Manager {
	return &Manager{
		dataDir: dataDir,
	}
}

// GetContextFilePath returns the path to the context file
func (m *Manager) GetContextFilePath() string {
	return filepath.Join(m.dataDir, "current-context")
}

// GetCurrentContext retrieves the name of the current instance context
func (m *Manager) GetCurrentContext() (string, error) {
	contextFile := m.GetContextFilePath()

	if !storage.FileExists(contextFile) {
		return "", fmt.Errorf("no current context set")
	}

	content, err := storage.ReadFile(contextFile)
	if err != nil {
		return "", fmt.Errorf("reading context file: %w", err)
	}

	contextName := strings.TrimSpace(string(content))
	if contextName == "" {
		return "", fmt.Errorf("context file is empty")
	}

	return contextName, nil
}

// SetCurrentContext sets the current instance context
func (m *Manager) SetCurrentContext(instanceName string) error {
	if instanceName == "" {
		return fmt.Errorf("instance name cannot be empty")
	}

	// Verify instance exists
	instancePath := filepath.Join(m.dataDir, "instances", instanceName)
	if !storage.FileExists(instancePath) {
		return fmt.Errorf("instance %s does not exist", instanceName)
	}

	contextFile := m.GetContextFilePath()

	// Ensure data directory exists
	if err := storage.EnsureDir(m.dataDir, 0755); err != nil {
		return err
	}

	// Acquire lock before writing
	lockPath := contextFile + ".lock"
	return storage.WithLock(lockPath, func() error {
		return storage.WriteFile(contextFile, []byte(instanceName), 0644)
	})
}

// ClearCurrentContext removes the current context
func (m *Manager) ClearCurrentContext() error {
	contextFile := m.GetContextFilePath()

	if !storage.FileExists(contextFile) {
		// Already cleared
		return nil
	}

	// Acquire lock before deleting
	lockPath := contextFile + ".lock"
	return storage.WithLock(lockPath, func() error {
		return storage.WriteFile(contextFile, []byte(""), 0644)
	})
}

// HasCurrentContext checks if a current context is set
func (m *Manager) HasCurrentContext() bool {
	_, err := m.GetCurrentContext()
	return err == nil
}

// ValidateContext checks if the current context is valid (instance exists)
func (m *Manager) ValidateContext() error {
	contextName, err := m.GetCurrentContext()
	if err != nil {
		return err
	}

	instancePath := filepath.Join(m.dataDir, "instances", contextName)
	if !storage.FileExists(instancePath) {
		return fmt.Errorf("current context %s points to non-existent instance", contextName)
	}

	return nil
}

// GetCurrentInstancePath returns the path to the current instance directory
func (m *Manager) GetCurrentInstancePath() (string, error) {
	contextName, err := m.GetCurrentContext()
	if err != nil {
		return "", err
	}

	return filepath.Join(m.dataDir, "instances", contextName), nil
}

// GetCurrentInstanceConfigPath returns the path to the current instance's config file
func (m *Manager) GetCurrentInstanceConfigPath() (string, error) {
	instancePath, err := m.GetCurrentInstancePath()
	if err != nil {
		return "", err
	}

	return filepath.Join(instancePath, "config.yaml"), nil
}

// GetCurrentInstanceSecretsPath returns the path to the current instance's secrets file
func (m *Manager) GetCurrentInstanceSecretsPath() (string, error) {
	instancePath, err := m.GetCurrentInstancePath()
	if err != nil {
		return "", err
	}

	return filepath.Join(instancePath, "secrets.yaml"), nil
}
