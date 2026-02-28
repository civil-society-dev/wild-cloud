package backup

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wild-cloud/wild-central/daemon/internal/apps"
)

// MockStrategy implements the Strategy interface for testing
type MockStrategy struct {
	Name_        string
	BackupFunc   func(string, string, *apps.AppManifest, BackupDestination) (*ComponentBackup, error)
	RestoreFunc  func(*ComponentBackup, BackupDestination) error
	VerifyFunc   func(*ComponentBackup, BackupDestination) error
	SupportsFunc func(*apps.AppManifest) bool
}

func (m *MockStrategy) Name() string {
	return m.Name_
}

func (m *MockStrategy) Backup(instanceName, appName string, manifest *apps.AppManifest, dest BackupDestination) (*ComponentBackup, error) {
	if m.BackupFunc != nil {
		return m.BackupFunc(instanceName, appName, manifest, dest)
	}
	return &ComponentBackup{
		Type:     m.Name_,
		Name:     appName + "-" + m.Name_,
		Size:     1024,
		Location: "test/location",
	}, nil
}

func (m *MockStrategy) Restore(component *ComponentBackup, dest BackupDestination) error {
	if m.RestoreFunc != nil {
		return m.RestoreFunc(component, dest)
	}
	return nil
}

func (m *MockStrategy) Verify(component *ComponentBackup, dest BackupDestination) error {
	if m.VerifyFunc != nil {
		return m.VerifyFunc(component, dest)
	}
	return nil
}

func (m *MockStrategy) Supports(manifest *apps.AppManifest) bool {
	if m.SupportsFunc != nil {
		return m.SupportsFunc(manifest)
	}
	return true
}

// MockDestination implements BackupDestination for testing
type MockDestination struct {
	PutFunc    func(string, io.Reader) (int64, error)
	GetFunc    func(string) (io.ReadCloser, error)
	DeleteFunc func(string) error
	ListFunc   func(string) ([]BackupObject, error)
	GetURLFunc func(string, time.Duration) (string, error)
	TypeFunc   func() string
}

func (m *MockDestination) Put(key string, reader io.Reader) (int64, error) {
	if m.PutFunc != nil {
		return m.PutFunc(key, reader)
	}
	// Default: consume the reader and return a size
	data, _ := io.ReadAll(reader)
	return int64(len(data)), nil
}

func (m *MockDestination) Get(key string) (io.ReadCloser, error) {
	if m.GetFunc != nil {
		return m.GetFunc(key)
	}
	return io.NopCloser(strings.NewReader("test data")), nil
}

func (m *MockDestination) Delete(key string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(key)
	}
	return nil
}

func (m *MockDestination) List(prefix string) ([]BackupObject, error) {
	if m.ListFunc != nil {
		return m.ListFunc(prefix)
	}
	return []BackupObject{}, nil
}

func (m *MockDestination) GetURL(key string, expiry time.Duration) (string, error) {
	if m.GetURLFunc != nil {
		return m.GetURLFunc(key, expiry)
	}
	return "https://example.com/backup/" + key, nil
}

func (m *MockDestination) Type() string {
	if m.TypeFunc != nil {
		return m.TypeFunc()
	}
	return "mock"
}

