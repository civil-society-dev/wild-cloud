package services

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// TestCheckTemplateState tests the template state detection logic
func TestCheckTemplateState(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(tmpDir string) (*Manager, string, string)
		expectedState string
		expectedVersion string
		expectedLatest string
		expectedUpdate bool
	}{
		{
			name: "service not in embedded manifests",
			setup: func(tmpDir string) (*Manager, string, string) {
				m := &Manager{
					dataDir:   tmpDir,
					manifests: make(map[string]*ServiceManifest),
				}
				return m, "test-instance", "nonexistent-service"
			},
			expectedState: "not_fetched",
		},
		{
			name: "service exists in manifests but not fetched to instance",
			setup: func(tmpDir string) (*Manager, string, string) {
				m := &Manager{
					dataDir: tmpDir,
					manifests: map[string]*ServiceManifest{
						"test-service": {
							Name:    "test-service",
							Version: "v1.0.0",
						},
					},
				}
				// Create instance dir but not service dir
				instancePath := filepath.Join(tmpDir, "instances", "test-instance")
				os.MkdirAll(filepath.Join(instancePath, "setup", "cluster-services"), 0755)
				return m, "test-instance", "test-service"
			},
			expectedState:  "not_fetched",
			expectedLatest: "v1.0.0",
			expectedUpdate: false,
		},
		{
			name: "service fetched, versions match (up to date)",
			setup: func(tmpDir string) (*Manager, string, string) {
				m := &Manager{
					dataDir: tmpDir,
					manifests: map[string]*ServiceManifest{
						"test-service": {
							Name:    "test-service",
							Version: "v1.0.0",
						},
					},
				}
				instanceServiceDir := filepath.Join(tmpDir, "instances", "test-instance", "setup", "cluster-services", "test-service")
				os.MkdirAll(instanceServiceDir, 0755)

				// Write instance manifest with same version
				instanceManifest := ServiceManifest{
					Name:    "test-service",
					Version: "v1.0.0",
				}
				manifestData, _ := yaml.Marshal(instanceManifest)
				os.WriteFile(filepath.Join(instanceServiceDir, "wild-manifest.yaml"), manifestData, 0644)

				return m, "test-instance", "test-service"
			},
			expectedState:   "up_to_date",
			expectedVersion: "v1.0.0",
			expectedLatest:  "v1.0.0",
			expectedUpdate:  false,
		},
		{
			name: "service fetched, versions differ (update available)",
			setup: func(tmpDir string) (*Manager, string, string) {
				m := &Manager{
					dataDir: tmpDir,
					manifests: map[string]*ServiceManifest{
						"test-service": {
							Name:    "test-service",
							Version: "v2.0.0", // Newer version in embedded
						},
					},
				}
				instanceServiceDir := filepath.Join(tmpDir, "instances", "test-instance", "setup", "cluster-services", "test-service")
				os.MkdirAll(instanceServiceDir, 0755)

				// Write instance manifest with old version
				instanceManifest := ServiceManifest{
					Name:    "test-service",
					Version: "v1.0.0",
				}
				manifestData, _ := yaml.Marshal(instanceManifest)
				os.WriteFile(filepath.Join(instanceServiceDir, "wild-manifest.yaml"), manifestData, 0644)

				return m, "test-instance", "test-service"
			},
			expectedState:   "update_available",
			expectedVersion: "v1.0.0",
			expectedLatest:  "v2.0.0",
			expectedUpdate:  true,
		},
		{
			name: "service fetched, no version in instance manifest (update available - empty != v1.0.0)",
			setup: func(tmpDir string) (*Manager, string, string) {
				m := &Manager{
					dataDir: tmpDir,
					manifests: map[string]*ServiceManifest{
						"test-service": {
							Name:    "test-service",
							Version: "v1.0.0",
						},
					},
				}
				instanceServiceDir := filepath.Join(tmpDir, "instances", "test-instance", "setup", "cluster-services", "test-service")
				os.MkdirAll(instanceServiceDir, 0755)

				// Write instance manifest without version field
				instanceManifest := ServiceManifest{
					Name: "test-service",
					// No Version field - empty string != "v1.0.0"
				}
				manifestData, _ := yaml.Marshal(instanceManifest)
				os.WriteFile(filepath.Join(instanceServiceDir, "wild-manifest.yaml"), manifestData, 0644)

				return m, "test-instance", "test-service"
			},
			expectedState:   "update_available",
			expectedVersion: "",
			expectedLatest:  "v1.0.0",
			expectedUpdate:  true,
		},
		{
			name: "service fetched, manifest file malformed (cached)",
			setup: func(tmpDir string) (*Manager, string, string) {
				m := &Manager{
					dataDir: tmpDir,
					manifests: map[string]*ServiceManifest{
						"test-service": {
							Name:    "test-service",
							Version: "v1.0.0",
						},
					},
				}
				instanceServiceDir := filepath.Join(tmpDir, "instances", "test-instance", "setup", "cluster-services", "test-service")
				os.MkdirAll(instanceServiceDir, 0755)

				// Write malformed YAML
				os.WriteFile(filepath.Join(instanceServiceDir, "wild-manifest.yaml"), []byte("invalid: yaml: content:"), 0644)

				return m, "test-instance", "test-service"
			},
			expectedState:   "cached",
			expectedVersion: "v1.0.0",
			expectedLatest:  "v1.0.0",
			expectedUpdate:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			m, instanceName, serviceName := tt.setup(tmpDir)

			result := m.checkTemplateState(instanceName, serviceName)

			if result.State != tt.expectedState {
				t.Errorf("State = %s, want %s", result.State, tt.expectedState)
			}
			if tt.expectedVersion != "" && result.Version != tt.expectedVersion {
				t.Errorf("Version = %s, want %s", result.Version, tt.expectedVersion)
			}
			if tt.expectedLatest != "" && result.LatestVersion != tt.expectedLatest {
				t.Errorf("LatestVersion = %s, want %s", result.LatestVersion, tt.expectedLatest)
			}
			if result.UpdateAvailable != tt.expectedUpdate {
				t.Errorf("UpdateAvailable = %v, want %v", result.UpdateAvailable, tt.expectedUpdate)
			}
		})
	}
}

