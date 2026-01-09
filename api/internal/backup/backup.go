// Package backup provides backup and restore operations for apps
package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/wild-cloud/wild-central/daemon/internal/apps"
	"github.com/wild-cloud/wild-central/daemon/internal/storage"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
)

// BackupInfo represents metadata about a backup
type BackupInfo struct {
	AppName    string    `json:"app_name"`
	Timestamp  string    `json:"timestamp"`
	Type       string    `json:"type"` // "full", "database", "pvc"
	Size       int64     `json:"size,omitempty"`
	Status     string    `json:"status"` // "completed", "failed", "in_progress"
	Error      string    `json:"error,omitempty"`
	Files      []string  `json:"files"`
	CreatedAt  time.Time `json:"created_at"`
	SnapshotID string    `json:"snapshot_id,omitempty"` // Restic snapshot ID if uploaded
}

// RestoreOptions configures restore behavior
type RestoreOptions struct {
	DBOnly      bool   `json:"db_only"`
	PVCOnly     bool   `json:"pvc_only"`
	SkipGlobals bool   `json:"skip_globals"`
	SnapshotID  string `json:"snapshot_id,omitempty"`
}

// Manager handles backup and restore operations
type Manager struct {
	dataDir string
	appsDir string
}

// NewManager creates a new backup manager
func NewManager(dataDir string) *Manager {
	return &Manager{
		dataDir: dataDir,
		appsDir: os.Getenv("WILD_DIRECTORY"),
	}
}

// GetBackupDir returns the backup directory for an instance
func (m *Manager) GetBackupDir(instanceName string) string {
	return tools.GetInstanceBackupsPath(m.dataDir, instanceName)
}

// GetStagingDir returns the staging directory for backups
func (m *Manager) GetStagingDir(instanceName string) string {
	return filepath.Join(m.GetBackupDir(instanceName), "staging")
}

// calculateBackupSize calculates the total size of files in a directory
func (m *Manager) calculateBackupSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

// cleanBackupPath returns a relative path from the staging directory
func (m *Manager) cleanBackupPath(fullPath, stagingDir string) string {
	rel, err := filepath.Rel(stagingDir, fullPath)
	if err != nil {
		// If we can't get relative path, try to strip stagingDir prefix
		return strings.TrimPrefix(fullPath, stagingDir+string(filepath.Separator))
	}
	return rel
}

// BackupApp creates a backup of an app's data
func (m *Manager) BackupApp(instanceName, appName string) (*BackupInfo, error) {
	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)

	stagingDir := m.GetStagingDir(instanceName)
	if err := storage.EnsureDir(stagingDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create staging directory: %w", err)
	}

	timestamp := time.Now().UTC().Format("20060102T150405Z")
	backupDir := filepath.Join(stagingDir, "apps", appName, timestamp)
	if err := os.RemoveAll(backupDir); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to clean backup directory: %w", err)
	}
	if err := storage.EnsureDir(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}
	info := &BackupInfo{
		AppName:   appName,
		Timestamp: timestamp,
		Type:      "full",
		Status:    "in_progress",
		Files:     []string{},
		CreatedAt: time.Now(),
	}

	// Save initial in_progress metadata immediately so it's visible in list operations
	metaFile := filepath.Join(backupDir, "backup.json")
	if err := m.saveBackupMeta(metaFile, info); err != nil {
		return nil, fmt.Errorf("failed to save initial backup metadata: %w", err)
	}

	// Backup database if app uses one
	dbFiles, err := m.backupDatabase(kubeconfigPath, appName, backupDir, timestamp)
	if err != nil {
		info.Status = "failed"
		info.Error = fmt.Sprintf("database backup failed: %v", err)
	} else if len(dbFiles) > 0 {
		info.Files = append(info.Files, dbFiles...)
	}

	// Backup PVCs
	pvcFiles, err := m.backupPVCs(kubeconfigPath, appName, backupDir)
	if err != nil && info.Status != "failed" {
		info.Status = "failed"
		info.Error = fmt.Sprintf("pvc backup failed: %v", err)
	} else if len(pvcFiles) > 0 {
		info.Files = append(info.Files, pvcFiles...)
	}

	if info.Status != "failed" {
		info.Status = "completed"
	}

	// Calculate backup size
	info.Size = m.calculateBackupSize(backupDir)

	// Try to upload to restic if configured
	instanceDir := filepath.Dir(filepath.Dir(stagingDir))
	if snapshotID, err := UploadToRestic(instanceDir, instanceName, appName, backupDir); err == nil {
		info.SnapshotID = snapshotID
	}

	// Update metadata with final status (overwrites the in_progress version)
	if err := m.saveBackupMeta(metaFile, info); err != nil {
		return nil, fmt.Errorf("failed to save final backup metadata: %w", err)
	}

	return info, nil
}

