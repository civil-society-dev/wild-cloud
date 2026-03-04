package destinations

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	btypes "github.com/wild-cloud/wild-central/daemon/internal/backup/types"
)

func TestLocalDestination_NewLocalDestination(t *testing.T) {
	tests := []struct {
		name        string
		config      *btypes.LocalConfig
		expectError bool
	}{
		{
			name: "successful creation",
			config: &btypes.LocalConfig{
				Path: t.TempDir(),
			},
			expectError: false,
		},
		{
			name: "creates missing directory",
			config: &btypes.LocalConfig{
				Path: filepath.Join(t.TempDir(), "new", "nested", "dir"),
			},
			expectError: false,
		},
		{
			name: "invalid path",
			config: &btypes.LocalConfig{
				Path: "/root/no-permission",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dest, err := NewLocalDestination(tt.config)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, dest)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, dest)
				// Verify directory exists
				_, statErr := os.Stat(tt.config.Path)
				assert.NoError(t, statErr)
			}
		})
	}
}

func TestLocalDestination_Put(t *testing.T) {
	tempDir := t.TempDir()
	dest, err := NewLocalDestination(&btypes.LocalConfig{Path: tempDir})
	require.NoError(t, err)

	tests := []struct {
		name        string
		key         string
		data        []byte
		expectError bool
	}{
		{
			name:        "simple file",
			key:         "backup.tar.gz",
			data:        []byte("test backup data"),
			expectError: false,
		},
		{
			name:        "nested path",
			key:         "instance/app/backup.tar.gz",
			data:        []byte("nested backup data"),
			expectError: false,
		},
		{
			name:        "empty file",
			key:         "empty.txt",
			data:        []byte{},
			expectError: false,
		},
		{
			name:        "large file",
			key:         "large.bin",
			data:        make([]byte, 1024*1024), // 1MB
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader(tt.data)
			size, err := dest.Put(tt.key, reader)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, int64(len(tt.data)), size)

				// Verify file was created with correct content
				fullPath := filepath.Join(tempDir, tt.key)
				content, readErr := os.ReadFile(fullPath)
				assert.NoError(t, readErr)
				assert.Equal(t, tt.data, content)
			}
		})
	}
}

func TestLocalDestination_Get(t *testing.T) {
	tempDir := t.TempDir()
	dest, err := NewLocalDestination(&btypes.LocalConfig{Path: tempDir})
	require.NoError(t, err)

	// Create test files
	testFiles := map[string][]byte{
		"file1.txt":           []byte("content1"),
		"nested/file2.txt":    []byte("content2"),
		"deep/nested/file.gz": []byte("compressed data"),
	}

	for key, content := range testFiles {
		fullPath := filepath.Join(tempDir, key)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
		require.NoError(t, os.WriteFile(fullPath, content, 0644))
	}

	tests := []struct {
		name        string
		key         string
		expectError bool
		expectData  []byte
	}{
		{
			name:        "existing file",
			key:         "file1.txt",
			expectError: false,
			expectData:  []byte("content1"),
		},
		{
			name:        "nested file",
			key:         "nested/file2.txt",
			expectError: false,
			expectData:  []byte("content2"),
		},
		{
			name:        "non-existent file",
			key:         "missing.txt",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, err := dest.Get(tt.key)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, reader)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, reader)
				defer reader.Close()

				content, readErr := io.ReadAll(reader)
				assert.NoError(t, readErr)
				assert.Equal(t, tt.expectData, content)
			}
		})
	}
}

func TestLocalDestination_Delete(t *testing.T) {
	tempDir := t.TempDir()
	dest, err := NewLocalDestination(&btypes.LocalConfig{Path: tempDir})
	require.NoError(t, err)

	// Create test files
	testFile := filepath.Join(tempDir, "test", "backup.tar.gz")
	require.NoError(t, os.MkdirAll(filepath.Dir(testFile), 0755))
	require.NoError(t, os.WriteFile(testFile, []byte("data"), 0644))

	// Create another file in same directory
	otherFile := filepath.Join(tempDir, "test", "other.txt")
	require.NoError(t, os.WriteFile(otherFile, []byte("other"), 0644))

	// Delete the first file
	err = dest.Delete("test/backup.tar.gz")
	assert.NoError(t, err)

	// Verify file is deleted
	_, statErr := os.Stat(testFile)
	assert.True(t, os.IsNotExist(statErr))

	// Verify other file still exists
	_, statErr = os.Stat(otherFile)
	assert.NoError(t, statErr)

	// Delete non-existent file should not error
	err = dest.Delete("non-existent.txt")
	assert.NoError(t, err)

	// Delete the other file
	err = dest.Delete("test/other.txt")
	assert.NoError(t, err)

	// The empty directory should be cleaned up
	_, statErr = os.Stat(filepath.Join(tempDir, "test"))
	assert.True(t, os.IsNotExist(statErr))
}

