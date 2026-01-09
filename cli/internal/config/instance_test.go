package config

import (
	"os"
	"path/filepath"
	"testing"
)

// mockInstanceLister is a mock implementation of InstanceLister for testing
type mockInstanceLister struct {
	instances []string
	err       error
}

func (m *mockInstanceLister) ListInstances() ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.instances, nil
}

func TestGetCurrentInstance(t *testing.T) {
	// Save and restore env var
	oldWildCLIData := os.Getenv("WILD_CLI_DATA")
	defer os.Setenv("WILD_CLI_DATA", oldWildCLIData)

	// Create temp directory for testing
	tmpDir := t.TempDir()
	os.Setenv("WILD_CLI_DATA", tmpDir)

	tests := []struct {
		name         string
		flagInstance string
		fileInstance string
		apiInstances []string
		wantInstance string
		wantSource   string
		wantErr      bool
	}{
		{
			name:         "flag takes priority",
			flagInstance: "flag-instance",
			fileInstance: "file-instance",
			apiInstances: []string{"api-instance"},
			wantInstance: "flag-instance",
			wantSource:   "flag",
			wantErr:      false,
		},
		{
			name:         "file takes priority over api",
			flagInstance: "",
			fileInstance: "file-instance",
			apiInstances: []string{"api-instance"},
			wantInstance: "file-instance",
			wantSource:   "file",
			wantErr:      false,
		},
		{
			name:         "auto-select first from api",
			flagInstance: "",
			fileInstance: "",
			apiInstances: []string{"first-instance", "second-instance"},
			wantInstance: "first-instance",
			wantSource:   "auto",
			wantErr:      false,
		},
		{
			name:         "no instance available",
			flagInstance: "",
			fileInstance: "",
			apiInstances: []string{},
			wantInstance: "",
			wantSource:   "",
			wantErr:      true,
		},
		{
			name:         "no api client and no config",
			flagInstance: "",
			fileInstance: "",
			apiInstances: nil,
			wantInstance: "",
			wantSource:   "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up file
			currentFile := filepath.Join(tmpDir, "current_instance")
			if tt.fileInstance != "" {
				if err := os.WriteFile(currentFile, []byte(tt.fileInstance), 0644); err != nil {
					t.Fatalf("Failed to write test file: %v", err)
				}
			} else {
				os.Remove(currentFile)
			}

			// Set up API client mock
			var lister InstanceLister
			if tt.apiInstances != nil {
				lister = &mockInstanceLister{instances: tt.apiInstances}
			}

			// Test
			gotInstance, gotSource, err := GetCurrentInstance(tt.flagInstance, lister)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetCurrentInstance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if gotInstance != tt.wantInstance {
				t.Errorf("GetCurrentInstance() instance = %v, want %v", gotInstance, tt.wantInstance)
			}

			if gotSource != tt.wantSource {
				t.Errorf("GetCurrentInstance() source = %v, want %v", gotSource, tt.wantSource)
			}
		})
	}
}

func TestSetCurrentInstance(t *testing.T) {
	// Save and restore env var
	oldWildCLIData := os.Getenv("WILD_CLI_DATA")
	defer os.Setenv("WILD_CLI_DATA", oldWildCLIData)

	// Create temp directory for testing
	tmpDir := t.TempDir()
	os.Setenv("WILD_CLI_DATA", tmpDir)

	// Test setting instance
	testInstance := "test-instance"
	err := SetCurrentInstance(testInstance)
	if err != nil {
		t.Fatalf("SetCurrentInstance() error = %v", err)
	}

	// Verify file was written
	currentFile := filepath.Join(tmpDir, "current_instance")
	data, err := os.ReadFile(currentFile)
	if err != nil {
		t.Fatalf("Failed to read current_instance file: %v", err)
	}

	if string(data) != testInstance {
		t.Errorf("File content = %v, want %v", string(data), testInstance)
	}

	// Test updating instance
	newInstance := "new-instance"
	err = SetCurrentInstance(newInstance)
	if err != nil {
		t.Fatalf("SetCurrentInstance() error = %v", err)
	}

	data, err = os.ReadFile(currentFile)
	if err != nil {
		t.Fatalf("Failed to read current_instance file: %v", err)
	}

	if string(data) != newInstance {
		t.Errorf("File content = %v, want %v", string(data), newInstance)
	}
}

func TestGetWildCLIDataDir(t *testing.T) {
	// Save and restore env var
	oldWildCLIData := os.Getenv("WILD_CLI_DATA")
	defer os.Setenv("WILD_CLI_DATA", oldWildCLIData)

	tests := []struct {
		name    string
		envVar  string
		wantDir string
	}{
		{
			name:    "custom directory from env",
			envVar:  "/custom/path",
			wantDir: "/custom/path",
		},
		{
			name:   "default directory when env not set",
			envVar: "",
			// We can't predict the exact home directory, so we'll just check it's not empty
			wantDir: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVar != "" {
				os.Setenv("WILD_CLI_DATA", tt.envVar)
			} else {
				os.Unsetenv("WILD_CLI_DATA")
			}

			gotDir := GetWildCLIDataDir()

			if tt.wantDir != "" && gotDir != tt.wantDir {
				t.Errorf("GetWildCLIDataDir() = %v, want %v", gotDir, tt.wantDir)
			}

			if tt.wantDir == "" && gotDir == "" {
				t.Errorf("GetWildCLIDataDir() should not be empty when using default")
			}
		})
	}
}