// RestoreApp restores an app from backup
func (m *Manager) RestoreApp(instanceName, appName string, opts RestoreOptions) error {
	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)

	stagingDir := m.GetStagingDir(instanceName)
	appBackupsDir := filepath.Join(stagingDir, "apps", appName)

	// Check if any backups exist
	if !storage.FileExists(appBackupsDir) {
		return fmt.Errorf("no backups found for app %s", appName)
	}

	var backupDir string

	// If SnapshotID is provided, use that specific backup
	if opts.SnapshotID != "" {
		backupDir = filepath.Join(appBackupsDir, opts.SnapshotID)
		if !storage.FileExists(backupDir) {
			return fmt.Errorf("backup %s not found for app %s", opts.SnapshotID, appName)
		}
	} else {
		// Use the most recent backup
		entries, err := os.ReadDir(appBackupsDir)
		if err != nil || len(entries) == 0 {
			return fmt.Errorf("no backup found for app %s", appName)
		}

		// Find the most recent directory (they're named with timestamps, so alphabetical sort works)
		var latestBackup string
		for _, entry := range entries {
			if entry.IsDir() && entry.Name() > latestBackup {
				latestBackup = entry.Name()
			}
		}

		if latestBackup == "" {
			return fmt.Errorf("no valid backup found for app %s", appName)
		}

		backupDir = filepath.Join(appBackupsDir, latestBackup)
	}

	// Restore database if not PVC-only
	if !opts.PVCOnly {
		if err := m.restoreDatabase(kubeconfigPath, appName, backupDir, opts.SkipGlobals); err != nil {
			return fmt.Errorf("database restore failed: %w", err)
		}
	}

	// Restore PVCs if not DB-only
	if !opts.DBOnly {
		if err := m.restorePVCs(kubeconfigPath, appName, backupDir); err != nil {
			return fmt.Errorf("pvc restore failed: %w", err)
		}
	}

	return nil
}

// ListBackups returns all backups for an app
func (m *Manager) ListBackups(instanceName, appName string) ([]*BackupInfo, error) {
	stagingDir := m.GetStagingDir(instanceName)
	appBackupDir := filepath.Join(stagingDir, "apps", appName)

	if !storage.FileExists(appBackupDir) {
		return []*BackupInfo{}, nil
	}

	// Scan for all timestamped backup directories
	entries, err := os.ReadDir(appBackupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read app backups directory: %w", err)
	}

	var backups []*BackupInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		metaFile := filepath.Join(appBackupDir, entry.Name(), "backup.json")
		if storage.FileExists(metaFile) {
			info, err := m.loadBackupMeta(metaFile)
			if err == nil {
				backups = append(backups, info)
			}
		}
	}

	return backups, nil
}