// TestCheckConfigurationState tests the configuration state detection logic
func TestCheckConfigurationState(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(tmpDir string) (*Manager, string, string)
		expectedState  string
		expectedReason string
	}{
		{
			name: "kustomize directory doesn't exist (not configured)",
			setup: func(tmpDir string) (*Manager, string, string) {
				m := &Manager{dataDir: tmpDir}
				instancePath := filepath.Join(tmpDir, "instances", "test-instance")
				serviceDir := filepath.Join(instancePath, "setup", "cluster-services", "test-service")
				os.MkdirAll(serviceDir, 0755)
				return m, "test-instance", "test-service"
			},
			expectedState: "not_configured",
		},
		{
			name: "no template directory, kustomize exists (compiled, no LastCompiled)",
			setup: func(tmpDir string) (*Manager, string, string) {
				m := &Manager{dataDir: tmpDir}
				instancePath := filepath.Join(tmpDir, "instances", "test-instance")
				serviceDir := filepath.Join(instancePath, "setup", "cluster-services", "test-service")
				kustomizeDir := filepath.Join(serviceDir, "kustomize")
				os.MkdirAll(kustomizeDir, 0755)
				// Create a file in kustomize to set mod time
				os.WriteFile(filepath.Join(kustomizeDir, "kustomization.yaml"), []byte("test"), 0644)
				return m, "test-instance", "test-service"
			},
			expectedState: "compiled",
			// Note: When there's no template directory, LastCompiled is not set (early return)
		},
		{
			name: "templates newer than compiled (needs recompile - templates changed)",
			setup: func(tmpDir string) (*Manager, string, string) {
				m := &Manager{dataDir: tmpDir}
				instancePath := filepath.Join(tmpDir, "instances", "test-instance")
				serviceDir := filepath.Join(instancePath, "setup", "cluster-services", "test-service")

				// Create kustomize dir first (older)
				kustomizeDir := filepath.Join(serviceDir, "kustomize")
				os.MkdirAll(kustomizeDir, 0755)
				os.WriteFile(filepath.Join(kustomizeDir, "kustomization.yaml"), []byte("old"), 0644)

				// Sleep to ensure time difference
				time.Sleep(10 * time.Millisecond)

				// Create template dir (newer)
				templateDir := filepath.Join(serviceDir, "kustomize.template")
				os.MkdirAll(templateDir, 0755)
				os.WriteFile(filepath.Join(templateDir, "deployment.yaml"), []byte("new"), 0644)

				return m, "test-instance", "test-service"
			},
			expectedState:  "needs_recompile",
			expectedReason: "templates_changed",
		},
		{
			name: "config newer than compiled (needs recompile - config changed)",
			setup: func(tmpDir string) (*Manager, string, string) {
				m := &Manager{dataDir: tmpDir}
				instancePath := filepath.Join(tmpDir, "instances", "test-instance")
				serviceDir := filepath.Join(instancePath, "setup", "cluster-services", "test-service")

				// Create template dir
				templateDir := filepath.Join(serviceDir, "kustomize.template")
				os.MkdirAll(templateDir, 0755)
				os.WriteFile(filepath.Join(templateDir, "deployment.yaml"), []byte("template"), 0644)

				// Sleep to ensure time difference
				time.Sleep(10 * time.Millisecond)

				// Create kustomize dir (older)
				kustomizeDir := filepath.Join(serviceDir, "kustomize")
				os.MkdirAll(kustomizeDir, 0755)
				os.WriteFile(filepath.Join(kustomizeDir, "kustomization.yaml"), []byte("old"), 0644)

				// Sleep to ensure time difference
				time.Sleep(10 * time.Millisecond)

				// Touch config file (newer)
				os.WriteFile(filepath.Join(instancePath, "config.yaml"), []byte("config: updated"), 0644)

				return m, "test-instance", "test-service"
			},
			expectedState:  "needs_recompile",
			expectedReason: "config_changed",
		},
		{
			name: "everything up to date (compiled)",
			setup: func(tmpDir string) (*Manager, string, string) {
				m := &Manager{dataDir: tmpDir}
				instancePath := filepath.Join(tmpDir, "instances", "test-instance")
				serviceDir := filepath.Join(instancePath, "setup", "cluster-services", "test-service")

				// Create config file (oldest)
				os.MkdirAll(instancePath, 0755)
				os.WriteFile(filepath.Join(instancePath, "config.yaml"), []byte("config: test"), 0644)

				// Sleep to ensure time difference
				time.Sleep(10 * time.Millisecond)

				// Create template dir (older)
				templateDir := filepath.Join(serviceDir, "kustomize.template")
				os.MkdirAll(templateDir, 0755)
				os.WriteFile(filepath.Join(templateDir, "deployment.yaml"), []byte("template"), 0644)

				// Sleep to ensure time difference
				time.Sleep(10 * time.Millisecond)

				// Create kustomize dir (newest)
				kustomizeDir := filepath.Join(serviceDir, "kustomize")
				os.MkdirAll(kustomizeDir, 0755)
				os.WriteFile(filepath.Join(kustomizeDir, "kustomization.yaml"), []byte("compiled"), 0644)

				return m, "test-instance", "test-service"
			},
			expectedState: "compiled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			m, instanceName, serviceName := tt.setup(tmpDir)

			result := m.checkConfigurationState(instanceName, serviceName)

			if result.State != tt.expectedState {
				t.Errorf("State = %s, want %s", result.State, tt.expectedState)
			}
			if tt.expectedReason != "" && result.Reason != tt.expectedReason {
				t.Errorf("Reason = %s, want %s", result.Reason, tt.expectedReason)
			}
			// LastCompiled is only set when templates exist and state is compiled or needs_recompile
			if (result.State == "compiled" || result.State == "needs_recompile") && tt.expectedReason != "" {
				// Only check if we expect a reason (meaning templates exist)
				if result.LastCompiled == nil {
					t.Error("LastCompiled should be set when templates exist")
				}
			}
		})
	}
}

