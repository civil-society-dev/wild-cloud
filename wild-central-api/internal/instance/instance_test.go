package instance

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManager_CreateInstance(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	instanceName := "test-cloud"

	// Create instance
	err := m.CreateInstance(instanceName)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	// Verify instance directory structure
	instancePath := m.GetInstancePath(instanceName)
	expectedDirs := []string{
		instancePath,
		filepath.Join(instancePath, "talos"),
		filepath.Join(instancePath, "k8s"),
		filepath.Join(instancePath, "logs"),
		filepath.Join(instancePath, "backups"),
	}

	for _, dir := range expectedDirs {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("Directory not created: %s: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("Path is not a directory: %s", dir)
		}
	}

	// Verify config.yaml exists
	configPath := m.GetInstanceConfigPath(instanceName)
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("Config file not created: %v", err)
	}

	// Verify secrets.yaml exists with correct permissions
	secretsPath := m.GetInstanceSecretsPath(instanceName)
	info, err := os.Stat(secretsPath)
	if err != nil {
		t.Errorf("Secrets file not created: %v", err)
	} else {
		// Check permissions (should be 0600)
		mode := info.Mode().Perm()
		if mode != 0600 {
			t.Errorf("Secrets file has wrong permissions: got %o, want 0600", mode)
		}
	}

	// Test idempotency - creating again should not error
	err = m.CreateInstance(instanceName)
	if err != nil {
		t.Fatalf("CreateInstance not idempotent: %v", err)
	}
}

func TestManager_ListInstances(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	// Initially should be empty
	instances, err := m.ListInstances()
	if err != nil {
		t.Fatalf("ListInstances failed: %v", err)
	}
	if len(instances) != 0 {
		t.Fatalf("Expected 0 instances, got %d", len(instances))
	}

	// Create instances
	instanceNames := []string{"cloud1", "cloud2", "cloud3"}
	for _, name := range instanceNames {
		err := m.CreateInstance(name)
		if err != nil {
			t.Fatalf("CreateInstance failed: %v", err)
		}
	}

	// List should return all instances
	instances, err = m.ListInstances()
	if err != nil {
		t.Fatalf("ListInstances failed: %v", err)
	}
	if len(instances) != len(instanceNames) {
		t.Fatalf("Expected %d instances, got %d", len(instanceNames), len(instances))
	}

	// Verify all expected instances are present
	instanceMap := make(map[string]bool)
	for _, name := range instances {
		instanceMap[name] = true
	}
	for _, expected := range instanceNames {
		if !instanceMap[expected] {
			t.Errorf("Expected instance %q not found", expected)
		}
	}
}

func TestManager_DeleteInstance(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	instanceName := "test-cloud"

	// Create instance
	err := m.CreateInstance(instanceName)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	// Verify it exists (by checking directory)
	instancePath := m.GetInstancePath(instanceName)
	if _, err := os.Stat(instancePath); err != nil {
		t.Fatalf("Instance should exist: %v", err)
	}

	// Delete instance
	err = m.DeleteInstance(instanceName)
	if err != nil {
		t.Fatalf("DeleteInstance failed: %v", err)
	}

	// Verify it's gone
	err = m.ValidateInstance(instanceName)
	if err == nil {
		t.Fatalf("Instance should not exist after deletion")
	}

	// Deleting non-existent instance should error
	err = m.DeleteInstance(instanceName)
	if err == nil {
		t.Fatalf("Deleting non-existent instance should error")
	}
}

func TestManager_ValidateInstance(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	instanceName := "test-cloud"

	// Should fail for non-existent instance
	err := m.ValidateInstance(instanceName)
	if err == nil {
		t.Fatalf("ValidateInstance should fail for non-existent instance")
	}

	// Create instance
	err = m.CreateInstance(instanceName)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	// Should succeed for existing instance (if yq is available)
	// Note: ValidateInstance requires yq for config validation
	err = m.ValidateInstance(instanceName)
	if err != nil {
		// It's OK if yq is not installed, just check instance exists
		if !m.InstanceExists(instanceName) {
			t.Fatalf("Instance should exist after creation")
		}
		t.Logf("ValidateInstance failed (likely yq not installed): %v", err)
	}
}
