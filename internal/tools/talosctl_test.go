package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewTalosctl(t *testing.T) {
	t.Run("creates Talosctl instance without config", func(t *testing.T) {
		tc := NewTalosctl()
		if tc == nil {
			t.Fatal("NewTalosctl() returned nil")
		}
		if tc.talosconfigPath != "" {
			t.Error("talosconfigPath should be empty for NewTalosctl()")
		}
	})

	t.Run("creates Talosctl instance with config", func(t *testing.T) {
		configPath := "/path/to/talosconfig"
		tc := NewTalosconfigWithConfig(configPath)
		if tc == nil {
			t.Fatal("NewTalosconfigWithConfig() returned nil")
		}
		if tc.talosconfigPath != configPath {
			t.Errorf("talosconfigPath = %q, want %q", tc.talosconfigPath, configPath)
		}
	})
}

func TestTalosconfigBuildArgs(t *testing.T) {
	tests := []struct {
		name           string
		talosconfigPath string
		baseArgs       []string
		wantPrefix     []string
	}{
		{
			name:           "no talosconfig adds no prefix",
			talosconfigPath: "",
			baseArgs:       []string{"version", "--short"},
			wantPrefix:     nil,
		},
		{
			name:           "with talosconfig adds prefix",
			talosconfigPath: "/path/to/talosconfig",
			baseArgs:       []string{"version", "--short"},
			wantPrefix:     []string{"--talosconfig", "/path/to/talosconfig"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := &Talosctl{talosconfigPath: tt.talosconfigPath}
			got := tc.buildArgs(tt.baseArgs)

			if tt.wantPrefix == nil {
				// Should return baseArgs unchanged
				if len(got) != len(tt.baseArgs) {
					t.Errorf("buildArgs() length = %d, want %d", len(got), len(tt.baseArgs))
				}
				for i, arg := range tt.baseArgs {
					if i >= len(got) || got[i] != arg {
						t.Errorf("buildArgs()[%d] = %q, want %q", i, got[i], arg)
					}
				}
			} else {
				// Should have prefix + baseArgs
				expectedLen := len(tt.wantPrefix) + len(tt.baseArgs)
				if len(got) != expectedLen {
					t.Errorf("buildArgs() length = %d, want %d", len(got), expectedLen)
				}
				// Check prefix
				for i, arg := range tt.wantPrefix {
					if i >= len(got) || got[i] != arg {
						t.Errorf("buildArgs() prefix[%d] = %q, want %q", i, got[i], arg)
					}
				}
				// Check baseArgs follow prefix
				for i, arg := range tt.baseArgs {
					idx := len(tt.wantPrefix) + i
					if idx >= len(got) || got[idx] != arg {
						t.Errorf("buildArgs()[%d] = %q, want %q", idx, got[idx], arg)
					}
				}
			}
		})
	}
}

func TestTalosconfigGenConfig(t *testing.T) {
	tests := []struct {
		name        string
		clusterName string
		endpoint    string
		outputDir   string
		skipTest    bool
	}{
		{
			name:        "gen config with valid params",
			clusterName: "test-cluster",
			endpoint:    "https://192.168.1.100:6443",
			outputDir:   "testdata",
			skipTest:    true, // Skip actual execution
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipTest {
				t.Skip("Skipping test that requires talosctl binary")
			}

			tmpDir := t.TempDir()
			tc := NewTalosctl()
			err := tc.GenConfig(tt.clusterName, tt.endpoint, tmpDir)

			// This will fail without talosctl, but tests the method signature
			if err == nil {
				// If it somehow succeeds, verify files were created
				expectedFiles := []string{
					"controlplane.yaml",
					"worker.yaml",
					"talosconfig",
				}
				for _, file := range expectedFiles {
					path := filepath.Join(tmpDir, file)
					if _, err := os.Stat(path); os.IsNotExist(err) {
						t.Errorf("Expected file not created: %s", file)
					}
				}
			}
		})
	}
}

func TestTalosconfigApplyConfig(t *testing.T) {
	tests := []struct {
		name             string
		nodeIP           string
		configFile       string
		insecure         bool
		talosconfigPath  string
		skipTest         bool
	}{
		{
			name:       "apply config with all params",
			nodeIP:     "192.168.1.100",
			configFile: "/path/to/config.yaml",
			insecure:   true,
			skipTest:   true,
		},
		{
			name:            "apply config with talosconfig",
			nodeIP:          "192.168.1.100",
			configFile:      "/path/to/config.yaml",
			insecure:        false,
			talosconfigPath: "/path/to/talosconfig",
			skipTest:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipTest {
				t.Skip("Skipping test that requires talosctl binary")
			}

			tc := NewTalosctl()
			err := tc.ApplyConfig(tt.nodeIP, tt.configFile, tt.insecure, tt.talosconfigPath)

			// Will fail without talosctl, but tests method signature
			_ = err
		})
	}
}

