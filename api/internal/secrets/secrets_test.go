package secrets

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/wild-cloud/wild-central/daemon/internal/storage"
)

// Test: GenerateSecret generates valid secrets
func TestGenerateSecret(t *testing.T) {
	tests := []struct {
		name   string
		length int
		want   int
	}{
		{
			name:   "default length",
			length: DefaultSecretLength,
			want:   DefaultSecretLength,
		},
		{
			name:   "custom length 64",
			length: 64,
			want:   64,
		},
		{
			name:   "custom length 128",
			length: 128,
			want:   128,
		},
		{
			name:   "zero length defaults to DefaultSecretLength",
			length: 0,
			want:   DefaultSecretLength,
		},
		{
			name:   "negative length defaults to DefaultSecretLength",
			length: -1,
			want:   DefaultSecretLength,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret, err := GenerateSecret(tt.length)
			if err != nil {
				t.Fatalf("GenerateSecret failed: %v", err)
			}

			if len(secret) != tt.want {
				t.Errorf("got length %d, want %d", len(secret), tt.want)
			}

			// Verify only alphanumeric characters
			for _, c := range secret {
				if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
					t.Errorf("non-alphanumeric character found: %c", c)
				}
			}
		})
	}
}

// Test: GenerateSecret produces unique values
func TestGenerateSecret_Uniqueness(t *testing.T) {
	const numSecrets = 100
	secrets := make(map[string]bool, numSecrets)

	for i := 0; i < numSecrets; i++ {
		secret, err := GenerateSecret(32)
		if err != nil {
			t.Fatalf("GenerateSecret failed: %v", err)
		}

		if secrets[secret] {
			t.Errorf("duplicate secret generated: %s", secret)
		}
		secrets[secret] = true
	}

	if len(secrets) != numSecrets {
		t.Errorf("expected %d unique secrets, got %d", numSecrets, len(secrets))
	}
}

// Test: NewManager creates manager successfully
func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	if m.yq == nil {
		t.Error("Manager.yq is nil")
	}
}

// Test: EnsureSecretsFile creates secrets file with proper structure and permissions
func TestEnsureSecretsFile(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T, instancePath string)
		wantErr     bool
		errContains string
	}{
		{
			name:      "creates secrets file when not exists",
			setupFunc: nil,
			wantErr:   false,
		},
		{
			name: "returns nil when secrets file exists",
			setupFunc: func(t *testing.T, instancePath string) {
				secretsPath := filepath.Join(instancePath, "secrets.yaml")
				content := `# Wild Cloud Instance Secrets
cluster:
  talosSecrets: ""
  kubeconfig: ""
certManager:
  cloudflare:
    apiToken: ""
`
				if err := storage.WriteFile(secretsPath, []byte(content), 0600); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			wantErr: false,
		},
		{
			name: "corrects permissions on existing file",
			setupFunc: func(t *testing.T, instancePath string) {
				secretsPath := filepath.Join(instancePath, "secrets.yaml")
				content := `# Wild Cloud Instance Secrets
cluster:
  talosSecrets: "existing-secret"
  kubeconfig: ""
certManager:
  cloudflare:
    apiToken: ""
`
				// Create with wrong permissions
				if err := storage.WriteFile(secretsPath, []byte(content), 0644); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instancePath := t.TempDir()
			m := NewManager()

			if tt.setupFunc != nil {
				tt.setupFunc(t, instancePath)
			}

			err := m.EnsureSecretsFile(instancePath)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify secrets file exists
			secretsPath := filepath.Join(instancePath, "secrets.yaml")
			if !storage.FileExists(secretsPath) {
				t.Error("secrets file not created")
			}

			// Verify permissions are 0600 (secure)
			info, err := os.Stat(secretsPath)
			if err != nil {
				t.Fatalf("failed to stat secrets file: %v", err)
			}
			if info.Mode().Perm() != 0600 {
				t.Errorf("expected permissions 0600, got %o", info.Mode().Perm())
			}

			// Verify file has expected structure
			content, err := storage.ReadFile(secretsPath)
			if err != nil {
				t.Fatalf("failed to read secrets: %v", err)
			}
			contentStr := string(content)
			requiredFields := []string{"cluster:", "certManager:"}
			for _, field := range requiredFields {
				if !strings.Contains(contentStr, field) {
					t.Errorf("secrets missing required field: %s", field)
				}
			}
		})
	}
}