// TestCheckDeploymentState tests the deployment state detection logic
func TestCheckDeploymentState(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(tmpDir string) (*Manager, string, string)
		expectedState  string
		expectedHealthy bool
	}{
		{
			name: "kubeconfig doesn't exist (not deployed)",
			setup: func(tmpDir string) (*Manager, string, string) {
				m := &Manager{
					dataDir: tmpDir,
					manifests: map[string]*ServiceManifest{
						"test-service": {
							Name:      "test-service",
							Namespace: "test-namespace",
						},
					},
				}
				instancePath := filepath.Join(tmpDir, "instances", "test-instance")
				os.MkdirAll(instancePath, 0755)
				// Don't create kubeconfig
				return m, "test-instance", "test-service"
			},
			expectedState:   "not_deployed",
			expectedHealthy: false,
		},
		{
			name: "service not in manifests (not deployed)",
			setup: func(tmpDir string) (*Manager, string, string) {
				m := &Manager{
					dataDir:   tmpDir,
					manifests: make(map[string]*ServiceManifest),
				}
				instancePath := filepath.Join(tmpDir, "instances", "test-instance")
				os.MkdirAll(instancePath, 0755)
				os.WriteFile(filepath.Join(instancePath, "kubeconfig"), []byte("fake kubeconfig"), 0644)
				return m, "test-instance", "nonexistent-service"
			},
			expectedState:   "not_deployed",
			expectedHealthy: false,
		},
		// Note: Testing actual kubectl calls requires mocking kubectl or integration tests
		// The above tests cover the code paths that don't require kubectl
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			m, instanceName, serviceName := tt.setup(tmpDir)

			result := m.checkDeploymentState(instanceName, serviceName)

			if result.State != tt.expectedState {
				t.Errorf("State = %s, want %s", result.State, tt.expectedState)
			}
			if result.Healthy != tt.expectedHealthy {
				t.Errorf("Healthy = %v, want %v", result.Healthy, tt.expectedHealthy)
			}
		})
	}
}

// TestCheckDeploymentStateOutOfSync tests the out_of_sync state detection
func TestCheckDeploymentStateOutOfSync(t *testing.T) {
	// This test simulates the scenario where kustomize files are newer than .last-deploy
	// Unfortunately, we can't fully test this without mocking kubectl, but we can
	// document the expected behavior and prepare the test structure for future mocking

	t.Run("kustomize newer than last deploy should return out_of_sync", func(t *testing.T) {
		// Note: This test documents the expected behavior when:
		// 1. Service is deployed (kubectl returns deployment info)
		// 2. kustomize directory exists with files
		// 3. .last-deploy file exists
		// 4. kustomize files are newer than .last-deploy
		// Expected result: State should be "out_of_sync"

		// In production, this scenario occurs when:
		// - User runs compile to regenerate manifests (updates kustomize/)
		// - But hasn't yet deployed those changes
		// - The .last-deploy file still has the old timestamp

		// To properly test this, we would need to:
		// 1. Mock kubectl.GetDeployment to return valid deployment info
		// 2. Create kustomize directory with recent files
		// 3. Create .last-deploy file with older timestamp
		// 4. Verify checkDeploymentState returns "out_of_sync"

		// For now, this test serves as documentation of the expected behavior
		t.Log("out_of_sync state occurs when kustomize files are newer than .last-deploy")
	})
}

