package backup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wild-cloud/wild-central/daemon/internal/apps"
)

func TestRestoreIntegration(t *testing.T) {
	// Skip this test in CI - it's for manual verification
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test - set RUN_INTEGRATION_TESTS=true to run")
	}

	tempDir := t.TempDir()
	instanceName := "test-instance"
	appName := "test-app"
	
	// Set up instance directory structure
	instanceDir := filepath.Join(tempDir, "instances", instanceName)
	appDir := filepath.Join(instanceDir, "apps", appName)
	require.NoError(t, os.MkdirAll(appDir, 0755))
	
	// Create manifest with postgres dependency
	manifestPath := filepath.Join(appDir, "manifest.yaml")
	manifestContent := `
name: test-app
description: Test application
requires:
  - name: postgres
defaultConfig:
  dbName: testdb
`
	require.NoError(t, os.WriteFile(manifestPath, []byte(manifestContent), 0644))
	
	// Create config
	configPath := filepath.Join(instanceDir, "config.yaml")
	configContent := `
backup:
  destination:
    type: local
    local:
      path: ` + filepath.Join(tempDir, "backups") + `
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))
	
	// Create manager
	mgr := NewManager(tempDir)
	
	// Create mock strategies for backup and restore
	mockConfigBackup := &ComponentBackup{
		Type:     "config",
		Name:     "config.test-app",
		Size:     100,
		Location: "config/test.tar.gz",
	}
	
	mockPostgresBackup := &ComponentBackup{
		Type:     "postgres", 
		Name:     "postgres.testdb",
		Size:     200,
		Location: "postgres/test.dump",
		Metadata: map[string]interface{}{
			"database": "testdb",
		},
	}
	
	// Mock strategies
	restoreCalled := map[string]bool{}
	
	mockConfigStrategy := &MockStrategy{
		Name_: "config",
		BackupFunc: func(inst, app string, manifest *apps.AppManifest, dest BackupDestination) (*ComponentBackup, error) {
			return mockConfigBackup, nil
		},
		RestoreFunc: func(backup *ComponentBackup, dest BackupDestination) error {
			restoreCalled["config"] = true
			assert.Equal(t, mockConfigBackup.Location, backup.Location)
			return nil
		},
	}
	
	mockPostgresStrategy := &MockStrategy{
		Name_: "postgres",
		BackupFunc: func(inst, app string, manifest *apps.AppManifest, dest BackupDestination) (*ComponentBackup, error) {
			return mockPostgresBackup, nil
		},
		RestoreFunc: func(backup *ComponentBackup, dest BackupDestination) error {
			restoreCalled["postgres"] = true
			assert.Equal(t, mockPostgresBackup.Location, backup.Location)
			return nil
		},
		SupportsFunc: func(manifest *apps.AppManifest) bool {
			for _, dep := range manifest.Requires {
				if dep.Name == "postgres" {
					return true
				}
			}
			return false
		},
	}
	
	mgr.strategies = map[string]Strategy{
		"config":   mockConfigStrategy,
		"postgres": mockPostgresStrategy,
	}
	
	// First create a backup
	backupInfo, err := mgr.BackupApp(instanceName, appName)
	require.NoError(t, err)
	assert.NotNil(t, backupInfo)
	assert.Equal(t, "completed", backupInfo.Status)
	assert.Len(t, backupInfo.Components, 2)
	
	// Now test restore
	opts := RestoreOptions{
		Components: []string{}, // Restore all components
	}
	
	err = mgr.RestoreApp(instanceName, appName, opts)
	require.NoError(t, err)
	
	// Verify both strategies were called
	assert.True(t, restoreCalled["config"], "Config restore should have been called")
	assert.True(t, restoreCalled["postgres"], "Postgres restore should have been called")
}