// Test: GetSecret retrieves secrets correctly
func TestGetSecret(t *testing.T) {
	tests := []struct {
		name        string
		secretsYAML string
		key         string
		want        string
		wantErr     bool
		errContains string
	}{
		{
			name: "get simple string value",
			secretsYAML: `cluster:
  talosSecrets: "my-secret-value"
`,
			key:     "cluster.talosSecrets",
			want:    "my-secret-value",
			wantErr: false,
		},
		{
			name: "get nested value with dot notation",
			secretsYAML: `certManager:
  cloudflare:
    apiToken: "cf-token-12345"
`,
			key:     "certManager.cloudflare.apiToken",
			want:    "cf-token-12345",
			wantErr: false,
		},
		{
			name: "get non-existent key returns error",
			secretsYAML: `cluster:
  talosSecrets: "value"
`,
			key:         "nonexistent",
			wantErr:     true,
			errContains: "secret not found",
		},
		{
			name: "get empty string value returns error",
			secretsYAML: `cluster:
  talosSecrets: ""
`,
			key:         "cluster.talosSecrets",
			wantErr:     true,
			errContains: "secret not found",
		},
		{
			name: "get null value returns error",
			secretsYAML: `cluster:
  talosSecrets: null
`,
			key:         "cluster.talosSecrets",
			wantErr:     true,
			errContains: "secret not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			secretsPath := filepath.Join(tempDir, "secrets.yaml")

			if err := storage.WriteFile(secretsPath, []byte(tt.secretsYAML), 0600); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			m := NewManager()
			got, err := m.GetSecret(secretsPath, tt.key)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// Test: GetSecret error cases
func TestGetSecret_Errors(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) string
		key         string
		errContains string
	}{
		{
			name: "non-existent file",
			setupFunc: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent.yaml")
			},
			key:         "cluster.talosSecrets",
			errContains: "secrets file not found",
		},
		{
			name: "malformed yaml",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				secretsPath := filepath.Join(tempDir, "secrets.yaml")
				content := `invalid: yaml: [[[`
				if err := storage.WriteFile(secretsPath, []byte(content), 0600); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
				return secretsPath
			},
			key:         "cluster.talosSecrets",
			errContains: "getting secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secretsPath := tt.setupFunc(t)
			m := NewManager()

			_, err := m.GetSecret(secretsPath, tt.key)
			if err == nil {
				t.Error("expected error, got nil")
			} else if !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
			}
		})
	}
}

// Test: GetSecret does not leak secrets in error messages
func TestGetSecret_NoSecretLeakage(t *testing.T) {
	tempDir := t.TempDir()
	secretsPath := filepath.Join(tempDir, "secrets.yaml")

	secretValue := "super-secret-password-12345"
	secretsYAML := `cluster:
  talosSecrets: "` + secretValue + `"
`
	if err := storage.WriteFile(secretsPath, []byte(secretsYAML), 0600); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	m := NewManager()

	// Try to get a non-existent key - error should not contain actual secret values
	_, err := m.GetSecret(secretsPath, "nonexistent.key")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Error message should not contain the secret value
	if strings.Contains(err.Error(), secretValue) {
		t.Errorf("error message leaked secret value: %v", err)
	}
}

