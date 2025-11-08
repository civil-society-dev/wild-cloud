package config

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/wild-cloud/wild-central/daemon/internal/storage"
)

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

// Test: EnsureInstanceConfig creates config file with proper structure
func TestEnsureInstanceConfig(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T, instancePath string)
		wantErr     bool
		errContains string
	}{
		{
			name:      "creates config when not exists",
			setupFunc: nil,
			wantErr:   false,
		},
		{
			name: "returns nil when config exists",
			setupFunc: func(t *testing.T, instancePath string) {
				configPath := filepath.Join(instancePath, "config.yaml")
				content := `baseDomain: "test.local"
domain: "test"
internalDomain: "internal.test"
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
				if err := storage.WriteFile(configPath, []byte(content), 0644); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			wantErr: false,
		},
		{
			name: "returns error when config is invalid yaml",
			setupFunc: func(t *testing.T, instancePath string) {
				configPath := filepath.Join(instancePath, "config.yaml")
				content := `invalid: yaml: content: [[[`
				if err := storage.WriteFile(configPath, []byte(content), 0644); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			wantErr:     true,
			errContains: "invalid config file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instancePath := t.TempDir()
			m := NewManager()

			if tt.setupFunc != nil {
				tt.setupFunc(t, instancePath)
			}

			err := m.EnsureInstanceConfig(instancePath)
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

			// Verify config file exists
			configPath := filepath.Join(instancePath, "config.yaml")
			if !storage.FileExists(configPath) {
				t.Error("config file not created")
			}

			// Verify config is valid YAML
			if err := m.ValidateConfig(configPath); err != nil {
				t.Errorf("config validation failed: %v", err)
			}

			// Verify config has expected structure
			content, err := storage.ReadFile(configPath)
			if err != nil {
				t.Fatalf("failed to read config: %v", err)
			}
			contentStr := string(content)
			requiredFields := []string{"baseDomain:", "domain:", "cluster:", "backup:", "nfs:"}
			for _, field := range requiredFields {
				if !strings.Contains(contentStr, field) {
					t.Errorf("config missing required field: %s", field)
				}
			}
		})
	}
}

// Test: GetConfigValue retrieves values correctly
func TestGetConfigValue(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		key         string
		want        string
		wantErr     bool
		errContains string
	}{
		{
			name: "get simple string value",
			configYAML: `baseDomain: "example.com"
domain: "test"
`,
			key:     "baseDomain",
			want:    "example.com",
			wantErr: false,
		},
		{
			name: "get nested value with dot notation",
			configYAML: `cluster:
  name: "my-cluster"
  nodes:
    talos:
      version: "v1.8.0"
`,
			key:     "cluster.nodes.talos.version",
			want:    "v1.8.0",
			wantErr: false,
		},
		{
			name: "get empty string value",
			configYAML: `baseDomain: ""
`,
			key:     "baseDomain",
			want:    "",
			wantErr: false,
		},
		{
			name: "get non-existent key returns null",
			configYAML: `baseDomain: "example.com"
`,
			key:     "nonexistent",
			want:    "null",
			wantErr: false,
		},
		{
			name: "get from array",
			configYAML: `cluster:
  nodes:
    activeNodes:
      - "node1"
      - "node2"
`,
			key:     "cluster.nodes.activeNodes.[0]",
			want:    "node1",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "config.yaml")

			if err := storage.WriteFile(configPath, []byte(tt.configYAML), 0644); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			m := NewManager()
			got, err := m.GetConfigValue(configPath, tt.key)

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

// Test: GetConfigValue error cases
func TestGetConfigValue_Errors(t *testing.T) {
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
			key:         "baseDomain",
			errContains: "config file not found",
		},
		{
			name: "malformed yaml",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				configPath := filepath.Join(tempDir, "config.yaml")
				content := `invalid: yaml: [[[`
				if err := storage.WriteFile(configPath, []byte(content), 0644); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
				return configPath
			},
			key:         "baseDomain",
			errContains: "getting config value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := tt.setupFunc(t)
			m := NewManager()

			_, err := m.GetConfigValue(configPath, tt.key)
			if err == nil {
				t.Error("expected error, got nil")
			} else if !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
			}
		})
	}
}

// Test: SetConfigValue sets values correctly
func TestSetConfigValue(t *testing.T) {
	tests := []struct {
		name        string
		initialYAML string
		key         string
		value       string
		verifyFunc  func(t *testing.T, configPath string)
	}{
		{
			name: "set simple value",
			initialYAML: `baseDomain: ""
domain: ""
`,
			key:   "baseDomain",
			value: "example.com",
			verifyFunc: func(t *testing.T, configPath string) {
				m := NewManager()
				got, err := m.GetConfigValue(configPath, "baseDomain")
				if err != nil {
					t.Fatalf("verify failed: %v", err)
				}
				if got != "example.com" {
					t.Errorf("got %q, want %q", got, "example.com")
				}
			},
		},
		{
			name: "set nested value",
			initialYAML: `cluster:
  name: ""
  nodes:
    talos:
      version: ""
`,
			key:   "cluster.nodes.talos.version",
			value: "v1.8.0",
			verifyFunc: func(t *testing.T, configPath string) {
				m := NewManager()
				got, err := m.GetConfigValue(configPath, "cluster.nodes.talos.version")
				if err != nil {
					t.Fatalf("verify failed: %v", err)
				}
				if got != "v1.8.0" {
					t.Errorf("got %q, want %q", got, "v1.8.0")
				}
			},
		},
		{
			name: "update existing value",
			initialYAML: `baseDomain: "old.com"
`,
			key:   "baseDomain",
			value: "new.com",
			verifyFunc: func(t *testing.T, configPath string) {
				m := NewManager()
				got, err := m.GetConfigValue(configPath, "baseDomain")
				if err != nil {
					t.Fatalf("verify failed: %v", err)
				}
				if got != "new.com" {
					t.Errorf("got %q, want %q", got, "new.com")
				}
			},
		},
		{
			name: "create new nested path",
			initialYAML: `cluster: {}
`,
			key:   "cluster.newField",
			value: "newValue",
			verifyFunc: func(t *testing.T, configPath string) {
				m := NewManager()
				got, err := m.GetConfigValue(configPath, "cluster.newField")
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
			initialYAML: `baseDomain: ""
`,
			key:   "baseDomain",
			value: `special"quotes'and\backslashes`,
			verifyFunc: func(t *testing.T, configPath string) {
				m := NewManager()
				got, err := m.GetConfigValue(configPath, "baseDomain")
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
			configPath := filepath.Join(tempDir, "config.yaml")

			if err := storage.WriteFile(configPath, []byte(tt.initialYAML), 0644); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			m := NewManager()
			if err := m.SetConfigValue(configPath, tt.key, tt.value); err != nil {
				t.Errorf("SetConfigValue failed: %v", err)
				return
			}

			// Verify the value was set correctly
			tt.verifyFunc(t, configPath)

			// Verify config is still valid YAML
			if err := m.ValidateConfig(configPath); err != nil {
				t.Errorf("config validation failed after set: %v", err)
			}
		})
	}
}

// Test: SetConfigValue error cases
func TestSetConfigValue_Errors(t *testing.T) {
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
			key:         "baseDomain",
			value:       "example.com",
			errContains: "config file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := tt.setupFunc(t)
			m := NewManager()

			err := m.SetConfigValue(configPath, tt.key, tt.value)
			if err == nil {
				t.Error("expected error, got nil")
			} else if !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
			}
		})
	}
}

// Test: SetConfigValue with concurrent access
func TestSetConfigValue_ConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	initialYAML := `counter: "0"
`
	if err := storage.WriteFile(configPath, []byte(initialYAML), 0644); err != nil {
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
			if err := m.SetConfigValue(configPath, key, value); err != nil {
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

	// Verify config is still valid after concurrent access
	if err := m.ValidateConfig(configPath); err != nil {
		t.Errorf("config validation failed after concurrent writes: %v", err)
	}

	// Verify we can read the value (should be one of the written values)
	value, err := m.GetConfigValue(configPath, "counter")
	if err != nil {
		t.Errorf("failed to read value after concurrent writes: %v", err)
	}
	if value == "" || value == "null" {
		t.Error("counter value is empty after concurrent writes")
	}
}

// Test: EnsureConfigValue sets value only when not set
func TestEnsureConfigValue(t *testing.T) {
	tests := []struct {
		name        string
		initialYAML string
		key         string
		value       string
		expectSet   bool
	}{
		{
			name: "sets value when empty string",
			initialYAML: `baseDomain: ""
`,
			key:       "baseDomain",
			value:     "example.com",
			expectSet: true,
		},
		{
			name: "sets value when null",
			initialYAML: `baseDomain: null
`,
			key:       "baseDomain",
			value:     "example.com",
			expectSet: true,
		},
		{
			name: "does not set value when already set",
			initialYAML: `baseDomain: "existing.com"
`,
			key:       "baseDomain",
			value:     "new.com",
			expectSet: false,
		},
		{
			name: "sets value when key does not exist",
			initialYAML: `domain: "test"
`,
			key:       "baseDomain",
			value:     "example.com",
			expectSet: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "config.yaml")

			if err := storage.WriteFile(configPath, []byte(tt.initialYAML), 0644); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			m := NewManager()

			// Get initial value
			initialVal, _ := m.GetConfigValue(configPath, tt.key)

			// Call EnsureConfigValue
			if err := m.EnsureConfigValue(configPath, tt.key, tt.value); err != nil {
				t.Errorf("EnsureConfigValue failed: %v", err)
				return
			}

			// Get final value
			finalVal, err := m.GetConfigValue(configPath, tt.key)
			if err != nil {
				t.Fatalf("GetConfigValue failed: %v", err)
			}

			if tt.expectSet {
				if finalVal != tt.value {
					t.Errorf("expected value to be set to %q, got %q", tt.value, finalVal)
				}
			} else {
				if finalVal != initialVal {
					t.Errorf("expected value to remain %q, got %q", initialVal, finalVal)
				}
			}

			// Call EnsureConfigValue again - should be idempotent
			if err := m.EnsureConfigValue(configPath, tt.key, "different.com"); err != nil {
				t.Errorf("second EnsureConfigValue failed: %v", err)
				return
			}

			// Value should not change on second call
			secondVal, err := m.GetConfigValue(configPath, tt.key)
			if err != nil {
				t.Fatalf("GetConfigValue failed: %v", err)
			}
			if secondVal != finalVal {
				t.Errorf("value changed on second ensure: %q -> %q", finalVal, secondVal)
			}
		})
	}
}

// Test: ValidateConfig validates YAML correctly
func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid yaml",
			configYAML: `baseDomain: "example.com"
domain: "test"
cluster:
  name: "my-cluster"
`,
			wantErr: false,
		},
		{
			name:        "invalid yaml - bad indentation",
			configYAML:  `baseDomain: "example.com"\n  domain: "test"`,
			wantErr:     true,
			errContains: "yaml validation failed",
		},
		{
			name:        "invalid yaml - unclosed bracket",
			configYAML:  `cluster: { name: "test"`,
			wantErr:     true,
			errContains: "yaml validation failed",
		},
		{
			name:       "empty file",
			configYAML: "",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "config.yaml")

			if err := storage.WriteFile(configPath, []byte(tt.configYAML), 0644); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			m := NewManager()
			err := m.ValidateConfig(configPath)

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
			}
		})
	}
}