func TestLocalDestination_List(t *testing.T) {
	tempDir := t.TempDir()
	dest, err := NewLocalDestination(&btypes.LocalConfig{Path: tempDir})
	require.NoError(t, err)

	// Create test files
	testFiles := map[string][]byte{
		"app1/backup1.tar.gz": []byte("backup1"),
		"app1/backup2.tar.gz": []byte("backup22"),
		"app2/backup1.tar.gz": []byte("app2backup"),
		"other/file.txt":      []byte("other"),
	}

	for key, content := range testFiles {
		fullPath := filepath.Join(tempDir, key)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
		require.NoError(t, os.WriteFile(fullPath, content, 0644))
		// Set specific mod time for testing
		modTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		os.Chtimes(fullPath, modTime, modTime)
	}

	tests := []struct {
		name         string
		prefix       string
		expectCount  int
		expectKeys   []string
	}{
		{
			name:        "list all",
			prefix:      "",
			expectCount: 4,
			expectKeys:  []string{"app1/backup1.tar.gz", "app1/backup2.tar.gz", "app2/backup1.tar.gz", "other/file.txt"},
		},
		{
			name:        "list app1 only",
			prefix:      "app1",
			expectCount: 2,
			expectKeys:  []string{"app1/backup1.tar.gz", "app1/backup2.tar.gz"},
		},
		{
			name:        "list app2 only",
			prefix:      "app2",
			expectCount: 1,
			expectKeys:  []string{"app2/backup1.tar.gz"},
		},
		{
			name:        "non-existent prefix",
			prefix:      "missing",
			expectCount: 0,
			expectKeys:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects, err := dest.List(tt.prefix)
			assert.NoError(t, err)
			assert.Len(t, objects, tt.expectCount)

			// Check expected keys are present
			foundKeys := make(map[string]bool)
			for _, obj := range objects {
				foundKeys[obj.Key] = true
				// Verify size is correct
				if content, exists := testFiles[obj.Key]; exists {
					assert.Equal(t, int64(len(content)), obj.Size)
				}
			}

			for _, expectedKey := range tt.expectKeys {
				assert.True(t, foundKeys[expectedKey], "Expected key %s not found", expectedKey)
			}
		})
	}
}

func TestLocalDestination_GetURL(t *testing.T) {
	tempDir := t.TempDir()
	dest, err := NewLocalDestination(&btypes.LocalConfig{Path: tempDir})
	require.NoError(t, err)

	// Create a test file
	testFile := "test/backup.tar.gz"
	fullPath := filepath.Join(tempDir, testFile)
	require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
	require.NoError(t, os.WriteFile(fullPath, []byte("data"), 0644))

	// Get URL for existing file
	url, err := dest.GetURL(testFile, 1*time.Hour)
	assert.NoError(t, err)
	assert.Contains(t, url, "file://")
	assert.Contains(t, url, "backup.tar.gz")

	// Get URL for non-existent file
	url, err = dest.GetURL("missing.txt", 1*time.Hour)
	assert.Error(t, err)
	assert.Empty(t, url)
}

func TestLocalDestination_Type(t *testing.T) {
	tempDir := t.TempDir()
	dest, err := NewLocalDestination(&btypes.LocalConfig{Path: tempDir})
	require.NoError(t, err)

	assert.Equal(t, "local", dest.Type())
}

func TestLocalDestination_GetDiskUsage(t *testing.T) {
	tempDir := t.TempDir()
	dest, err := NewLocalDestination(&btypes.LocalConfig{Path: tempDir})
	require.NoError(t, err)

	// Initially empty
	usage, err := dest.GetDiskUsage()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), usage)

	// Add some files
	files := map[string]int{
		"file1.txt":        100,
		"dir/file2.txt":    200,
		"dir/sub/file3.gz": 300,
	}

	totalSize := int64(0)
	for path, size := range files {
		fullPath := filepath.Join(tempDir, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
		data := make([]byte, size)
		require.NoError(t, os.WriteFile(fullPath, data, 0644))
		totalSize += int64(size)
	}

	// Check usage
	usage, err = dest.GetDiskUsage()
	assert.NoError(t, err)
	assert.Equal(t, totalSize, usage)
}