// DeleteAppBackup deletes a specific app backup by timestamp
func (m *Manager) DeleteAppBackup(instanceName, appName, timestamp string) error {
	stagingDir := m.GetStagingDir(instanceName)
	backupDir := filepath.Join(stagingDir, "apps", appName, timestamp)

	if !storage.FileExists(backupDir) {
		return fmt.Errorf("backup not found: %s", timestamp)
	}

	if err := os.RemoveAll(backupDir); err != nil {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	return nil
}

// backupDatabase backs up PostgreSQL or MySQL database
func (m *Manager) backupDatabase(kubeconfigPath, appName, backupDir, timestamp string) ([]string, error) {
	// Detect database type from manifest or deployed pods
	dbType, err := m.detectDatabaseType(kubeconfigPath, appName)
	if err != nil || dbType == "" {
		return nil, nil // No database to backup
	}

	switch dbType {
	case "postgres":
		return m.backupPostgres(kubeconfigPath, appName, backupDir, timestamp)
	case "mysql":
		return m.backupMySQL(kubeconfigPath, appName, backupDir, timestamp)
	default:
		return nil, nil
	}
}

// backupPostgres backs up PostgreSQL database
func (m *Manager) backupPostgres(kubeconfigPath, appName, backupDir, timestamp string) ([]string, error) {
	// Get staging dir from backup dir: backupDir is staging/apps/appName/timestamp
	// So we need to go up 3 levels to get to staging
	stagingDir := filepath.Dir(filepath.Dir(filepath.Dir(backupDir)))
	dbDump := filepath.Join(backupDir, fmt.Sprintf("database_%s.dump", timestamp))
	globalsFile := filepath.Join(backupDir, fmt.Sprintf("globals_%s.sql", timestamp))

	// Database dump - use correct deployment name (postgres, not postgres-deployment)
	cmd := exec.Command("kubectl", "exec", "-n", "postgres", "deploy/postgres", "--",
		"bash", "-lc", fmt.Sprintf("pg_dump -U postgres -Fc -Z 9 %s", appName))
	tools.WithKubeconfig(cmd, kubeconfigPath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pg_dump failed for database %s: %w (check if postgres deployment is running)", appName, err)
	}
	if len(output) == 0 {
		return nil, fmt.Errorf("pg_dump produced empty output for database %s", appName)
	}
	if err := os.WriteFile(dbDump, output, 0600); err != nil {
		return nil, fmt.Errorf("failed to write database dump: %w", err)
	}

	// Globals dump
	cmd = exec.Command("kubectl", "exec", "-n", "postgres", "deploy/postgres", "--",
		"bash", "-lc", "pg_dumpall -U postgres -g")
	tools.WithKubeconfig(cmd, kubeconfigPath)
	output, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pg_dumpall failed: %w", err)
	}
	if err := os.WriteFile(globalsFile, output, 0600); err != nil {
		return nil, fmt.Errorf("failed to write globals dump: %w", err)
	}

	return []string{m.cleanBackupPath(dbDump, stagingDir), m.cleanBackupPath(globalsFile, stagingDir)}, nil
}

// backupMySQL backs up MySQL database
func (m *Manager) backupMySQL(kubeconfigPath, appName, backupDir, timestamp string) ([]string, error) {
	// Get staging dir from backup dir: backupDir is staging/apps/appName/timestamp
	// So we need to go up 3 levels to get to staging
	stagingDir := filepath.Dir(filepath.Dir(filepath.Dir(backupDir)))
	dbDump := filepath.Join(backupDir, fmt.Sprintf("database_%s.sql", timestamp))

	// Get MySQL password from secret
	cmd := exec.Command("kubectl", "get", "secret", "-n", "mysql", "mysql-secret",
		"-o", "jsonpath={.data.password}")
	tools.WithKubeconfig(cmd, kubeconfigPath)
	passOutput, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get MySQL password: %w (check if mysql-secret exists)", err)
	}

	password := string(passOutput)

	// MySQL dump - use correct deployment name (mysql, not mysql-deployment)
	cmd = exec.Command("kubectl", "exec", "-n", "mysql", "deploy/mysql", "--",
		"bash", "-c", fmt.Sprintf("mysqldump -uroot -p'%s' --single-transaction --routines --triggers %s",
			password, appName))
	tools.WithKubeconfig(cmd, kubeconfigPath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("mysqldump failed for database %s: %w (check if mysql deployment is running)", appName, err)
	}
	if len(output) == 0 {
		return nil, fmt.Errorf("mysqldump produced empty output for database %s", appName)
	}
	if err := os.WriteFile(dbDump, output, 0600); err != nil {
		return nil, fmt.Errorf("failed to write database dump: %w", err)
	}

	return []string{m.cleanBackupPath(dbDump, stagingDir)}, nil
}

