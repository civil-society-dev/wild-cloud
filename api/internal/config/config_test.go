package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test: LoadGlobalConfig loads valid configuration
func TestLoadGlobalConfig(t *testing.T) {
	tests := []struct {
		name       string
		configYAML string
		verify     func(t *testing.T, config *GlobalConfig)
		wantErr    bool
	}{
		{
			name: "loads complete configuration",
			configYAML: `operator:
  email: "admin@example.com"
cloud:
  router:
    ip: "192.168.1.254"
    dynamicDns: "example.dyndns.org"
  dnsmasq:
    ip: "192.168.1.1"
    interface: "eth0"
`,
			verify: func(t *testing.T, config *GlobalConfig) {
				if config.Operator.Email != "admin@example.com" {
					t.Error("operator email not loaded correctly")
				}
				if config.Cloud.Dnsmasq.IP != "192.168.1.1" {
					t.Error("DNS IP not loaded correctly")
				}
				if config.Cloud.Router.IP != "192.168.1.254" {
					t.Error("router IP not loaded correctly")
				}
				if config.Cloud.Dnsmasq.Interface != "eth0" {
					t.Error("dnsmasq interface not loaded correctly")
				}
			},
			wantErr: false,
		},
		{
			name: "loads minimal configuration",
			configYAML: `cloud:
  dnsmasq:
    ip: "192.168.1.1"
`,
			verify: func(t *testing.T, config *GlobalConfig) {
				if config.Cloud.Dnsmasq.IP != "192.168.1.1" {
					t.Error("DNS IP not loaded correctly")
				}
			},
			wantErr: false,
		},
		{
			name: "loads empty configuration",
			configYAML: `{}
`,
			verify: func(t *testing.T, config *GlobalConfig) {
				if config.Cloud.Dnsmasq.IP != "" {
					t.Error("expected empty DNS IP")
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "config.yaml")

			if err := os.WriteFile(configPath, []byte(tt.configYAML), 0644); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			config, err := LoadGlobalConfig(configPath)
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

			if config == nil {
				t.Fatal("config is nil")
			}

			if tt.verify != nil {
				tt.verify(t, config)
			}
		})
	}
}

// Test: LoadGlobalConfig error cases
func TestLoadGlobalConfig_Errors(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) string
		errContains string
	}{
		{
			name: "non-existent file",
			setupFunc: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent.yaml")
			},
			errContains: "reading config file",
		},
		{
			name: "invalid yaml",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				configPath := filepath.Join(tempDir, "config.yaml")
				content := `invalid: yaml: [[[`
				if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
				return configPath
			},
			errContains: "parsing config file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := tt.setupFunc(t)
			_, err := LoadGlobalConfig(configPath)

			if err == nil {
				t.Error("expected error, got nil")
			} else if !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
			}
		})
	}
}

// Test: SaveGlobalConfig saves configuration correctly
func TestSaveGlobalConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *GlobalConfig
		verify func(t *testing.T, configPath string)
	}{
		{
			name: "saves complete configuration",
			config: func() *GlobalConfig {
				cfg := &GlobalConfig{}
				cfg.Operator.Email = "admin@example.com"
				cfg.Cloud.Dnsmasq.IP = "192.168.1.1"
				cfg.Cloud.Router.IP = "192.168.1.254"
				cfg.Cloud.Dnsmasq.Interface = "eth0"
				return cfg
			}(),
			verify: func(t *testing.T, configPath string) {
				content, err := os.ReadFile(configPath)
				if err != nil {
					t.Fatalf("failed to read saved config: %v", err)
				}
				contentStr := string(content)
				if !strings.Contains(contentStr, "admin@example.com") {
					t.Error("saved config missing operator email")
				}
				if !strings.Contains(contentStr, "192.168.1.1") {
					t.Error("saved config missing DNS IP")
				}
			},
		},
		{
			name:   "saves empty configuration",
			config: &GlobalConfig{},
			verify: func(t *testing.T, configPath string) {
				if _, err := os.Stat(configPath); os.IsNotExist(err) {
					t.Error("config file not created")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "subdir", "config.yaml")

			err := SaveGlobalConfig(tt.config, configPath)
			if err != nil {
				t.Errorf("SaveGlobalConfig failed: %v", err)
				return
			}

			// Verify file exists
			if _, err := os.Stat(configPath); err != nil {
				t.Errorf("config file not created: %v", err)
				return
			}

			// Verify file permissions
			info, err := os.Stat(configPath)
			if err != nil {
				t.Fatalf("failed to stat config file: %v", err)
			}
			if info.Mode().Perm() != 0644 {
				t.Errorf("expected permissions 0644, got %v", info.Mode().Perm())
			}

			// Verify content can be loaded back
			loadedConfig, err := LoadGlobalConfig(configPath)
			if err != nil {
				t.Errorf("failed to reload saved config: %v", err)
			} else if loadedConfig == nil {
				t.Error("loaded config is nil")
			}

			if tt.verify != nil {
				tt.verify(t, configPath)
			}
		})
	}
}

