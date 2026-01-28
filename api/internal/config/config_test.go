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
			configYAML: `wildcloud:
  repository: "https://github.com/example/repo"
  currentPhase: "setup"
  completedPhases:
    - "phase1"
    - "phase2"
server:
  port: 8080
  host: "localhost"
operator:
  email: "admin@example.com"
cloud:
  dns:
    ip: "192.168.1.1"
    externalResolver: "8.8.8.8"
  router:
    ip: "192.168.1.254"
    dynamicDns: "example.dyndns.org"
  dnsmasq:
    interface: "eth0"
cluster:
  endpointIp: "192.168.1.100"
  nodes:
    talos:
      version: "v1.8.0"
`,
			verify: func(t *testing.T, config *GlobalConfig) {
				if config.Wildcloud.Repository != "https://github.com/example/repo" {
					t.Error("repository not loaded correctly")
				}
				if config.Server.Port != 8080 {
					t.Error("port not loaded correctly")
				}
				if config.Cloud.DNS.IP != "192.168.1.1" {
					t.Error("DNS IP not loaded correctly")
				}
				if config.Cluster.EndpointIP != "192.168.1.100" {
					t.Error("endpoint IP not loaded correctly")
				}
			},
			wantErr: false,
		},
		{
			name: "applies default values",
			configYAML: `cloud:
  dns:
    ip: "192.168.1.1"
cluster:
  nodes:
    talos:
      version: "v1.8.0"
`,
			verify: func(t *testing.T, config *GlobalConfig) {
				if config.Server.Port != 5055 {
					t.Errorf("default port not applied, got %d, want 5055", config.Server.Port)
				}
				if config.Server.Host != "0.0.0.0" {
					t.Errorf("default host not applied, got %q, want %q", config.Server.Host, "0.0.0.0")
				}
			},
			wantErr: false,
		},
		{
			name: "preserves custom port and host",
			configYAML: `server:
  port: 9000
  host: "127.0.0.1"
cloud:
  dns:
    ip: "192.168.1.1"
cluster:
  nodes:
    talos:
      version: "v1.8.0"
`,
			verify: func(t *testing.T, config *GlobalConfig) {
				if config.Server.Port != 9000 {
					t.Errorf("custom port not preserved, got %d, want 9000", config.Server.Port)
				}
				if config.Server.Host != "127.0.0.1" {
					t.Errorf("custom host not preserved, got %q, want %q", config.Server.Host, "127.0.0.1")
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
			config: &GlobalConfig{
				Wildcloud: struct {
					Repository      string   `yaml:"repository,omitempty" json:"repository,omitempty"`
					CurrentPhase    string   `yaml:"currentPhase,omitempty" json:"currentPhase,omitempty"`
					CompletedPhases []string `yaml:"completedPhases,omitempty" json:"completedPhases,omitempty"`
				}{
					Repository:      "https://github.com/example/repo",
					CurrentPhase:    "setup",
					CompletedPhases: []string{"phase1", "phase2"},
				},
				Server: struct {
					Port int    `yaml:"port,omitempty" json:"port,omitempty"`
					Host string `yaml:"host,omitempty" json:"host,omitempty"`
				}{
					Port: 8080,
					Host: "localhost",
				},
			},
			verify: func(t *testing.T, configPath string) {
				content, err := os.ReadFile(configPath)
				if err != nil {
					t.Fatalf("failed to read saved config: %v", err)
				}
				contentStr := string(content)
				if !strings.Contains(contentStr, "repository") {
					t.Error("saved config missing repository field")
				}
				if !strings.Contains(contentStr, "8080") {
					t.Error("saved config missing port value")
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
			name: "config with only DNS IP is empty",
			config: &GlobalConfig{
				Cloud: struct {
					DNS struct {
						IP               string `yaml:"ip,omitempty" json:"ip,omitempty"`
						ExternalResolver string `yaml:"externalResolver,omitempty" json:"externalResolver,omitempty"`
					} `yaml:"dns,omitempty" json:"dns,omitempty"`
					Router struct {
						IP         string `yaml:"ip,omitempty" json:"ip,omitempty"`
						DynamicDns string `yaml:"dynamicDns,omitempty" json:"dynamicDns,omitempty"`
					} `yaml:"router,omitempty" json:"router,omitempty"`
					Dnsmasq struct {
						Interface string `yaml:"interface,omitempty" json:"interface,omitempty"`
					} `yaml:"dnsmasq,omitempty" json:"dnsmasq,omitempty"`
				}{
					DNS: struct {
						IP               string `yaml:"ip,omitempty" json:"ip,omitempty"`
						ExternalResolver string `yaml:"externalResolver,omitempty" json:"externalResolver,omitempty"`
					}{
						IP: "192.168.1.1",
					},
				},
			},
			want: true,
		},
		{
			name: "config with only Talos version is empty",
			config: &GlobalConfig{
				Cluster: struct {
					EndpointIP string `yaml:"endpointIp,omitempty" json:"endpointIp,omitempty"`
					Nodes      struct {
						Talos struct {
							Version string `yaml:"version,omitempty" json:"version,omitempty"`
						} `yaml:"talos,omitempty" json:"talos,omitempty"`
					} `yaml:"nodes,omitempty" json:"nodes,omitempty"`
				}{
					Nodes: struct {
						Talos struct {
							Version string `yaml:"version,omitempty" json:"version,omitempty"`
						} `yaml:"talos,omitempty" json:"talos,omitempty"`
					}{
						Talos: struct {
							Version string `yaml:"version,omitempty" json:"version,omitempty"`
						}{
							Version: "v1.8.0",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "config with both DNS IP and Talos version is not empty",
			config: &GlobalConfig{
				Cloud: struct {
					DNS struct {
						IP               string `yaml:"ip,omitempty" json:"ip,omitempty"`
						ExternalResolver string `yaml:"externalResolver,omitempty" json:"externalResolver,omitempty"`
					} `yaml:"dns,omitempty" json:"dns,omitempty"`
					Router struct {
						IP         string `yaml:"ip,omitempty" json:"ip,omitempty"`
						DynamicDns string `yaml:"dynamicDns,omitempty" json:"dynamicDns,omitempty"`
					} `yaml:"router,omitempty" json:"router,omitempty"`
					Dnsmasq struct {
						Interface string `yaml:"interface,omitempty" json:"interface,omitempty"`
					} `yaml:"dnsmasq,omitempty" json:"dnsmasq,omitempty"`
				}{
					DNS: struct {
						IP               string `yaml:"ip,omitempty" json:"ip,omitempty"`
						ExternalResolver string `yaml:"externalResolver,omitempty" json:"externalResolver,omitempty"`
					}{
						IP: "192.168.1.1",
					},
				},
				Cluster: struct {
					EndpointIP string `yaml:"endpointIp,omitempty" json:"endpointIp,omitempty"`
					Nodes      struct {
						Talos struct {
							Version string `yaml:"version,omitempty" json:"version,omitempty"`
						} `yaml:"talos,omitempty" json:"talos,omitempty"`
					} `yaml:"nodes,omitempty" json:"nodes,omitempty"`
				}{
					Nodes: struct {
						Talos struct {
							Version string `yaml:"version,omitempty" json:"version,omitempty"`
						} `yaml:"talos,omitempty" json:"talos,omitempty"`
					}{
						Talos: struct {
							Version string `yaml:"version,omitempty" json:"version,omitempty"`
						}{
							Version: "v1.8.0",
						},
					},
				},
			},
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
  router:
    ip: "192.168.1.254"
  dns:
    ip: "192.168.1.1"
    externalResolver: "8.8.8.8"
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
			config: &InstanceConfig{
				Cloud: struct {
					Router struct {
						IP string `yaml:"ip" json:"ip"`
					} `yaml:"router" json:"router"`
					DNS struct {
						IP               string `yaml:"ip" json:"ip"`
						ExternalResolver string `yaml:"externalResolver" json:"externalResolver"`
					} `yaml:"dns" json:"dns"`
					DHCPRange string `yaml:"dhcpRange" json:"dhcpRange"`
					Dnsmasq   struct {
						Interface string `yaml:"interface" json:"interface"`
					} `yaml:"dnsmasq" json:"dnsmasq"`
					BaseDomain     string `yaml:"baseDomain" json:"baseDomain"`
					Domain         string `yaml:"domain" json:"domain"`
					InternalDomain string `yaml:"internalDomain" json:"internalDomain"`
					NFS            struct {
						MediaPath       string `yaml:"mediaPath" json:"mediaPath"`
						Host            string `yaml:"host" json:"host"`
						StorageCapacity string `yaml:"storageCapacity" json:"storageCapacity"`
					} `yaml:"nfs" json:"nfs"`
					DockerRegistryHost string `yaml:"dockerRegistryHost" json:"dockerRegistryHost"`
					Backup             struct {
						Root string `yaml:"root" json:"root"`
					} `yaml:"backup" json:"backup"`
				}{
					BaseDomain: "example.com",
					Domain:     "home",
				},
			},
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
	original := &GlobalConfig{
		Wildcloud: struct {
			Repository      string   `yaml:"repository,omitempty" json:"repository,omitempty"`
			CurrentPhase    string   `yaml:"currentPhase,omitempty" json:"currentPhase,omitempty"`
			CompletedPhases []string `yaml:"completedPhases,omitempty" json:"completedPhases,omitempty"`
		}{
			Repository:      "https://github.com/example/repo",
			CurrentPhase:    "setup",
			CompletedPhases: []string{"phase1", "phase2"},
		},
		Server: struct {
			Port int    `yaml:"port,omitempty" json:"port,omitempty"`
			Host string `yaml:"host,omitempty" json:"host,omitempty"`
		}{
			Port: 8080,
			Host: "localhost",
		},
		Operator: struct {
			Email string `yaml:"email,omitempty" json:"email,omitempty"`
		}{
			Email: "admin@example.com",
		},
	}

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
	if loaded.Wildcloud.Repository != original.Wildcloud.Repository {
		t.Errorf("repository mismatch: got %q, want %q", loaded.Wildcloud.Repository, original.Wildcloud.Repository)
	}
	if loaded.Server.Port != original.Server.Port {
		t.Errorf("port mismatch: got %d, want %d", loaded.Server.Port, original.Server.Port)
	}
	if loaded.Operator.Email != original.Operator.Email {
		t.Errorf("email mismatch: got %q, want %q", loaded.Operator.Email, original.Operator.Email)
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
