package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateSecret(t *testing.T) {
	// Test various lengths
	lengths := []int{32, 64, 128}
	for _, length := range lengths {
		secret, err := GenerateSecret(length)
		if err != nil {
			t.Fatalf("GenerateSecret(%d) failed: %v", length, err)
		}

		if len(secret) != length {
			t.Errorf("Expected length %d, got %d", length, len(secret))
		}

		// Verify only alphanumeric characters
		for _, c := range secret {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
				t.Errorf("Non-alphanumeric character found: %c", c)
			}
		}
	}

	// Test that secrets are different (not deterministic)
	secret1, _ := GenerateSecret(32)
	secret2, _ := GenerateSecret(32)
	if secret1 == secret2 {
		t.Errorf("Generated secrets should be different")
	}
}

func TestManager_EnsureSecretsFile(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager()

	instancePath := filepath.Join(tmpDir, "test-cloud")
	err := os.MkdirAll(instancePath, 0755)
	if err != nil {
		t.Fatalf("Failed to create instance dir: %v", err)
	}

	// Ensure secrets
	err = m.EnsureSecretsFile(instancePath)
	if err != nil {
		t.Fatalf("EnsureSecretsFile failed: %v", err)
	}

	secretsPath := filepath.Join(instancePath, "secrets.yaml")

	// Verify file exists
	info, err := os.Stat(secretsPath)
	if err != nil {
		t.Fatalf("Secrets file not created: %v", err)
	}

	// Verify permissions are 0600
	mode := info.Mode().Perm()
	if mode != 0600 {
		t.Errorf("Wrong permissions: got %o, want 0600", mode)
	}

	// Test idempotency - calling again should not error
	err = m.EnsureSecretsFile(instancePath)
	if err != nil {
		t.Fatalf("EnsureSecretsFile not idempotent: %v", err)
	}
}

func TestManager_SetAndGetSecret(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager()

	instancePath := filepath.Join(tmpDir, "test-cloud")
	err := os.MkdirAll(instancePath, 0755)
	if err != nil {
		t.Fatalf("Failed to create instance dir: %v", err)
	}

	secretsPath := filepath.Join(instancePath, "secrets.yaml")

	// Initialize secrets
	err = m.EnsureSecretsFile(instancePath)
	if err != nil {
		t.Fatalf("EnsureSecretsFile failed: %v", err)
	}

	// Set a custom secret (requires yq)
	err = m.SetSecret(secretsPath, "customSecret", "myvalue123")
	if err != nil {
		t.Skipf("SetSecret requires yq: %v", err)
		return
	}

	// Get the secret back
	value, err := m.GetSecret(secretsPath, "customSecret")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}

	if value != "myvalue123" {
		t.Errorf("Secret not retrieved correctly: got %q, want %q", value, "myvalue123")
	}

	// Verify permissions still 0600
	info, _ := os.Stat(secretsPath)
	if info.Mode().Perm() != 0600 {
		t.Errorf("Permissions changed after SetSecret")
	}

	// Get non-existent secret should error
	_, err = m.GetSecret(secretsPath, "nonExistent")
	if err == nil {
		t.Fatalf("GetSecret should fail for non-existent secret")
	}
}