// Test: SaveGlobalConfig creates directory
func TestSaveGlobalConfig_CreatesDirectory(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "nested", "dirs", "config.yaml")

	config := &GlobalConfig{}
	err := SaveGlobalConfig(config, configPath)
	if err != nil {
		t.Fatalf("SaveGlobalConfig failed: %v", err)
	}

	// Verify nested directories were created
	if _, err := os.Stat(filepath.Dir(configPath)); err != nil {
		t.Errorf("directory not created: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("config file not created: %v", err)
	}
}

// Test: GlobalConfig.IsEmpty checks if config is empty
func TestGlobalConfig_IsEmpty(t *testing.T) {
	tests := []struct {
		name   string
		config *GlobalConfig
		want   bool
	}{
		{
			name:   "nil config is empty",
			config: nil,
			want:   true,
		},
		{
			name:   "default config is empty",
			config: &GlobalConfig{},
			want:   true,
		},
		{
			name: "config with only DNS IP is not empty",
			config: func() *GlobalConfig {
				cfg := &GlobalConfig{}
				cfg.Cloud.Dnsmasq.IP = "192.168.1.1"
				return cfg
			}(),
			want: false,
		},
		{
			name: "config with only router IP is not empty",
			config: func() *GlobalConfig {
				cfg := &GlobalConfig{}
				cfg.Cloud.Router.IP = "192.168.1.254"
				return cfg
			}(),
			want: false,
		},
		{
			name: "config with only operator email is not empty",
			config: func() *GlobalConfig {
				cfg := &GlobalConfig{}
				cfg.Operator.Email = "admin@example.com"
				return cfg
			}(),
			want: false,
		},
		{
			name: "config with all fields is not empty",
			config: func() *GlobalConfig {
				cfg := &GlobalConfig{}
				cfg.Cloud.Dnsmasq.IP = "192.168.1.1"
				cfg.Cloud.Router.IP = "192.168.1.254"
				cfg.Operator.Email = "admin@example.com"
				return cfg
			}(),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.IsEmpty()
			if got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test: LoadCloudConfig loads instance configuration
func TestLoadCloudConfig(t *testing.T) {
	tests := []struct {
		name       string
		configYAML string
		verify     func(t *testing.T, config *InstanceConfig)
		wantErr    bool
	}{
		{
			name: "loads complete instance configuration",
			configYAML: `cloud:
  dhcpRange: "192.168.1.100,192.168.1.200"
  baseDomain: "example.com"
  domain: "home"
  internalDomain: "internal.example.com"
cluster:
  name: "my-cluster"
  loadBalancerIp: "192.168.1.10"
  nodes:
    talos:
      version: "v1.8.0"
    activeNodes:
      - node1:
          role: "control"
          interface: "eth0"
          disk: "/dev/sda"
`,
			verify: func(t *testing.T, config *InstanceConfig) {
				if config.Cloud.BaseDomain != "example.com" {
					t.Error("base domain not loaded correctly")
				}
				if config.Cloud.DHCPRange != "192.168.1.100,192.168.1.200" {
					t.Error("DHCP range not loaded correctly")
				}
				if config.Cluster.Name != "my-cluster" {
					t.Error("cluster name not loaded correctly")
				}
				if config.Cluster.Nodes.Talos.Version != "v1.8.0" {
					t.Error("talos version not loaded correctly")
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "config.yaml")

			if err := os.WriteFile(configPath, []byte(tt.configYAML), 0644); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			config, err := LoadCloudConfig(configPath)
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

			if config == nil {
				t.Fatal("config is nil")
			}

			if tt.verify != nil {
				tt.verify(t, config)
			}
		})
	}
}

// Test: LoadCloudConfig error cases
func TestLoadCloudConfig_Errors(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) string
		errContains string
	}{
		{
			name: "non-existent file",
			setupFunc: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent.yaml")
			},
			errContains: "reading config file",
		},
		{
			name: "invalid yaml",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				configPath := filepath.Join(tempDir, "config.yaml")
				content := `invalid: yaml: [[[`
				if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
				return configPath
			},
			errContains: "parsing config file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := tt.setupFunc(t)
			_, err := LoadCloudConfig(configPath)

			if err == nil {
				t.Error("expected error, got nil")
			} else if !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
			}
		})
	}
}

