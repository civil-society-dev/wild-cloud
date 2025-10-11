package secrets

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"path/filepath"

	"github.com/wild-cloud/wild-central/daemon/internal/storage"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
)

const (
	// DefaultSecretLength is 32 characters
	DefaultSecretLength = 32
	// Alphanumeric characters for secret generation
	alphanumeric = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

// Manager handles secret generation and storage
type Manager struct {
	yq *tools.YQ
}

// NewManager creates a new secrets manager
func NewManager() *Manager {
	return &Manager{
		yq: tools.NewYQ(),
	}
}

// GenerateSecret generates a cryptographically secure random alphanumeric string
func GenerateSecret(length int) (string, error) {
	if length <= 0 {
		length = DefaultSecretLength
	}

	result := make([]byte, length)
	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphanumeric))))
		if err != nil {
			return "", fmt.Errorf("generating random number: %w", err)
		}
		result[i] = alphanumeric[num.Int64()]
	}

	return string(result), nil
}

// EnsureSecretsFile ensures a secrets file exists with proper structure and permissions
func (m *Manager) EnsureSecretsFile(instancePath string) error {
	secretsPath := filepath.Join(instancePath, "secrets.yaml")

	// Check if secrets file already exists
	if storage.FileExists(secretsPath) {
		// Ensure proper permissions
		if err := storage.EnsureFilePermissions(secretsPath, 0600); err != nil {
			return err
		}
		return nil
	}

	// Create minimal secrets structure
	initialSecrets := `# Wild Cloud Instance Secrets
# WARNING: This file contains sensitive data. Keep secure!
cluster:
  talosSecrets: ""
  kubeconfig: ""
certManager:
  cloudflare:
    apiToken: ""
`

	// Ensure instance directory exists
	if err := storage.EnsureDir(instancePath, 0755); err != nil {
		return err
	}

	// Write secrets file with restrictive permissions (0600)
	if err := storage.WriteFile(secretsPath, []byte(initialSecrets), 0600); err != nil {
		return err
	}

	return nil
}

// GetSecret retrieves a secret value from a secrets file
func (m *Manager) GetSecret(secretsPath, key string) (string, error) {
	if !storage.FileExists(secretsPath) {
		return "", fmt.Errorf("secrets file not found: %s", secretsPath)
	}

	value, err := m.yq.Get(secretsPath, fmt.Sprintf(".%s", key))
	if err != nil {
		return "", fmt.Errorf("getting secret %s: %w", key, err)
	}

	return value, nil
}

// SetSecret sets a secret value in a secrets file
func (m *Manager) SetSecret(secretsPath, key, value string) error {
	if !storage.FileExists(secretsPath) {
		return fmt.Errorf("secrets file not found: %s", secretsPath)
	}

	// Acquire lock before modifying
	lockPath := secretsPath + ".lock"
	return storage.WithLock(lockPath, func() error {
		// Don't wrap value in quotes - yq handles YAML quoting automatically
		if err := m.yq.Set(secretsPath, fmt.Sprintf(".%s", key), value); err != nil {
			return err
		}
		// Ensure permissions remain secure after modification
		return storage.EnsureFilePermissions(secretsPath, 0600)
	})
}

// EnsureSecret generates and sets a secret only if it doesn't exist (idempotent)
func (m *Manager) EnsureSecret(secretsPath, key string, length int) (string, error) {
	if !storage.FileExists(secretsPath) {
		return "", fmt.Errorf("secrets file not found: %s", secretsPath)
	}

	// Check if secret already exists
	existingSecret, err := m.GetSecret(secretsPath, key)
	if err == nil && existingSecret != "" && existingSecret != "null" {
		// Secret already exists, return it
		return existingSecret, nil
	}

	// Generate new secret
	secret, err := GenerateSecret(length)
	if err != nil {
		return "", err
	}

	// Set the secret
	if err := m.SetSecret(secretsPath, key, secret); err != nil {
		return "", err
	}

	return secret, nil
}

// GenerateAndStoreSecret is a convenience function that generates a secret and stores it
func (m *Manager) GenerateAndStoreSecret(secretsPath, key string) (string, error) {
	return m.EnsureSecret(secretsPath, key, DefaultSecretLength)
}

// DeleteSecret removes a secret from a secrets file
func (m *Manager) DeleteSecret(secretsPath, key string) error {
	if !storage.FileExists(secretsPath) {
		return fmt.Errorf("secrets file not found: %s", secretsPath)
	}

	// Acquire lock before modifying
	lockPath := secretsPath + ".lock"
	return storage.WithLock(lockPath, func() error {
		if err := m.yq.Delete(secretsPath, fmt.Sprintf(".%s", key)); err != nil {
			return err
		}
		// Ensure permissions remain secure after modification
		return storage.EnsureFilePermissions(secretsPath, 0600)
	})
}