// Test: SetSecret sets secrets correctly
func TestSetSecret(t *testing.T) {
	tests := []struct {
		name        string
		initialYAML string
		key         string
		value       string
		verifyFunc  func(t *testing.T, secretsPath string)
	}{
		{
			name: "set simple value",
			initialYAML: `cluster:
  talosSecrets: ""
`,
			key:   "cluster.talosSecrets",
			value: "new-secret-value",
			verifyFunc: func(t *testing.T, secretsPath string) {
				m := NewManager()
				got, err := m.GetSecret(secretsPath, "cluster.talosSecrets")
				if err != nil {
					t.Fatalf("verify failed: %v", err)
				}
				if got != "new-secret-value" {
					t.Errorf("got %q, want %q", got, "new-secret-value")
				}
			},
		},
		{
			name: "set nested value",
			initialYAML: `certManager:
  cloudflare:
    apiToken: ""
`,
			key:   "certManager.cloudflare.apiToken",
			value: "cf-token-xyz",
			verifyFunc: func(t *testing.T, secretsPath string) {
				m := NewManager()
				got, err := m.GetSecret(secretsPath, "certManager.cloudflare.apiToken")
				if err != nil {
					t.Fatalf("verify failed: %v", err)
				}
				if got != "cf-token-xyz" {
					t.Errorf("got %q, want %q", got, "cf-token-xyz")
				}
			},
		},
		{
			name: "update existing value",
			initialYAML: `cluster:
  talosSecrets: "old-secret"
`,
			key:   "cluster.talosSecrets",
			value: "new-secret",
			verifyFunc: func(t *testing.T, secretsPath string) {
				m := NewManager()
				got, err := m.GetSecret(secretsPath, "cluster.talosSecrets")
				if err != nil {
					t.Fatalf("verify failed: %v", err)
				}
				if got != "new-secret" {
					t.Errorf("got %q, want %q", got, "new-secret")
				}
			},
		},
		{
			name: "create new nested path",
			initialYAML: `cluster: {}
`,
			key:   "cluster.newSecret",
			value: "newValue",
			verifyFunc: func(t *testing.T, secretsPath string) {
				m := NewManager()
				got, err := m.GetSecret(secretsPath, "cluster.newSecret")
				if err != nil {
					t.Fatalf("verify failed: %v", err)
				}
				if got != "newValue" {
					t.Errorf("got %q, want %q", got, "newValue")
				}
			},
		},
		{
			name: "set value with special characters",
			initialYAML: `cluster:
  talosSecrets: ""
`,
			key:   "cluster.talosSecrets",
			value: `special"quotes'and\backslashes`,
			verifyFunc: func(t *testing.T, secretsPath string) {
				m := NewManager()
				got, err := m.GetSecret(secretsPath, "cluster.talosSecrets")
				if err != nil {
					t.Fatalf("verify failed: %v", err)
				}
				if got != `special"quotes'and\backslashes` {
					t.Errorf("got %q, want %q", got, `special"quotes'and\backslashes`)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			secretsPath := filepath.Join(tempDir, "secrets.yaml")

			if err := storage.WriteFile(secretsPath, []byte(tt.initialYAML), 0600); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			m := NewManager()
			if err := m.SetSecret(secretsPath, tt.key, tt.value); err != nil {
				t.Errorf("SetSecret failed: %v", err)
				return
			}

			// Verify the value was set correctly
			tt.verifyFunc(t, secretsPath)

			// Verify permissions remain secure (0600)
			info, err := os.Stat(secretsPath)
			if err != nil {
				t.Fatalf("failed to stat secrets file: %v", err)
			}
			if info.Mode().Perm() != 0600 {
				t.Errorf("permissions changed after SetSecret: got %o, want 0600", info.Mode().Perm())
			}
		})
	}
}

// Test: SetSecret error cases
func TestSetSecret_Errors(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) string
		key         string
		value       string
		errContains string
	}{
		{
			name: "non-existent file",
			setupFunc: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent.yaml")
			},
			key:         "cluster.talosSecrets",
			value:       "secret",
			errContains: "secrets file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secretsPath := tt.setupFunc(t)
			m := NewManager()

			err := m.SetSecret(secretsPath, tt.key, tt.value)
			if err == nil {
				t.Error("expected error, got nil")
			} else if !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
			}
		})
	}
}

// Test: SetSecret with concurrent access
func TestSetSecret_ConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	secretsPath := filepath.Join(tempDir, "secrets.yaml")

	initialYAML := `counter: "0"
`
	if err := storage.WriteFile(secretsPath, []byte(initialYAML), 0600); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	m := NewManager()
	const numGoroutines = 10

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	// Launch multiple goroutines trying to write different values
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			key := "counter"
			value := string(rune('0' + val))
			if err := m.SetSecret(secretsPath, key, value); err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check if any errors occurred
	for err := range errors {
		t.Errorf("concurrent write error: %v", err)
	}

	// Verify permissions remain secure after concurrent access
	info, err := os.Stat(secretsPath)
	if err != nil {
		t.Fatalf("failed to stat secrets file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("permissions changed after concurrent writes: got %o, want 0600", info.Mode().Perm())
	}

	// Verify we can read a value (should be one of the written values)
	value, err := m.GetSecret(secretsPath, "counter")
	if err != nil {
		t.Errorf("failed to read value after concurrent writes: %v", err)
	}
	if value == "" || value == "null" {
		t.Error("counter value is empty after concurrent writes")
	}
}