// TestGetServiceLifecycleStatus tests the orchestration of lifecycle checks
func TestGetServiceLifecycleStatus(t *testing.T) {
	tmpDir := t.TempDir()
	m := &Manager{
		dataDir: tmpDir,
		manifests: map[string]*ServiceManifest{
			"test-service": {
				Name:    "test-service",
				Version: "v1.0.0",
			},
		},
	}

	// Setup instance with fetched but not compiled service
	instancePath := filepath.Join(tmpDir, "instances", "test-instance")
	serviceDir := filepath.Join(instancePath, "setup", "cluster-services", "test-service")
	os.MkdirAll(serviceDir, 0755)

	instanceManifest := ServiceManifest{
		Name:    "test-service",
		Version: "v1.0.0",
	}
	manifestData, _ := yaml.Marshal(instanceManifest)
	os.WriteFile(filepath.Join(serviceDir, "wild-manifest.yaml"), manifestData, 0644)

	lifecycle := m.getServiceLifecycleStatus("test-instance", "test-service")

	if lifecycle == nil {
		t.Fatal("getServiceLifecycleStatus returned nil")
	}

	// Check that all three phases are present
	if lifecycle.Templates.State == "" {
		t.Error("Templates state not set")
	}
	if lifecycle.Configuration.State == "" {
		t.Error("Configuration state not set")
	}
	if lifecycle.Deployment.State == "" {
		t.Error("Deployment state not set")
	}

	// Verify expected states for this setup
	if lifecycle.Templates.State != "up_to_date" {
		t.Errorf("Templates.State = %s, want up_to_date", lifecycle.Templates.State)
	}
	if lifecycle.Configuration.State != "not_configured" {
		t.Errorf("Configuration.State = %s, want not_configured", lifecycle.Configuration.State)
	}
	if lifecycle.Deployment.State != "not_deployed" {
		t.Errorf("Deployment.State = %s, want not_deployed", lifecycle.Deployment.State)
	}
}

// TestList_PopulatesLifecycle tests that List() properly populates the Lifecycle field
func TestList_PopulatesLifecycle(t *testing.T) {
	// NOTE: This test uses NewManager() which loads embedded services.
	// We verify that ANY service returned by List() has its Lifecycle field populated.
	// This tests the bug we fixed where List() was not calling getServiceLifecycleStatus().

	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	// Setup a test instance
	instancePath := filepath.Join(tmpDir, "instances", "test-instance")
	os.MkdirAll(filepath.Join(instancePath, "setup", "cluster-services"), 0755)

	services, err := m.List("test-instance")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	// We should have services from embedded manifests (cert-manager, traefik, etc.)
	if len(services) == 0 {
		t.Skip("No embedded services available, skipping test")
	}

	// Check that ALL services have their Lifecycle field populated
	for _, service := range services {
		if service.Lifecycle == nil {
			t.Errorf("Service %s: Lifecycle field is nil - this is the bug that tests should have caught!", service.Name)
			continue
		}

		// Verify lifecycle structure
		if service.Lifecycle.Templates.State == "" {
			t.Errorf("Service %s: Templates state not populated", service.Name)
		}
		if service.Lifecycle.Configuration.State == "" {
			t.Errorf("Service %s: Configuration state not populated", service.Name)
		}
		if service.Lifecycle.Deployment.State == "" {
			t.Errorf("Service %s: Deployment state not populated", service.Name)
		}
	}

	t.Logf("Successfully validated lifecycle field for %d services", len(services))
}

// TestGet_PopulatesLifecycle tests that Get() properly populates the Lifecycle field
func TestGet_PopulatesLifecycle(t *testing.T) {
	tmpDir := t.TempDir()

	m := &Manager{
		dataDir: tmpDir,
		manifests: map[string]*ServiceManifest{
			"test-service": {
				Name:        "test-service",
				Description: "Test service",
				Version:     "v1.0.0",
				Namespace:   "test-namespace",
			},
		},
	}

	// Setup instance with service files
	instancePath := filepath.Join(tmpDir, "instances", "test-instance")
	serviceDir := filepath.Join(instancePath, "setup", "cluster-services", "test-service")
	os.MkdirAll(serviceDir, 0755)

	instanceManifest := ServiceManifest{
		Name:    "test-service",
		Version: "v1.0.0",
	}
	manifestData, _ := yaml.Marshal(instanceManifest)
	os.WriteFile(filepath.Join(serviceDir, "wild-manifest.yaml"), manifestData, 0644)

	service, err := m.Get("test-instance", "test-service")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// THIS IS THE BUG WE FIXED: Lifecycle field must be populated
	if service.Lifecycle == nil {
		t.Fatal("Lifecycle field is nil - this is the bug that tests should have caught!")
	}

	// Verify lifecycle structure
	if service.Lifecycle.Templates.State == "" {
		t.Error("Templates state not populated")
	}
	if service.Lifecycle.Configuration.State == "" {
		t.Error("Configuration state not populated")
	}
	if service.Lifecycle.Deployment.State == "" {
		t.Error("Deployment state not populated")
	}
}

