package context

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManager_GetSetCurrentContext(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	// Create test instances
	instancesDir := filepath.Join(tmpDir, "instances")
	instances := []string{"cloud1", "cloud2"}
	for _, name := range instances {
		instancePath := filepath.Join(instancesDir, name)
		err := os.MkdirAll(instancePath, 0755)
		if err != nil {
			t.Fatalf("Failed to create instance dir: %v", err)
		}
	}

	// Initially should have no context
	_, err := m.GetCurrentContext()
	if err == nil {
		t.Fatalf("Should have no context initially")
	}

	// Set context
	err = m.SetCurrentContext("cloud1")
	if err != nil {
		t.Fatalf("SetCurrentContext failed: %v", err)
	}

	// Get context
	ctx, err := m.GetCurrentContext()
	if err != nil {
		t.Fatalf("GetCurrentContext failed: %v", err)
	}
	if ctx != "cloud1" {
		t.Errorf("Wrong context: got %q, want %q", ctx, "cloud1")
	}

	// Change context
	err = m.SetCurrentContext("cloud2")
	if err != nil {
		t.Fatalf("SetCurrentContext failed: %v", err)
	}

	ctx, err = m.GetCurrentContext()
	if err != nil {
		t.Fatalf("GetCurrentContext failed: %v", err)
	}
	if ctx != "cloud2" {
		t.Errorf("Wrong context: got %q, want %q", ctx, "cloud2")
	}
}

func TestManager_SetCurrentContext_ValidationError(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	// Trying to set context to non-existent instance should fail
	err := m.SetCurrentContext("non-existent")
	if err == nil {
		t.Fatalf("SetCurrentContext should fail for non-existent instance")
	}
}

func TestManager_ClearCurrentContext(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	// Create test instance
	instancesDir := filepath.Join(tmpDir, "instances")
	instancePath := filepath.Join(instancesDir, "test-cloud")
	err := os.MkdirAll(instancePath, 0755)
	if err != nil {
		t.Fatalf("Failed to create instance dir: %v", err)
	}

	// Set context
	err = m.SetCurrentContext("test-cloud")
	if err != nil {
		t.Fatalf("SetCurrentContext failed: %v", err)
	}

	// Clear context
	err = m.ClearCurrentContext()
	if err != nil {
		t.Fatalf("ClearCurrentContext failed: %v", err)
	}

	// Context should be gone
	_, err = m.GetCurrentContext()
	if err == nil {
		t.Fatalf("Context should be cleared")
	}
}