// Test: EnsureSecret generates and sets secret only when not set
func TestEnsureSecret(t *testing.T) {
	tests := []struct {
		name        string
		initialYAML string
		key         string
		length      int
		expectNew   bool
	}{
		{
			name: "generates secret when empty string",
			initialYAML: `cluster:
  talosSecrets: ""
`,
			key:       "cluster.talosSecrets",
			length:    32,
			expectNew: true,
		},
		{
			name: "generates secret when null",
			initialYAML: `cluster:
  talosSecrets: null
`,
			key:       "cluster.talosSecrets",
			length:    32,
			expectNew: true,
		},
		{
			name: "does not generate when secret exists",
			initialYAML: `cluster:
  talosSecrets: "existing-secret"
`,
			key:       "cluster.talosSecrets",
			length:    32,
			expectNew: false,
		},
		{
			name: "generates secret for non-existent key",
			initialYAML: `cluster: {}
`,
			key:       "cluster.newSecret",
			length:    64,
			expectNew: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			secretsPath := filepath.Join(tempDir, "secrets.yaml")

			if err := storage.WriteFile(secretsPath, []byte(tt.initialYAML), 0600); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			m := NewManager()

			// Get initial value if exists
			initialVal, _ := m.GetSecret(secretsPath, tt.key)

			// Call EnsureSecret
			secret, err := m.EnsureSecret(secretsPath, tt.key, tt.length)
			if err != nil {
				t.Errorf("EnsureSecret failed: %v", err)
				return
			}

			// Verify secret is returned
			if secret == "" {
				t.Error("EnsureSecret returned empty secret")
			}

			// Verify length
			if tt.expectNew && len(secret) != tt.length {
				t.Errorf("expected secret length %d, got %d", tt.length, len(secret))
			}

			// Get final value
			finalVal, err := m.GetSecret(secretsPath, tt.key)
			if err != nil {
				t.Fatalf("GetSecret failed: %v", err)
			}

			if tt.expectNew {
				// Should have generated new secret
				if initialVal != "" && finalVal == initialVal {
					t.Errorf("expected new secret, got same value: %q", finalVal)
				}
			} else {
				// Should have kept existing secret
				if finalVal != initialVal {
					t.Errorf("expected to keep existing secret %q, got %q", initialVal, finalVal)
				}
			}

			// Call EnsureSecret again - should be idempotent
			secret2, err := m.EnsureSecret(secretsPath, tt.key, tt.length)
			if err != nil {
				t.Errorf("second EnsureSecret failed: %v", err)
				return
			}

			// Secret should not change on second call
			if secret2 != secret {
				t.Errorf("secret changed on second ensure: %q -> %q", secret, secret2)
			}
		})
	}
}

// Test: GenerateAndStoreSecret convenience function
func TestGenerateAndStoreSecret(t *testing.T) {
	tempDir := t.TempDir()
	secretsPath := filepath.Join(tempDir, "secrets.yaml")

	initialYAML := `cluster:
  talosSecrets: ""
`
	if err := storage.WriteFile(secretsPath, []byte(initialYAML), 0600); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	m := NewManager()

	// Generate and store secret
	secret, err := m.GenerateAndStoreSecret(secretsPath, "cluster.talosSecrets")
	if err != nil {
		t.Fatalf("GenerateAndStoreSecret failed: %v", err)
	}

	// Verify secret length matches default
	if len(secret) != DefaultSecretLength {
		t.Errorf("expected length %d, got %d", DefaultSecretLength, len(secret))
	}

	// Verify secret is stored
	stored, err := m.GetSecret(secretsPath, "cluster.talosSecrets")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}

	if stored != secret {
		t.Errorf("stored secret %q does not match returned secret %q", stored, secret)
	}
}

// Test: DeleteSecret removes secrets correctly
func TestDeleteSecret(t *testing.T) {
	tests := []struct {
		name        string
		initialYAML string
		key         string
		wantErr     bool
	}{
		{
			name: "delete existing secret",
			initialYAML: `cluster:
  talosSecrets: "secret-to-delete"
  kubeconfig: "other-secret"
`,
			key:     "cluster.talosSecrets",
			wantErr: false,
		},
		{
			name: "delete nested secret",
			initialYAML: `certManager:
  cloudflare:
    apiToken: "token-to-delete"
`,
			key:     "certManager.cloudflare.apiToken",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			secretsPath := filepath.Join(tempDir, "secrets.yaml")

			if err := storage.WriteFile(secretsPath, []byte(tt.initialYAML), 0600); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			m := NewManager()

			// Verify secret exists before deletion
			_, err := m.GetSecret(secretsPath, tt.key)
			if err != nil {
				t.Fatalf("secret should exist before deletion: %v", err)
			}

			// Delete secret
			err = m.DeleteSecret(secretsPath, tt.key)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify secret no longer exists
			_, err = m.GetSecret(secretsPath, tt.key)
			if err == nil {
				t.Error("secret should not exist after deletion")
			}

			// Verify permissions remain secure (0600)
			info, err := os.Stat(secretsPath)
			if err != nil {
				t.Fatalf("failed to stat secrets file: %v", err)
			}
			if info.Mode().Perm() != 0600 {
				t.Errorf("permissions changed after DeleteSecret: got %o, want 0600", info.Mode().Perm())
			}
		})
	}
}