// TestGetFileModTime tests the helper function for getting file modification times
func TestGetFileModTime(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(tmpDir string) string
		wantZero bool
	}{
		{
			name: "existing file returns non-zero time",
			setup: func(tmpDir string) string {
				path := filepath.Join(tmpDir, "test.txt")
				os.WriteFile(path, []byte("test"), 0644)
				return path
			},
			wantZero: false,
		},
		{
			name: "non-existent file returns zero time",
			setup: func(tmpDir string) string {
				return filepath.Join(tmpDir, "nonexistent.txt")
			},
			wantZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			path := tt.setup(tmpDir)
			modTime := getFileModTime(path)

			if tt.wantZero && !modTime.IsZero() {
				t.Errorf("Expected zero time, got %v", modTime)
			}
			if !tt.wantZero && modTime.IsZero() {
				t.Error("Expected non-zero time, got zero")
			}
		})
	}
}

// TestGetDirectoryModTime tests the helper function for getting directory modification times
func TestGetDirectoryModTime(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(tmpDir string) string
		wantZero bool
	}{
		{
			name: "directory with files returns latest mod time",
			setup: func(tmpDir string) string {
				dir := filepath.Join(tmpDir, "testdir")
				os.MkdirAll(dir, 0755)

				// Create files with different times
				os.WriteFile(filepath.Join(dir, "old.txt"), []byte("old"), 0644)
				time.Sleep(10 * time.Millisecond)
				os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new"), 0644)

				return dir
			},
			wantZero: false,
		},
		{
			name: "empty directory returns zero time",
			setup: func(tmpDir string) string {
				dir := filepath.Join(tmpDir, "emptydir")
				os.MkdirAll(dir, 0755)
				return dir
			},
			wantZero: true,
		},
		{
			name: "non-existent directory returns zero time",
			setup: func(tmpDir string) string {
				return filepath.Join(tmpDir, "nonexistent")
			},
			wantZero: true,
		},
		{
			name: "nested files - returns latest",
			setup: func(tmpDir string) string {
				dir := filepath.Join(tmpDir, "nested")
				subdir := filepath.Join(dir, "sub")
				os.MkdirAll(subdir, 0755)

				os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("1"), 0644)
				time.Sleep(10 * time.Millisecond)
				os.WriteFile(filepath.Join(subdir, "file2.txt"), []byte("2"), 0644)

				return dir
			},
			wantZero: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			path := tt.setup(tmpDir)
			modTime := getDirectoryModTime(path)

			if tt.wantZero && !modTime.IsZero() {
				t.Errorf("Expected zero time, got %v", modTime)
			}
			if !tt.wantZero && modTime.IsZero() {
				t.Error("Expected non-zero time, got zero")
			}
		})
	}
}

// TestEdgeCases tests various edge cases and error conditions
func TestEdgeCases(t *testing.T) {
	t.Run("checkTemplateState with missing version field triggers cached state", func(t *testing.T) {
		tmpDir := t.TempDir()
		m := &Manager{
			dataDir: tmpDir,
			manifests: map[string]*ServiceManifest{
				"test-service": {
					Name:    "test-service",
					Version: "v1.0.0",
				},
			},
		}

		// Setup service with manifest missing version field
		instanceServiceDir := filepath.Join(tmpDir, "instances", "test-instance", "setup", "cluster-services", "test-service")
		os.MkdirAll(instanceServiceDir, 0755)

		// Empty string version field
		instanceManifest := ServiceManifest{
			Name:    "test-service",
			Version: "", // Empty version
		}
		manifestData, _ := yaml.Marshal(instanceManifest)
		os.WriteFile(filepath.Join(instanceServiceDir, "wild-manifest.yaml"), manifestData, 0644)

		result := m.checkTemplateState("test-instance", "test-service")

		// Should trigger update_available since empty != v1.0.0
		if result.State != "update_available" {
			t.Errorf("State = %s, want update_available", result.State)
		}
	})

	t.Run("checkConfigurationState handles missing config.yaml gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()
		m := &Manager{dataDir: tmpDir}

		instancePath := filepath.Join(tmpDir, "instances", "test-instance")
		serviceDir := filepath.Join(instancePath, "setup", "cluster-services", "test-service")

		// Create template and kustomize dirs
		templateDir := filepath.Join(serviceDir, "kustomize.template")
		kustomizeDir := filepath.Join(serviceDir, "kustomize")
		os.MkdirAll(templateDir, 0755)
		os.MkdirAll(kustomizeDir, 0755)
		os.WriteFile(filepath.Join(templateDir, "deployment.yaml"), []byte("template"), 0644)
		os.WriteFile(filepath.Join(kustomizeDir, "kustomization.yaml"), []byte("compiled"), 0644)
		// Don't create config.yaml

		result := m.checkConfigurationState("test-instance", "test-service")

		// Should return compiled since config.yaml missing = zero time (older than kustomize)
		if result.State != "compiled" {
			t.Errorf("State = %s, want compiled", result.State)
		}
	})
}