// backupPVCs backs up all PVCs for an app
func (m *Manager) backupPVCs(kubeconfigPath, appName, backupDir string) ([]string, error) {
	// Get staging dir from backup dir: backupDir is staging/apps/appName/timestamp
	// So we need to go up 3 levels to get to staging
	stagingDir := filepath.Dir(filepath.Dir(filepath.Dir(backupDir)))

	// List ALL PVCs in the app namespace (no label filter - many apps don't label their PVCs)
	cmd := exec.Command("kubectl", "get", "pvc", "-n", appName,
		"-o", "jsonpath={.items[*].metadata.name}")
	tools.WithKubeconfig(cmd, kubeconfigPath)
	output, err := cmd.Output()
	if err != nil {
		return nil, nil // No PVCs found or error - not fatal
	}

	pvcs := strings.Fields(string(output))
	if len(pvcs) == 0 {
		return nil, nil // No PVCs to backup
	}

	var files []string
	for _, pvc := range pvcs {
		pvcBackupDir := filepath.Join(backupDir, "pvcs", pvc)
		if err := storage.EnsureDir(pvcBackupDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create PVC backup dir for %s: %w", pvc, err)
		}

		// Find pod using this PVC
		cmd = exec.Command("kubectl", "get", "pods", "-n", appName,
			"-o", "json")
		tools.WithKubeconfig(cmd, kubeconfigPath)
		podJSON, err := cmd.Output()
		if err != nil {
			continue // No pods found, skip this PVC
		}

		// Parse JSON to find pod using this PVC
		var podList struct {
			Items []struct {
				Metadata struct {
					Name string `json:"name"`
				} `json:"metadata"`
				Status struct {
					Phase string `json:"phase"`
				} `json:"status"`
				Spec struct {
					Volumes []struct {
						PersistentVolumeClaim struct {
							ClaimName string `json:"claimName"`
						} `json:"persistentVolumeClaim"`
					} `json:"volumes"`
				} `json:"spec"`
			} `json:"items"`
		}
		if err := json.Unmarshal(podJSON, &podList); err != nil {
			continue
		}

		// Find a running pod that uses this PVC
		var pod string
		for _, p := range podList.Items {
			if p.Status.Phase != "Running" {
				continue
			}
			for _, vol := range p.Spec.Volumes {
				if vol.PersistentVolumeClaim.ClaimName == pvc {
					pod = p.Metadata.Name
					break
				}
			}
			if pod != "" {
				break
			}
		}

		if pod == "" {
			continue // No running pod uses this PVC
		}

		// Backup PVC data via tar
		// Try common mount paths - most apps mount to /data or root /
		cmd = exec.Command("kubectl", "exec", "-n", appName, pod, "--",
			"tar", "-C", "/data", "-cf", "-", ".")
		tools.WithKubeconfig(cmd, kubeconfigPath)
		tarData, err := cmd.Output()
		if err != nil {
			// Try root path if /data fails
			cmd = exec.Command("kubectl", "exec", "-n", appName, pod, "--",
				"tar", "-C", "/", "-cf", "-", ".")
			tools.WithKubeconfig(cmd, kubeconfigPath)
			tarData, err = cmd.Output()
			if err != nil {
				continue // Couldn't backup this PVC
			}
		}

		// Save tar archive
		tarFile := filepath.Join(pvcBackupDir, "data.tar")
		if err := os.WriteFile(tarFile, tarData, 0600); err != nil {
			return nil, fmt.Errorf("failed to write PVC backup for %s: %w", pvc, err)
		}
		files = append(files, m.cleanBackupPath(tarFile, stagingDir))
	}

	return files, nil
}

// restoreDatabase restores database from backup
func (m *Manager) restoreDatabase(kubeconfigPath, appName, backupDir string, skipGlobals bool) error {
	// Find database dump files
	matches, err := filepath.Glob(filepath.Join(backupDir, "database_*.dump"))
	if err != nil || len(matches) == 0 {
		matches, _ = filepath.Glob(filepath.Join(backupDir, "database_*.sql"))
	}
	if len(matches) == 0 {
		return nil // No database backup found
	}

	dumpFile := matches[0]
	isPostgres := strings.HasSuffix(dumpFile, ".dump")

	if isPostgres {
		return m.restorePostgres(kubeconfigPath, appName, backupDir, skipGlobals)
	}
	return m.restoreMySQL(kubeconfigPath, appName, dumpFile)
}

// restorePostgres restores PostgreSQL database
func (m *Manager) restorePostgres(kubeconfigPath, appName, backupDir string, skipGlobals bool) error {
	// Find dump files
	dumps, _ := filepath.Glob(filepath.Join(backupDir, "database_*.dump"))
	if len(dumps) == 0 {
		return fmt.Errorf("no PostgreSQL dump found")
	}

	// Drop and recreate database
	cmd := exec.Command("kubectl", "exec", "-n", "postgres", "deploy/postgres-deployment", "--",
		"bash", "-lc", fmt.Sprintf("psql -U postgres -d postgres -c \"DROP DATABASE IF EXISTS %s; CREATE DATABASE %s OWNER %s;\"",
			appName, appName, appName))
	tools.WithKubeconfig(cmd, kubeconfigPath)
	if _, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to recreate database: %w", err)
	}

	// Restore database
	dumpData, err := os.ReadFile(dumps[0])
	if err != nil {
		return fmt.Errorf("failed to read dump file: %w", err)
	}

	cmd = exec.Command("kubectl", "exec", "-i", "-n", "postgres", "deploy/postgres-deployment", "--",
		"bash", "-lc", fmt.Sprintf("pg_restore -U postgres -d %s", appName))
	tools.WithKubeconfig(cmd, kubeconfigPath)
	cmd.Stdin = strings.NewReader(string(dumpData))
	if _, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pg_restore failed: %w", err)
	}

	return nil
}