func TestBackupApp(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Set up test data directory structure
	instanceName := "test-instance"
	appName := "test-app"

	// Create instance directory
	instanceDir := filepath.Join(tempDir, "instances", instanceName)
	appsDir := filepath.Join(instanceDir, "apps", appName)
	backupsDir := filepath.Join(instanceDir, "backups")

	require.NoError(t, os.MkdirAll(appsDir, 0755))
	require.NoError(t, os.MkdirAll(backupsDir, 0755))

	// Create test manifest with postgres dependency
	manifestPath := filepath.Join(appsDir, "manifest.yaml")
	manifestContent := `
name: test-app
description: Test application
version: 1.0.0
requires:
  - name: postgres
defaultConfig:
  image: test:latest
`
	require.NoError(t, os.WriteFile(manifestPath, []byte(manifestContent), 0644))

	// Create test config
	configPath := filepath.Join(instanceDir, "config.yaml")
	configContent := `
backup:
  destination:
    type: local
    local:
      path: ` + backupsDir + `
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	// Create manager with mock strategy
	mgr := NewManager(tempDir)
	mockStrategy := &MockStrategy{
		Name_: "test",
		BackupFunc: func(inst, app string, manifest *apps.AppManifest, dest BackupDestination) (*ComponentBackup, error) {
			assert.Equal(t, instanceName, inst)
			assert.Equal(t, appName, app)
			return &ComponentBackup{
				Type:     "test",
				Name:     "test-component",
				Size:     2048,
				Location: "backups/test-component.tar.gz",
			}, nil
		},
	}

	// Replace strategies with our mock
	mgr.strategies = map[string]Strategy{
		"postgres": mockStrategy, // Postgres strategy for dependency
		"config":   mockStrategy, // Config is always included
	}

	// Perform backup
	info, err := mgr.BackupApp(instanceName, appName)
	require.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, appName, info.AppName)
	assert.Equal(t, "completed", info.Status)
	assert.Len(t, info.Components, 2) // postgres + config
	assert.Equal(t, int64(4096), info.Size) // 2048 * 2
}

func TestRestoreApp(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	instanceName := "test-instance"
	appName := "test-app"
	timestamp := time.Now().UTC().Format("20060102T150405Z")

	// Set up directory structure
	instanceDir := filepath.Join(tempDir, "instances", instanceName)
	backupsDir := filepath.Join(instanceDir, "backups", appName, timestamp)
	require.NoError(t, os.MkdirAll(backupsDir, 0755))

	// Create backup metadata
	metadataPath := filepath.Join(backupsDir, "metadata.json")
	metadata := &BackupInfo{
		AppName:   appName,
		Timestamp: timestamp,
		Type:      "full",
		Status:    "completed",
		Components: []ComponentBackup{
			{
				Type:     "test",
				Name:     "test-component",
				Size:     1024,
				Location: "test/backup.tar.gz",
			},
		},
		CreatedAt: time.Now(),
	}

	metadataJSON, _ := json.MarshalIndent(metadata, "", "  ")
	require.NoError(t, os.WriteFile(metadataPath, metadataJSON, 0644))

	// Create config
	configPath := filepath.Join(instanceDir, "config.yaml")
	configContent := `
backup:
  destination:
    type: local
    local:
      path: ` + filepath.Join(instanceDir, "backup-storage") + `
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	// Create manager with mock strategy
	mgr := NewManager(tempDir)

	restoreCalled := false
	mockStrategy := &MockStrategy{
		Name_: "test",
		RestoreFunc: func(component *ComponentBackup, dest BackupDestination) error {
			restoreCalled = true
			assert.Equal(t, "test", component.Type)
			assert.Equal(t, "test-component", component.Name)
			return nil
		},
	}

	mgr.strategies = map[string]Strategy{
		"test": mockStrategy,
	}

	// Perform restore
	err := mgr.RestoreApp(instanceName, appName, RestoreOptions{})
	require.NoError(t, err)
	assert.True(t, restoreCalled, "Restore strategy should have been called")
}

func TestListBackups(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	instanceName := "test-instance"
	appName := "test-app"

	// Create multiple backups
	backupsDir := filepath.Join(tempDir, "instances", instanceName, "backups", appName)

	timestamps := []string{
		"20240101T120000Z",
		"20240102T120000Z",
		"20240103T120000Z",
	}

	for _, ts := range timestamps {
		backupDir := filepath.Join(backupsDir, ts)
		require.NoError(t, os.MkdirAll(backupDir, 0755))

		metadata := &BackupInfo{
			AppName:   appName,
			Timestamp: ts,
			Status:    "completed",
			CreatedAt: time.Now(),
		}

		metadataJSON, _ := json.MarshalIndent(metadata, "", "  ")
		metadataPath := filepath.Join(backupDir, "metadata.json")
		require.NoError(t, os.WriteFile(metadataPath, metadataJSON, 0644))
	}

	// List backups
	mgr := NewManager(tempDir)
	backups, err := mgr.ListBackups(instanceName, appName)
	require.NoError(t, err)
	assert.Len(t, backups, 3)

	// Check all timestamps are present
	foundTimestamps := make(map[string]bool)
	for _, backup := range backups {
		foundTimestamps[backup.Timestamp] = true
	}

	for _, ts := range timestamps {
		assert.True(t, foundTimestamps[ts], "Timestamp %s should be in list", ts)
	}
}