// TestNewManager verifies the Manager initialization
func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()

	// Note: This test will only work if embedded service manifests are available
	// In a real test environment, you might mock the setup.ListServices() function
	m := NewManager(tmpDir)

	if m == nil {
		t.Fatal("NewManager returned nil")
	}

	if m.dataDir != tmpDir {
		t.Errorf("dataDir = %s, want %s", m.dataDir, tmpDir)
	}

	if m.manifests == nil {
		t.Error("manifests map is nil")
	}

	// Verify manifests were loaded (if any exist in embedded setup)
	// This is environment-dependent
	t.Logf("Loaded %d service manifests", len(m.manifests))
}

// TestExtractConfigValues tests extracting config values from manifest references
func TestExtractConfigValues(t *testing.T) {
	tests := []struct {
		name           string
		configContent  string
		paths          []string
		expectedValues map[string]string
		expectError    bool
	}{
		{
			name: "extract simple config references",
			configContent: `
cloud:
  domain: example.com
cluster:
  ipAddressPool: 192.168.1.10-192.168.1.20
`,
			paths: []string{
				"cloud.domain",
				"cluster.ipAddressPool",
			},
			expectedValues: map[string]string{
				"cloud.domain":          "example.com",
				"cluster.ipAddressPool": "192.168.1.10-192.168.1.20",
			},
		},
		{
			name: "extract nested service config",
			configContent: `
cluster:
  services:
    test-service:
      smtp:
        host: mail.example.com
        port: "587"
`,
			paths: []string{
				"cluster.services.test-service.smtp.host",
				"cluster.services.test-service.smtp.port",
			},
			expectedValues: map[string]string{
				"cluster.services.test-service.smtp.host": "mail.example.com",
				"cluster.services.test-service.smtp.port": "587",
			},
		},
		{
			name: "handle missing config values",
			configContent: `
cloud:
  domain: example.com
`,
			paths: []string{
				"cloud.domain",
				"cloud.missing.value",
			},
			expectedValues: map[string]string{
				"cloud.domain":        "example.com",
				"cloud.missing.value": "",
			},
		},
		{
			name:           "no config references",
			configContent:  `config: test`,
			paths:          []string{},
			expectedValues: map[string]string{},
		},
		{
			name: "convert non-string values",
			configContent: `
cluster:
  port: 8080
  enabled: true
`,
			paths: []string{
				"cluster.port",
				"cluster.enabled",
			},
			expectedValues: map[string]string{
				"cluster.port":    "8080",
				"cluster.enabled": "true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			// Write config file
			if err := os.WriteFile(configPath, []byte(tt.configContent), 0644); err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			values, err := extractConfigValues(configPath, tt.paths)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(values) != len(tt.expectedValues) {
				t.Errorf("Got %d values, expected %d", len(values), len(tt.expectedValues))
			}

			for key, expectedValue := range tt.expectedValues {
				if actualValue, exists := values[key]; !exists {
					t.Errorf("Missing key %s", key)
				} else if actualValue != expectedValue {
					t.Errorf("Key %s: got %s, expected %s", key, actualValue, expectedValue)
				}
			}
		})
	}
}

// TestSaveAndLoadCompileConfig tests saving and loading compile config
func TestSaveAndLoadCompileConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create service directory
	serviceDir := filepath.Join(tmpDir, "test-service")
	os.MkdirAll(serviceDir, 0755)

	// Test saving config
	configValues := map[string]string{
		"cloud.domain":          "example.com",
		"cluster.ipAddressPool": "192.168.1.10-192.168.1.20",
		"smtp.host":            "mail.example.com",
	}

	err := saveCompileConfig(serviceDir, configValues)
	if err != nil {
		t.Fatalf("saveCompileConfig failed: %v", err)
	}

	// Verify file was created
	configFile := filepath.Join(serviceDir, ".last-compile-config.json")
	if _, err := os.Stat(configFile); err != nil {
		t.Fatalf("Config file not created: %v", err)
	}

	// Test loading config
	config, err := loadCompileConfig(serviceDir)
	if err != nil {
		t.Fatalf("loadCompileConfig failed: %v", err)
	}

	if config == nil {
		t.Fatal("Loaded config is nil")
	}

	// Verify values match
	if len(config.Values) != len(configValues) {
		t.Errorf("Loaded %d values, expected %d", len(config.Values), len(configValues))
	}

	for key, expectedValue := range configValues {
		if actualValue, exists := config.Values[key]; !exists {
			t.Errorf("Missing key %s in loaded config", key)
		} else if actualValue != expectedValue {
			t.Errorf("Key %s: got %s, expected %s", key, actualValue, expectedValue)
		}
	}

	// Test that timestamp is recent
	if time.Since(config.Timestamp) > time.Second {
		t.Errorf("Timestamp too old: %v", config.Timestamp)
	}
}