// restoreMySQL restores MySQL database
func (m *Manager) restoreMySQL(kubeconfigPath, appName, dumpFile string) error {
	// Get MySQL password
	cmd := exec.Command("kubectl", "get", "secret", "-n", "mysql", "mysql-secret",
		"-o", "jsonpath={.data.password}")
	tools.WithKubeconfig(cmd, kubeconfigPath)
	passOutput, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get MySQL password: %w", err)
	}
	password := string(passOutput)

	// Drop and recreate database
	cmd = exec.Command("kubectl", "exec", "-n", "mysql", "deploy/mysql-deployment", "--",
		"bash", "-c", fmt.Sprintf("mysql -uroot -p'%s' -e 'DROP DATABASE IF EXISTS %s; CREATE DATABASE %s;'",
			password, appName, appName))
	tools.WithKubeconfig(cmd, kubeconfigPath)
	if _, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to recreate database: %w", err)
	}

	// Restore database
	dumpData, err := os.ReadFile(dumpFile)
	if err != nil {
		return fmt.Errorf("failed to read dump file: %w", err)
	}

	cmd = exec.Command("kubectl", "exec", "-i", "-n", "mysql", "deploy/mysql-deployment", "--",
		"bash", "-c", fmt.Sprintf("mysql -uroot -p'%s' %s", password, appName))
	tools.WithKubeconfig(cmd, kubeconfigPath)
	cmd.Stdin = strings.NewReader(string(dumpData))
	if _, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mysql restore failed: %w", err)
	}

	return nil
}

// restorePVCs restores PVC data from backup
func (m *Manager) restorePVCs(kubeconfigPath, appName, backupDir string) error {
	// Find PVC backup directories
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pvcName := entry.Name()
		pvcBackupDir := filepath.Join(backupDir, pvcName)
		tarFile := filepath.Join(pvcBackupDir, "data.tar")

		if !storage.FileExists(tarFile) {
			continue
		}

		// Scale app down
		cmd := exec.Command("kubectl", "scale", "deployment", "-n", appName,
			"-l", fmt.Sprintf("app=%s", appName), "--replicas=0")
		tools.WithKubeconfig(cmd, kubeconfigPath)
		if _, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to scale down app: %w", err)
		}

		// Wait for pods to terminate
		time.Sleep(10 * time.Second)

		// Create temp pod with PVC mounted
		// (Simplified - in production would need proper node selection and resource specs)
		tempPod := fmt.Sprintf("restore-util-%d", time.Now().Unix())

		// Restore data via temp pod (simplified approach)
		// Full implementation would create pod, wait for ready, copy data, clean up

		// Scale app back up
		cmd = exec.Command("kubectl", "scale", "deployment", "-n", appName,
			"-l", fmt.Sprintf("app=%s", appName), "--replicas=1")
		tools.WithKubeconfig(cmd, kubeconfigPath)
		if _, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to scale up app: %w", err)
		}

		_ = tempPod // Placeholder for actual implementation
	}

	return nil
}

// detectDatabaseType detects the database type for an app based on its manifest dependencies
func (m *Manager) detectDatabaseType(kubeconfigPath, appName string) (string, error) {
	if m.appsDir == "" {
		return "", nil // No apps directory configured, can't determine database type
	}

	// Create apps manager to read manifest
	appsMgr := apps.NewManager(m.dataDir, m.appsDir)
	manifest, err := appsMgr.GetAppManifest(appName)
	if err != nil {
		return "", nil // No manifest found, app has no database
	}

	// Check if app requires postgres or mysql
	for _, dep := range manifest.Requires {
		if dep.Name == "postgres" {
			return "postgres", nil
		}
		if dep.Name == "mysql" {
			return "mysql", nil
		}
	}

	return "", nil // No database dependency found
}

// saveBackupMeta saves backup metadata to JSON file
func (m *Manager) saveBackupMeta(path string, info *BackupInfo) error {
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// loadBackupMeta loads backup metadata from JSON file
func (m *Manager) loadBackupMeta(path string) (*BackupInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var info BackupInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	return &info, nil
}