// Test: DeleteSecret error cases
func TestDeleteSecret_Errors(t *testing.T) {
	t.Run("non-existent file", func(t *testing.T) {
		tempDir := t.TempDir()
		secretsPath := filepath.Join(tempDir, "nonexistent.yaml")

		m := NewManager()
		err := m.DeleteSecret(secretsPath, "cluster.talosSecrets")

		if err == nil {
			t.Error("expected error, got nil")
		} else if !strings.Contains(err.Error(), "secrets file not found") {
			t.Errorf("error %q does not contain 'secrets file not found'", err.Error())
		}
	})
}

// Test: File permissions are always 0600
func TestEnsureSecretsFile_FilePermissions(t *testing.T) {
	tempDir := t.TempDir()
	m := NewManager()

	if err := m.EnsureSecretsFile(tempDir); err != nil {
		t.Fatalf("EnsureSecretsFile failed: %v", err)
	}

	secretsPath := filepath.Join(tempDir, "secrets.yaml")
	info, err := os.Stat(secretsPath)
	if err != nil {
		t.Fatalf("failed to stat secrets file: %v", err)
	}

	// Verify file has 0600 permissions (read/write for owner only)
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected permissions 0600, got %o", info.Mode().Perm())
	}
}

// Test: Idempotent secrets creation
func TestEnsureSecretsFile_Idempotent(t *testing.T) {
	tempDir := t.TempDir()
	m := NewManager()

	// First call creates secrets
	if err := m.EnsureSecretsFile(tempDir); err != nil {
		t.Fatalf("first EnsureSecretsFile failed: %v", err)
	}

	secretsPath := filepath.Join(tempDir, "secrets.yaml")
	firstContent, err := storage.ReadFile(secretsPath)
	if err != nil {
		t.Fatalf("failed to read secrets: %v", err)
	}

	// Second call should not modify secrets
	if err := m.EnsureSecretsFile(tempDir); err != nil {
		t.Fatalf("second EnsureSecretsFile failed: %v", err)
	}

	secondContent, err := storage.ReadFile(secretsPath)
	if err != nil {
		t.Fatalf("failed to read secrets: %v", err)
	}

	if string(firstContent) != string(secondContent) {
		t.Error("secrets content changed on second call")
	}
}

// Test: Secrets structure contains required fields
func TestEnsureSecretsFile_RequiredFields(t *testing.T) {
	tempDir := t.TempDir()
	m := NewManager()

	if err := m.EnsureSecretsFile(tempDir); err != nil {
		t.Fatalf("EnsureSecretsFile failed: %v", err)
	}

	secretsPath := filepath.Join(tempDir, "secrets.yaml")
	content, err := storage.ReadFile(secretsPath)
	if err != nil {
		t.Fatalf("failed to read secrets: %v", err)
	}

	contentStr := string(content)
	requiredFields := []string{
		"cluster:",
		"talosSecrets:",
		"kubeconfig:",
		"certManager:",
		"cloudflare:",
		"apiToken:",
	}

	for _, field := range requiredFields {
		if !strings.Contains(contentStr, field) {
			t.Errorf("secrets missing required field: %s", field)
		}
	}

	// Verify warning comment exists
	if !strings.Contains(contentStr, "WARNING") || !strings.Contains(contentStr, "sensitive") {
		t.Error("secrets file missing security warning comment")
	}
}

// Test: Secrets are more restrictive than config
func TestSecretsPermissions_MoreRestrictiveThanConfig(t *testing.T) {
	tempDir := t.TempDir()
	secretsPath := filepath.Join(tempDir, "secrets.yaml")
	configPath := filepath.Join(tempDir, "config.yaml")

	// Create secrets file
	m := NewManager()
	if err := m.EnsureSecretsFile(tempDir); err != nil {
		t.Fatalf("EnsureSecretsFile failed: %v", err)
	}

	// Create config file (typically 0644)
	configContent := `baseDomain: "example.com"`
	if err := storage.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Check permissions
	secretsInfo, err := os.Stat(secretsPath)
	if err != nil {
		t.Fatalf("failed to stat secrets: %v", err)
	}

	configInfo, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("failed to stat config: %v", err)
	}

	secretsPerm := secretsInfo.Mode().Perm()
	configPerm := configInfo.Mode().Perm()

	// Secrets (0600) should be more restrictive than config (0644)
	if secretsPerm >= configPerm {
		t.Errorf("secrets permissions %o should be more restrictive than config %o", secretsPerm, configPerm)
	}

	// Secrets should not be group or world readable
	if secretsPerm&0077 != 0 {
		t.Errorf("secrets file should not have group/world permissions, got %o", secretsPerm)
	}
}