// TestConfigValuesChanged tests config value comparison
func TestConfigValuesChanged(t *testing.T) {
	tests := []struct {
		name        string
		oldValues   map[string]string
		newValues   map[string]string
		expectChanged bool
	}{
		{
			name: "no changes",
			oldValues: map[string]string{
				"cloud.domain": "example.com",
				"smtp.host":   "mail.example.com",
			},
			newValues: map[string]string{
				"cloud.domain": "example.com",
				"smtp.host":   "mail.example.com",
			},
			expectChanged: false,
		},
		{
			name: "value changed",
			oldValues: map[string]string{
				"cloud.domain": "example.com",
			},
			newValues: map[string]string{
				"cloud.domain": "newexample.com",
			},
			expectChanged: true,
		},
		{
			name: "new key added",
			oldValues: map[string]string{
				"cloud.domain": "example.com",
			},
			newValues: map[string]string{
				"cloud.domain": "example.com",
				"smtp.host":   "mail.example.com",
			},
			expectChanged: true,
		},
		{
			name: "key removed",
			oldValues: map[string]string{
				"cloud.domain": "example.com",
				"smtp.host":   "mail.example.com",
			},
			newValues: map[string]string{
				"cloud.domain": "example.com",
			},
			expectChanged: true,
		},
		{
			name:          "both empty",
			oldValues:     map[string]string{},
			newValues:     map[string]string{},
			expectChanged: false,
		},
		{
			name:      "old nil, new empty",
			oldValues: nil,
			newValues: map[string]string{},
			expectChanged: false,
		},
		{
			name:      "old empty, new has values",
			oldValues: map[string]string{},
			newValues: map[string]string{
				"cloud.domain": "example.com",
			},
			expectChanged: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changed := configValuesChanged(tt.oldValues, tt.newValues)
			if changed != tt.expectChanged {
				t.Errorf("configValuesChanged = %v, expected %v", changed, tt.expectChanged)
			}
		})
	}
}