func TestDeleteAppBackup(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	instanceName := "test-instance"
	appName := "test-app"
	timestamp := "20240101T120000Z"

	// Create backup directory and metadata
	backupDir := filepath.Join(tempDir, "instances", instanceName, "backups", appName, timestamp)
	require.NoError(t, os.MkdirAll(backupDir, 0755))

	metadata := &BackupInfo{
		AppName:   appName,
		Timestamp: timestamp,
		Status:    "completed",
		Components: []ComponentBackup{
			{
				Type:     "test",
				Location: "test/backup.tar.gz",
			},
		},
		CreatedAt: time.Now(),
	}

	metadataJSON, _ := json.MarshalIndent(metadata, "", "  ")
	metadataPath := filepath.Join(backupDir, "metadata.json")
	require.NoError(t, os.WriteFile(metadataPath, metadataJSON, 0644))

	// Create config
	configPath := filepath.Join(tempDir, "instances", instanceName, "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
	configContent := `
backup:
  destination:
    type: local
    local:
      path: /tmp/backups
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	// Delete backup
	mgr := NewManager(tempDir)
	err := mgr.DeleteAppBackup(instanceName, appName, timestamp)
	require.NoError(t, err)

	// Verify backup directory is deleted
	_, err = os.Stat(backupDir)
	assert.True(t, os.IsNotExist(err), "Backup directory should be deleted")
}

func TestVerifyBackup(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	instanceName := "test-instance"
	appName := "test-app"
	timestamp := "20240101T120000Z"

	// Set up backup metadata
	backupDir := filepath.Join(tempDir, "instances", instanceName, "backups", appName, timestamp)
	require.NoError(t, os.MkdirAll(backupDir, 0755))

	metadata := &BackupInfo{
		AppName:   appName,
		Timestamp: timestamp,
		Status:    "completed",
		Components: []ComponentBackup{
			{
				Type:     "test",
				Name:     "component1",
				Location: "test/comp1.tar.gz",
			},
			{
				Type:     "config",
				Name:     "config",
				Location: "test/config.tar.gz",
			},
		},
		CreatedAt: time.Now(),
	}

	metadataJSON, _ := json.MarshalIndent(metadata, "", "  ")
	require.NoError(t, os.WriteFile(filepath.Join(backupDir, "metadata.json"), metadataJSON, 0644))

	// Create config
	configPath := filepath.Join(tempDir, "instances", instanceName, "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
	configContent := `
backup:
  destination:
    type: local
    local:
      path: /tmp/backups
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	// Create manager with mock strategies
	mgr := NewManager(tempDir)

	testVerifyCount := 0
	configVerifyCount := 0

	mgr.strategies = map[string]Strategy{
		"test": &MockStrategy{
			Name_: "test",
			VerifyFunc: func(component *ComponentBackup, dest BackupDestination) error {
				testVerifyCount++
				return nil // Success
			},
		},
		"config": &MockStrategy{
			Name_: "config",
			VerifyFunc: func(component *ComponentBackup, dest BackupDestination) error {
				configVerifyCount++
				return nil // Success
			},
		},
	}

	// Verify backup
	result, err := mgr.VerifyBackup(instanceName, appName, timestamp)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Len(t, result.Components, 2)
	assert.Equal(t, 1, testVerifyCount)
	assert.Equal(t, 1, configVerifyCount)

	// Check all components verified successfully
	for _, comp := range result.Components {
		assert.True(t, comp.Success)
		assert.Empty(t, comp.Error)
	}
}