// Test: ValidateConfig error cases
func TestValidateConfig_Errors(t *testing.T) {
	t.Run("non-existent file", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "nonexistent.yaml")

		m := NewManager()
		err := m.ValidateConfig(configPath)

		if err == nil {
			t.Error("expected error, got nil")
		} else if !strings.Contains(err.Error(), "config file not found") {
			t.Errorf("error %q does not contain 'config file not found'", err.Error())
		}
	})
}

// Test: CopyConfig copies configuration correctly
func TestCopyConfig(t *testing.T) {
	tests := []struct {
		name        string
		srcYAML     string
		setupDst    func(t *testing.T, dstPath string)
		wantErr     bool
		errContains string
	}{
		{
			name: "copies config successfully",
			srcYAML: `baseDomain: "example.com"
domain: "test"
cluster:
  name: "my-cluster"
`,
			setupDst: nil,
			wantErr:  false,
		},
		{
			name:        "creates destination directory",
			srcYAML:     `baseDomain: "example.com"`,
			setupDst:    nil,
			wantErr:     false,
		},
		{
			name: "overwrites existing destination",
			srcYAML: `baseDomain: "new.com"
`,
			setupDst: func(t *testing.T, dstPath string) {
				oldContent := `baseDomain: "old.com"`
				if err := storage.WriteFile(dstPath, []byte(oldContent), 0644); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			srcPath := filepath.Join(tempDir, "source.yaml")
			dstPath := filepath.Join(tempDir, "subdir", "dest.yaml")

			// Create source file
			if err := storage.WriteFile(srcPath, []byte(tt.srcYAML), 0644); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			// Setup destination if needed
			if tt.setupDst != nil {
				if err := storage.EnsureDir(filepath.Dir(dstPath), 0755); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
				tt.setupDst(t, dstPath)
			}

			m := NewManager()
			err := m.CopyConfig(srcPath, dstPath)

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

			// Verify destination file exists
			if !storage.FileExists(dstPath) {
				t.Error("destination file not created")
			}

			// Verify content matches source
			srcContent, err := storage.ReadFile(srcPath)
			if err != nil {
				t.Fatalf("failed to read source: %v", err)
			}
			dstContent, err := storage.ReadFile(dstPath)
			if err != nil {
				t.Fatalf("failed to read destination: %v", err)
			}

			if string(srcContent) != string(dstContent) {
				t.Error("destination content does not match source")
			}

			// Verify destination is valid YAML
			if err := m.ValidateConfig(dstPath); err != nil {
				t.Errorf("destination config validation failed: %v", err)
			}
		})
	}
}