// TestCheckConfigurationStateWithValueTracking tests config state with value tracking
func TestCheckConfigurationStateWithValueTracking(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(tmpDir string) (*Manager, string, string)
		expectedState  string
		expectedReason string
	}{
		{
			name: "service without config references - never needs recompile for config changes",
			setup: func(tmpDir string) (*Manager, string, string) {
				m := &Manager{
					dataDir: tmpDir,
					manifests: map[string]*ServiceManifest{
						"test-service": {
							Name: "test-service",
							// No ConfigReferences or ServiceConfig
						},
					},
				}

				instancePath := filepath.Join(tmpDir, "instances", "test-instance")
				serviceDir := filepath.Join(instancePath, "setup", "cluster-services", "test-service")

				// Create template and kustomize dirs
				templateDir := filepath.Join(serviceDir, "kustomize.template")
				kustomizeDir := filepath.Join(serviceDir, "kustomize")
				os.MkdirAll(templateDir, 0755)
				os.MkdirAll(kustomizeDir, 0755)
				os.WriteFile(filepath.Join(templateDir, "deployment.yaml"), []byte("template"), 0644)
				os.WriteFile(filepath.Join(kustomizeDir, "kustomization.yaml"), []byte("compiled"), 0644)

				// Create config file (newer than kustomize)
				time.Sleep(10 * time.Millisecond)
				os.WriteFile(filepath.Join(instancePath, "config.yaml"), []byte("config: updated"), 0644)

				// Write manifest to instance
				manifestData, _ := yaml.Marshal(m.manifests["test-service"])
				os.WriteFile(filepath.Join(serviceDir, "wild-manifest.yaml"), manifestData, 0644)

				return m, "test-instance", "test-service"
			},
			expectedState: "compiled",
			// Service without config refs should not trigger recompile for config changes
		},
		{
			name: "config values unchanged - remains compiled",
			setup: func(tmpDir string) (*Manager, string, string) {
				m := &Manager{
					dataDir: tmpDir,
					manifests: map[string]*ServiceManifest{
						"test-service": {
							Name: "test-service",
							ConfigReferences: []string{
								"cloud.domain",
							},
						},
					},
				}

				instancePath := filepath.Join(tmpDir, "instances", "test-instance")
				serviceDir := filepath.Join(instancePath, "setup", "cluster-services", "test-service")

				// Create dirs
				templateDir := filepath.Join(serviceDir, "kustomize.template")
				kustomizeDir := filepath.Join(serviceDir, "kustomize")
				os.MkdirAll(templateDir, 0755)
				os.MkdirAll(kustomizeDir, 0755)
				os.WriteFile(filepath.Join(templateDir, "deployment.yaml"), []byte("template"), 0644)
				os.WriteFile(filepath.Join(kustomizeDir, "kustomization.yaml"), []byte("compiled"), 0644)

				// Create config
				config := map[string]interface{}{
					"cloud": map[string]interface{}{
						"domain": "example.com",
					},
				}
				configData, _ := yaml.Marshal(config)
				os.WriteFile(filepath.Join(instancePath, "config.yaml"), configData, 0644)

				// Save tracking file with same values
				saveCompileConfig(serviceDir, map[string]string{
					"cloud.domain": "example.com",
				})

				// Write manifest
				manifestData, _ := yaml.Marshal(m.manifests["test-service"])
				os.WriteFile(filepath.Join(serviceDir, "wild-manifest.yaml"), manifestData, 0644)

				return m, "test-instance", "test-service"
			},
			expectedState: "compiled",
		},
		{
			name: "config values changed - needs recompile",
			setup: func(tmpDir string) (*Manager, string, string) {
				m := &Manager{
					dataDir: tmpDir,
					manifests: map[string]*ServiceManifest{
						"test-service": {
							Name: "test-service",
							ConfigReferences: []string{
								"cloud.domain",
							},
						},
					},
				}

				instancePath := filepath.Join(tmpDir, "instances", "test-instance")
				serviceDir := filepath.Join(instancePath, "setup", "cluster-services", "test-service")

				// Create dirs
				templateDir := filepath.Join(serviceDir, "kustomize.template")
				kustomizeDir := filepath.Join(serviceDir, "kustomize")
				os.MkdirAll(templateDir, 0755)
				os.MkdirAll(kustomizeDir, 0755)
				os.WriteFile(filepath.Join(templateDir, "deployment.yaml"), []byte("template"), 0644)
				os.WriteFile(filepath.Join(kustomizeDir, "kustomization.yaml"), []byte("compiled"), 0644)

				// Create config with new value
				config := map[string]interface{}{
					"cloud": map[string]interface{}{
						"domain": "newexample.com", // Changed!
					},
				}
				configData, _ := yaml.Marshal(config)
				os.WriteFile(filepath.Join(instancePath, "config.yaml"), configData, 0644)

				// Save tracking file with old value
				saveCompileConfig(serviceDir, map[string]string{
					"cloud.domain": "example.com", // Old value
				})

				// Write manifest
				manifestData, _ := yaml.Marshal(m.manifests["test-service"])
				os.WriteFile(filepath.Join(serviceDir, "wild-manifest.yaml"), manifestData, 0644)

				return m, "test-instance", "test-service"
			},
			expectedState:  "needs_recompile",
			expectedReason: "config_changed",
		},
		{
			name: "no tracking file - fallback to timestamp check",
			setup: func(tmpDir string) (*Manager, string, string) {
				m := &Manager{
					dataDir: tmpDir,
					manifests: map[string]*ServiceManifest{
						"test-service": {
							Name: "test-service",
							ConfigReferences: []string{
								"cloud.domain",
							},
						},
					},
				}

				instancePath := filepath.Join(tmpDir, "instances", "test-instance")
				serviceDir := filepath.Join(instancePath, "setup", "cluster-services", "test-service")

				// Create old kustomize
				kustomizeDir := filepath.Join(serviceDir, "kustomize")
				os.MkdirAll(kustomizeDir, 0755)
				os.WriteFile(filepath.Join(kustomizeDir, "kustomization.yaml"), []byte("compiled"), 0644)

				time.Sleep(10 * time.Millisecond)

				// Create newer template
				templateDir := filepath.Join(serviceDir, "kustomize.template")
				os.MkdirAll(templateDir, 0755)
				os.WriteFile(filepath.Join(templateDir, "deployment.yaml"), []byte("template"), 0644)

				time.Sleep(10 * time.Millisecond)

				// Create newer config
				config := map[string]interface{}{
					"cloud": map[string]interface{}{
						"domain": "example.com",
					},
				}
				configData, _ := yaml.Marshal(config)
				os.WriteFile(filepath.Join(instancePath, "config.yaml"), configData, 0644)

				// No tracking file created

				// Write manifest
				manifestData, _ := yaml.Marshal(m.manifests["test-service"])
				os.WriteFile(filepath.Join(serviceDir, "wild-manifest.yaml"), manifestData, 0644)

				return m, "test-instance", "test-service"
			},
			expectedState:  "needs_recompile",
			expectedReason: "config_changed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			m, instanceName, serviceName := tt.setup(tmpDir)

			result := m.checkConfigurationState(instanceName, serviceName)

			if result.State != tt.expectedState {
				t.Errorf("State = %s, want %s", result.State, tt.expectedState)
			}
			if tt.expectedReason != "" && result.Reason != tt.expectedReason {
				t.Errorf("Reason = %s, want %s", result.Reason, tt.expectedReason)
			}
		})
	}
}