func TestTalosconfigGetDisks(t *testing.T) {
	tests := []struct {
		name     string
		nodeIP   string
		insecure bool
		skipTest bool
	}{
		{
			name:     "get disks in insecure mode",
			nodeIP:   "192.168.1.100",
			insecure: true,
			skipTest: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipTest {
				t.Skip("Skipping test that requires talosctl binary and running node")
			}

			tc := NewTalosctl()
			disks, err := tc.GetDisks(tt.nodeIP, tt.insecure)

			if err == nil {
				// If successful, verify return type
				if disks == nil {
					t.Error("GetDisks() returned nil slice without error")
				}
				// Each disk should have path and size
				for i, disk := range disks {
					if disk.Path == "" {
						t.Errorf("disk[%d].Path is empty", i)
					}
					if disk.Size <= 0 {
						t.Errorf("disk[%d].Size = %d, want > 0", i, disk.Size)
					}
					// Size should be > 10GB per filtering
					if disk.Size <= 10000000000 {
						t.Errorf("disk[%d].Size = %d, should be filtered (> 10GB)", i, disk.Size)
					}
				}
			}
		})
	}
}

func TestTalosconfigGetLinks(t *testing.T) {
	tests := []struct {
		name     string
		nodeIP   string
		insecure bool
		skipTest bool
	}{
		{
			name:     "get links in insecure mode",
			nodeIP:   "192.168.1.100",
			insecure: true,
			skipTest: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipTest {
				t.Skip("Skipping test that requires talosctl binary and running node")
			}

			tc := NewTalosctl()
			links, err := tc.GetLinks(tt.nodeIP, tt.insecure)

			if err == nil {
				if links == nil {
					t.Error("GetLinks() returned nil slice without error")
				}
			}
		})
	}
}

func TestTalosconfigGetRoutes(t *testing.T) {
	tests := []struct {
		name     string
		nodeIP   string
		insecure bool
		skipTest bool
	}{
		{
			name:     "get routes in insecure mode",
			nodeIP:   "192.168.1.100",
			insecure: true,
			skipTest: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipTest {
				t.Skip("Skipping test that requires talosctl binary and running node")
			}

			tc := NewTalosctl()
			routes, err := tc.GetRoutes(tt.nodeIP, tt.insecure)

			if err == nil {
				if routes == nil {
					t.Error("GetRoutes() returned nil slice without error")
				}
			}
		})
	}
}

func TestTalosconfigGetDefaultInterface(t *testing.T) {
	tests := []struct {
		name     string
		nodeIP   string
		insecure bool
		skipTest bool
	}{
		{
			name:     "get default interface",
			nodeIP:   "192.168.1.100",
			insecure: true,
			skipTest: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipTest {
				t.Skip("Skipping test that requires talosctl binary and running node")
			}

			tc := NewTalosctl()
			iface, err := tc.GetDefaultInterface(tt.nodeIP, tt.insecure)

			if err == nil {
				if iface == "" {
					t.Error("GetDefaultInterface() returned empty string without error")
				}
			}
		})
	}
}

func TestTalosconfigGetPhysicalInterface(t *testing.T) {
	tests := []struct {
		name     string
		nodeIP   string
		insecure bool
		skipTest bool
	}{
		{
			name:     "get physical interface",
			nodeIP:   "192.168.1.100",
			insecure: true,
			skipTest: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipTest {
				t.Skip("Skipping test that requires talosctl binary and running node")
			}

			tc := NewTalosctl()
			iface, err := tc.GetPhysicalInterface(tt.nodeIP, tt.insecure)

			if err == nil {
				if iface == "" {
					t.Error("GetPhysicalInterface() returned empty string without error")
				}
				// Should not be loopback
				if iface == "lo" {
					t.Error("GetPhysicalInterface() returned loopback interface")
				}
			}
		})
	}
}

func TestTalosconfigGetVersion(t *testing.T) {
	tests := []struct {
		name     string
		nodeIP   string
		insecure bool
		want     string // Expected for maintenance mode or version string
		skipTest bool
	}{
		{
			name:     "get version in insecure mode",
			nodeIP:   "192.168.1.100",
			insecure: true,
			skipTest: true,
		},
		{
			name:     "get version in secure mode",
			nodeIP:   "192.168.1.100",
			insecure: false,
			skipTest: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipTest {
				t.Skip("Skipping test that requires talosctl binary and running node")
			}

			tc := NewTalosctl()
			version, err := tc.GetVersion(tt.nodeIP, tt.insecure)

			if err == nil {
				if version == "" {
					t.Error("GetVersion() returned empty string without error")
				}
				// Version should be either "maintenance" or start with "v"
				if version != "maintenance" && version[0] != 'v' {
					t.Errorf("GetVersion() = %q, expected 'maintenance' or version starting with 'v'", version)
				}
			}
		})
	}
}