// Test: CopyConfig error cases
func TestCopyConfig_Errors(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T, tempDir string) (srcPath, dstPath string)
		errContains string
	}{
		{
			name: "source file does not exist",
			setupFunc: func(t *testing.T, tempDir string) (string, string) {
				return filepath.Join(tempDir, "nonexistent.yaml"),
					filepath.Join(tempDir, "dest.yaml")
			},
			errContains: "source config file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			srcPath, dstPath := tt.setupFunc(t, tempDir)

			m := NewManager()
			err := m.CopyConfig(srcPath, dstPath)

			if err == nil {
				t.Error("expected error, got nil")
			} else if !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
			}
		})
	}
}

// Test: File permissions are preserved
func TestEnsureInstanceConfig_FilePermissions(t *testing.T) {
	tempDir := t.TempDir()
	m := NewManager()

	if err := m.EnsureInstanceConfig(tempDir); err != nil {
		t.Fatalf("EnsureInstanceConfig failed: %v", err)
	}

	configPath := filepath.Join(tempDir, "config.yaml")
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("failed to stat config file: %v", err)
	}

	// Verify file has 0644 permissions
	if info.Mode().Perm() != 0644 {
		t.Errorf("expected permissions 0644, got %v", info.Mode().Perm())
	}
}

// Test: Idempotent config creation
func TestEnsureInstanceConfig_Idempotent(t *testing.T) {
	tempDir := t.TempDir()
	m := NewManager()

	// First call creates config
	if err := m.EnsureInstanceConfig(tempDir); err != nil {
		t.Fatalf("first EnsureInstanceConfig failed: %v", err)
	}

	configPath := filepath.Join(tempDir, "config.yaml")
	firstContent, err := storage.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	// Second call should not modify config
	if err := m.EnsureInstanceConfig(tempDir); err != nil {
		t.Fatalf("second EnsureInstanceConfig failed: %v", err)
	}

	secondContent, err := storage.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	if string(firstContent) != string(secondContent) {
		t.Error("config content changed on second call")
	}
}

// Test: Config structure contains all required fields
func TestEnsureInstanceConfig_RequiredFields(t *testing.T) {
	tempDir := t.TempDir()
	m := NewManager()

	if err := m.EnsureInstanceConfig(tempDir); err != nil {
		t.Fatalf("EnsureInstanceConfig failed: %v", err)
	}

	configPath := filepath.Join(tempDir, "config.yaml")
	content, err := storage.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	contentStr := string(content)
	requiredFields := []string{
		"baseDomain:",
		"domain:",
		"internalDomain:",
		"dhcpRange:",
		"backup:",
		"nfs:",
		"cluster:",
		"loadBalancerIp:",
		"ipAddressPool:",
		"hostnamePrefix:",
		"certManager:",
		"externalDns:",
		"nodes:",
		"talos:",
		"version:",
		"schematicId:",
		"control:",
		"vip:",
		"activeNodes:",
	}

	for _, field := range requiredFields {
		if !strings.Contains(contentStr, field) {
			t.Errorf("config missing required field: %s", field)
		}
	}
}