// Test: SaveCloudConfig saves instance configuration
func TestSaveCloudConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *InstanceConfig
		verify func(t *testing.T, configPath string)
	}{
		{
			name: "saves instance configuration",
			config: func() *InstanceConfig {
				cfg := &InstanceConfig{}
				cfg.Cloud.BaseDomain = "example.com"
				cfg.Cloud.Domain = "home"
				return cfg
			}(),
			verify: func(t *testing.T, configPath string) {
				content, err := os.ReadFile(configPath)
				if err != nil {
					t.Fatalf("failed to read saved config: %v", err)
				}
				contentStr := string(content)
				if !strings.Contains(contentStr, "example.com") {
					t.Error("saved config missing base domain")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "subdir", "config.yaml")

			err := SaveCloudConfig(tt.config, configPath)
			if err != nil {
				t.Errorf("SaveCloudConfig failed: %v", err)
				return
			}

			// Verify file exists
			if _, err := os.Stat(configPath); err != nil {
				t.Errorf("config file not created: %v", err)
				return
			}

			// Verify content can be loaded back
			loadedConfig, err := LoadCloudConfig(configPath)
			if err != nil {
				t.Errorf("failed to reload saved config: %v", err)
			} else if loadedConfig == nil {
				t.Error("loaded config is nil")
			}

			if tt.verify != nil {
				tt.verify(t, configPath)
			}
		})
	}
}

// Test: Round-trip save and load preserves data
func TestGlobalConfig_RoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Create config with all fields
	original := &GlobalConfig{}
	original.Operator.Email = "admin@example.com"
	original.Cloud.Dnsmasq.IP = "192.168.1.1"
	original.Cloud.Router.IP = "192.168.1.254"
	original.Cloud.Router.DynamicDns = "example.dyndns.org"
	original.Cloud.Dnsmasq.Interface = "eth0"

	// Save config
	if err := SaveGlobalConfig(original, configPath); err != nil {
		t.Fatalf("SaveGlobalConfig failed: %v", err)
	}

	// Load config
	loaded, err := LoadGlobalConfig(configPath)
	if err != nil {
		t.Fatalf("LoadGlobalConfig failed: %v", err)
	}

	// Verify all fields match
	if loaded.Operator.Email != original.Operator.Email {
		t.Errorf("email mismatch: got %q, want %q", loaded.Operator.Email, original.Operator.Email)
	}
	if loaded.Cloud.Dnsmasq.IP != original.Cloud.Dnsmasq.IP {
		t.Errorf("DNS IP mismatch: got %q, want %q", loaded.Cloud.Dnsmasq.IP, original.Cloud.Dnsmasq.IP)
	}
	if loaded.Cloud.Router.IP != original.Cloud.Router.IP {
		t.Errorf("router IP mismatch: got %q, want %q", loaded.Cloud.Router.IP, original.Cloud.Router.IP)
	}
	if loaded.Cloud.Dnsmasq.Interface != original.Cloud.Dnsmasq.Interface {
		t.Errorf("dnsmasq interface mismatch: got %q, want %q", loaded.Cloud.Dnsmasq.Interface, original.Cloud.Dnsmasq.Interface)
	}
}

// Test: Round-trip save and load for instance config
func TestInstanceConfig_RoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Create instance config
	original := &InstanceConfig{}
	original.Cloud.BaseDomain = "example.com"
	original.Cloud.Domain = "home"
	original.Cluster.Name = "my-cluster"

	// Save config
	if err := SaveCloudConfig(original, configPath); err != nil {
		t.Fatalf("SaveCloudConfig failed: %v", err)
	}

	// Load config
	loaded, err := LoadCloudConfig(configPath)
	if err != nil {
		t.Fatalf("LoadCloudConfig failed: %v", err)
	}

	// Verify fields match
	if loaded.Cloud.BaseDomain != original.Cloud.BaseDomain {
		t.Errorf("base domain mismatch: got %q, want %q", loaded.Cloud.BaseDomain, original.Cloud.BaseDomain)
	}
	if loaded.Cluster.Name != original.Cluster.Name {
		t.Errorf("cluster name mismatch: got %q, want %q", loaded.Cluster.Name, original.Cluster.Name)
	}
}