func TestTalosconfigValidate(t *testing.T) {
	t.Run("validate checks for talosctl", func(t *testing.T) {
		tc := NewTalosctl()
		err := tc.Validate()

		// This will pass if talosctl is installed, fail otherwise
		// We can't guarantee talosctl is installed in all test environments
		_ = err
	})
}

func TestDiskInfoStruct(t *testing.T) {
	t.Run("DiskInfo has required fields", func(t *testing.T) {
		disk := DiskInfo{
			Path: "/dev/sda",
			Size: 1000000000000, // 1TB
		}

		if disk.Path != "/dev/sda" {
			t.Errorf("Path = %q, want %q", disk.Path, "/dev/sda")
		}
		if disk.Size != 1000000000000 {
			t.Errorf("Size = %d, want %d", disk.Size, 1000000000000)
		}
	})
}

func TestTalosconfigResourceJSONParsing(t *testing.T) {
	// This test verifies the logic of getResourceJSON without actually calling talosctl
	t.Run("getResourceJSON uses correct command structure", func(t *testing.T) {
		tc := &Talosctl{talosconfigPath: "/path/to/talosconfig"}

		// We can't easily test the actual command execution without mocking,
		// but we can verify buildArgs works correctly
		baseArgs := []string{"get", "disks", "--nodes", "192.168.1.100", "-o", "json"}
		finalArgs := tc.buildArgs(baseArgs)

		// Should have talosconfig prepended
		if len(finalArgs) < 2 || finalArgs[0] != "--talosconfig" {
			t.Error("buildArgs() should prepend --talosconfig")
		}
	})
}

func TestTalosconfigInterfaceFiltering(t *testing.T) {
	// Test the logic for filtering physical interfaces
	tests := []struct {
		name          string
		interfaceName string
		linkType      string
		operState     string
		shouldAccept  bool
	}{
		{
			name:          "eth0 up and ethernet",
			interfaceName: "eth0",
			linkType:      "ether",
			operState:     "up",
			shouldAccept:  true,
		},
		{
			name:          "eno1 up and ethernet",
			interfaceName: "eno1",
			linkType:      "ether",
			operState:     "up",
			shouldAccept:  true,
		},
		{
			name:          "loopback should be filtered",
			interfaceName: "lo",
			linkType:      "loopback",
			operState:     "up",
			shouldAccept:  false,
		},
		{
			name:          "cni interface should be filtered",
			interfaceName: "cni0",
			linkType:      "ether",
			operState:     "up",
			shouldAccept:  false,
		},
		{
			name:          "flannel interface should be filtered",
			interfaceName: "flannel.1",
			linkType:      "ether",
			operState:     "up",
			shouldAccept:  false,
		},
		{
			name:          "docker interface should be filtered",
			interfaceName: "docker0",
			linkType:      "ether",
			operState:     "up",
			shouldAccept:  false,
		},
		{
			name:          "bridge interface should be filtered",
			interfaceName: "br-1234",
			linkType:      "ether",
			operState:     "up",
			shouldAccept:  false,
		},
		{
			name:          "veth interface should be filtered",
			interfaceName: "veth123",
			linkType:      "ether",
			operState:     "up",
			shouldAccept:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This simulates the filtering logic in GetPhysicalInterface
			id := tt.interfaceName
			linkType := tt.linkType
			operState := tt.operState

			shouldAccept := (linkType == "ether" && operState == "up" &&
				id != "lo" &&
				(id[:3] == "eth" || id[:2] == "en") &&
				!containsAny(id, []string{"cni", "flannel", "docker", "br-", "veth"}))

			if shouldAccept != tt.shouldAccept {
				t.Errorf("Interface %q filtering = %v, want %v", id, shouldAccept, tt.shouldAccept)
			}
		})
	}
}

// Helper function for interface filtering test
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if len(substr) > 0 {
			if substr[len(substr)-1] == '-' {
				// Prefix match for things like "br-"
				if len(s) >= len(substr) && s[:len(substr)] == substr {
					return true
				}
			} else {
				// Contains match
				if len(s) >= len(substr) {
					for i := 0; i <= len(s)-len(substr); i++ {
						if s[i:i+len(substr)] == substr {
							return true
						}
					}
				}
			}
		}
	}
	return false
}
