package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDir(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test", "nested", "dir")

	err := EnsureDir(testDir, 0755)
	if err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	// Verify directory exists
	info, err := os.Stat(testDir)
	if err != nil {
		t.Fatalf("Directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("Path is not a directory")
	}

	// Calling again should be idempotent
	err = EnsureDir(testDir, 0755)
	if err != nil {
		t.Fatalf("EnsureDir not idempotent: %v", err)
	}
}

func TestWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testData := []byte("test content")

	// Write file
	err := WriteFile(testFile, testData, 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Read file back
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if string(data) != string(testData) {
		t.Fatalf("Data mismatch: got %q, want %q", string(data), string(testData))
	}
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// File should not exist initially
	if FileExists(testFile) {
		t.Fatalf("File should not exist")
	}

	// Create file
	err := WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// File should exist now
	if !FileExists(testFile) {
		t.Fatalf("File should exist")
	}
}

func TestWithLock(t *testing.T) {
	tmpDir := t.TempDir()
	lockFile := filepath.Join(tmpDir, "test.lock")
	counter := 0

	// Execute with lock
	err := WithLock(lockFile, func() error {
		counter++
		return nil
	})
	if err != nil {
		t.Fatalf("WithLock failed: %v", err)
	}

	if counter != 1 {
		t.Fatalf("Function not executed: counter=%d", counter)
	}

	// Should be idempotent - can acquire lock multiple times sequentially
	err = WithLock(lockFile, func() error {
		counter++
		return nil
	})
	if err != nil {
		t.Fatalf("WithLock failed on second call: %v", err)
	}

	if counter != 2 {
		t.Fatalf("Function not executed on second call: counter=%d", counter)
	}
}
