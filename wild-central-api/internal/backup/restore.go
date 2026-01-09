package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wild-cloud/wild-central/daemon/internal/storage"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
)

// RestoreFromSnapshot restores an app from a restic snapshot
func (m *Manager) RestoreFromSnapshot(instanceName, appName, snapshotID string, opts RestoreOptions) error {
	// Create restic client
	instanceDir := tools.GetInstancePath(m.dataDir, instanceName)
	client, err := NewResticClient(instanceDir)
	if err != nil {
		return fmt.Errorf("failed to create restic client: %w", err)
	}

	// Verify snapshot exists and belongs to this app
	snapshot, err := client.GetSnapshot(snapshotID)
	if err != nil {
		return fmt.Errorf("failed to get snapshot: %w", err)
	}

	// Validate snapshot belongs to this app
	hasAppTag := false
	expectedTag := fmt.Sprintf("app:%s", appName)
	for _, tag := range snapshot.Tags {
		if tag == expectedTag {
			hasAppTag = true
			break
		}
	}
	if !hasAppTag {
		return fmt.Errorf("snapshot %s does not belong to app %s", snapshotID, appName)
	}

	// Create temporary restore directory
	stagingDir := m.GetStagingDir(instanceName)
	restoreDir := filepath.Join(stagingDir, "restore-temp", snapshotID)
	if err := os.RemoveAll(restoreDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clean restore directory: %w", err)
	}
	if err := storage.EnsureDir(restoreDir, 0755); err != nil {
		return fmt.Errorf("failed to create restore directory: %w", err)
	}

	// Download snapshot
	if err := client.RestoreSnapshot(snapshotID, restoreDir); err != nil {
		return fmt.Errorf("failed to restore snapshot: %w", err)
	}

	// Find backup data directory within restored snapshot
	backupDataDir, err := findBackupDataDir(restoreDir, appName)
	if err != nil {
		return fmt.Errorf("failed to locate backup data: %w", err)
	}

	// Restore using existing restore logic
	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)

	// Restore database if not PVC-only
	if !opts.PVCOnly {
		if err := m.restoreDatabase(kubeconfigPath, appName, backupDataDir, opts.SkipGlobals); err != nil {
			return fmt.Errorf("database restore failed: %w", err)
		}
	}

	// Restore PVCs if not DB-only
	if !opts.DBOnly {
		if err := m.restorePVCs(kubeconfigPath, appName, backupDataDir); err != nil {
			return fmt.Errorf("pvc restore failed: %w", err)
		}
	}

	// Clean up restore directory
	if err := os.RemoveAll(restoreDir); err != nil {
		return fmt.Errorf("restore completed but cleanup failed: %w", err)
	}

	return nil
}

// findBackupDataDir locates the backup data directory within a restored snapshot
// The snapshot structure is: restoreDir/[full-path]/apps/appName/timestamp/
func findBackupDataDir(restoreDir, appName string) (string, error) {
	// Walk the restore directory to find apps/appName/timestamp structure
	var foundDir string

	err := filepath.Walk(restoreDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Look for backup.json which marks a backup directory
		if info.Name() == "backup.json" {
			// Verify this is for the correct app
			dirPath := filepath.Dir(path)
			if strings.Contains(dirPath, fmt.Sprintf("/apps/%s/", appName)) {
				foundDir = dirPath
				return filepath.SkipDir
			}
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	if foundDir == "" {
		return "", fmt.Errorf("backup data not found for app %s in snapshot", appName)
	}

	return foundDir, nil
}

// ListSnapshotsForInstance lists all snapshots for an instance
func ListSnapshotsForInstance(dataDir, instanceName string, tags ...string) ([]ResticSnapshot, error) {
	instanceDir := tools.GetInstancePath(dataDir, instanceName)
	client, err := NewResticClient(instanceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create restic client: %w", err)
	}

	if !client.IsInitialized() {
		return []ResticSnapshot{}, nil
	}

	// Add instance tag to filter
	allTags := append([]string{fmt.Sprintf("instance:%s", instanceName)}, tags...)
	return client.Snapshots(allTags...)
}

// ListSnapshotsForApp lists all snapshots for a specific app
func ListSnapshotsForApp(dataDir, instanceName, appName string) ([]ResticSnapshot, error) {
	return ListSnapshotsForInstance(dataDir, instanceName, fmt.Sprintf("app:%s", appName))
}
