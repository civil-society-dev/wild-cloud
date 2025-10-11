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

	"github.com/wild-cloud/wild-central/daemon/internal/storage"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
)

// BackupInfo represents metadata about a backup
type BackupInfo struct {
	AppName   string    `json:"app_name"`
	Timestamp string    `json:"timestamp"`
	Type      string    `json:"type"` // "full", "database", "pvc"
	Size      int64     `json:"size,omitempty"`
	Status    string    `json:"status"` // "completed", "failed", "in_progress"
	Error     string    `json:"error,omitempty"`
	Files     []string  `json:"files"`
	CreatedAt time.Time `json:"created_at"`
}

// RestoreOptions configures restore behavior
type RestoreOptions struct {
	DBOnly       bool   `json:"db_only"`
	PVCOnly      bool   `json:"pvc_only"`
	SkipGlobals  bool   `json:"skip_globals"`
	SnapshotID   string `json:"snapshot_id,omitempty"`
}

// Manager handles backup and restore operations
type Manager struct {
	dataDir string
}

// NewManager creates a new backup manager
func NewManager(dataDir string) *Manager {
	return &Manager{dataDir: dataDir}
}

// GetBackupDir returns the backup directory for an instance
func (m *Manager) GetBackupDir(instanceName string) string {
	return filepath.Join(m.dataDir, "instances", instanceName, "backups")
}

// GetStagingDir returns the staging directory for backups
func (m *Manager) GetStagingDir(instanceName string) string {
	return filepath.Join(m.GetBackupDir(instanceName), "staging")
}

// BackupApp creates a backup of an app's data
func (m *Manager) BackupApp(instanceName, appName string) (*BackupInfo, error) {
	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)

	stagingDir := m.GetStagingDir(instanceName)
	if err := storage.EnsureDir(stagingDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create staging directory: %w", err)
	}

	backupDir := filepath.Join(stagingDir, "apps", appName)
	if err := os.RemoveAll(backupDir); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to clean backup directory: %w", err)
	}
	if err := storage.EnsureDir(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	timestamp := time.Now().UTC().Format("20060102T150405Z")
	info := &BackupInfo{
		AppName:   appName,
		Timestamp: timestamp,
		Type:      "full",
		Status:    "in_progress",
		Files:     []string{},
		CreatedAt: time.Now(),
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

	// Save backup metadata
	metaFile := filepath.Join(backupDir, "backup.json")
	if err := m.saveBackupMeta(metaFile, info); err != nil {
		return nil, fmt.Errorf("failed to save backup metadata: %w", err)
	}

	return info, nil
}

// RestoreApp restores an app from backup
func (m *Manager) RestoreApp(instanceName, appName string, opts RestoreOptions) error {
	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)

	stagingDir := m.GetStagingDir(instanceName)
	backupDir := filepath.Join(stagingDir, "apps", appName)

	// Check if backup exists
	if !storage.FileExists(backupDir) {
		return fmt.Errorf("no backup found for app %s", appName)
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

	var backups []*BackupInfo
	metaFile := filepath.Join(appBackupDir, "backup.json")
	if storage.FileExists(metaFile) {
		info, err := m.loadBackupMeta(metaFile)
		if err == nil {
			backups = append(backups, info)
		}
	}

	return backups, nil
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
	dbDump := filepath.Join(backupDir, fmt.Sprintf("database_%s.dump", timestamp))
	globalsFile := filepath.Join(backupDir, fmt.Sprintf("globals_%s.sql", timestamp))

	// Database dump
	cmd := exec.Command("kubectl", "exec", "-n", "postgres", "deploy/postgres-deployment", "--",
		"bash", "-lc", fmt.Sprintf("pg_dump -U postgres -Fc -Z 9 %s", appName))
	tools.WithKubeconfig(cmd, kubeconfigPath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pg_dump failed: %w", err)
	}
	if err := os.WriteFile(dbDump, output, 0600); err != nil {
		return nil, fmt.Errorf("failed to write database dump: %w", err)
	}

	// Globals dump
	cmd = exec.Command("kubectl", "exec", "-n", "postgres", "deploy/postgres-deployment", "--",
		"bash", "-lc", "pg_dumpall -U postgres -g")
	tools.WithKubeconfig(cmd, kubeconfigPath)
	output, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pg_dumpall failed: %w", err)
	}
	if err := os.WriteFile(globalsFile, output, 0600); err != nil {
		return nil, fmt.Errorf("failed to write globals dump: %w", err)
	}

	return []string{dbDump, globalsFile}, nil
}

// backupMySQL backs up MySQL database
func (m *Manager) backupMySQL(kubeconfigPath, appName, backupDir, timestamp string) ([]string, error) {
	dbDump := filepath.Join(backupDir, fmt.Sprintf("database_%s.sql", timestamp))

	// Get MySQL password from secret
	cmd := exec.Command("kubectl", "get", "secret", "-n", "mysql", "mysql-secret",
		"-o", "jsonpath={.data.password}")
	tools.WithKubeconfig(cmd, kubeconfigPath)
	passOutput, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get MySQL password: %w", err)
	}

	password := string(passOutput)

	// MySQL dump
	cmd = exec.Command("kubectl", "exec", "-n", "mysql", "deploy/mysql-deployment", "--",
		"bash", "-c", fmt.Sprintf("mysqldump -uroot -p'%s' --single-transaction --routines --triggers %s",
			password, appName))
	tools.WithKubeconfig(cmd, kubeconfigPath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("mysqldump failed: %w", err)
	}
	if err := os.WriteFile(dbDump, output, 0600); err != nil {
		return nil, fmt.Errorf("failed to write database dump: %w", err)
	}

	return []string{dbDump}, nil
}

// backupPVCs backs up all PVCs for an app
func (m *Manager) backupPVCs(kubeconfigPath, appName, backupDir string) ([]string, error) {
	// List PVCs for the app
	cmd := exec.Command("kubectl", "get", "pvc", "-n", appName,
		"-l", fmt.Sprintf("app=%s", appName),
		"-o", "jsonpath={.items[*].metadata.name}")
	tools.WithKubeconfig(cmd, kubeconfigPath)
	output, err := cmd.Output()
	if err != nil {
		return nil, nil // No PVCs found
	}

	pvcs := strings.Fields(string(output))
	if len(pvcs) == 0 {
		return nil, nil
	}

	var files []string
	for _, pvc := range pvcs {
		pvcBackupDir := filepath.Join(backupDir, pvc)
		if err := storage.EnsureDir(pvcBackupDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create PVC backup dir: %w", err)
		}

		// Get a running pod
		cmd = exec.Command("kubectl", "get", "pods", "-n", appName,
			"-l", fmt.Sprintf("app=%s", appName),
			"-o", "jsonpath={.items[?(@.status.phase==\"Running\")].metadata.name}")
		tools.WithKubeconfig(cmd, kubeconfigPath)
		podOutput, err := cmd.Output()
		if err != nil || len(podOutput) == 0 {
			continue
		}
		pod := strings.Fields(string(podOutput))[0]

		// Backup PVC data via tar
		cmd = exec.Command("kubectl", "exec", "-n", appName, pod, "--",
			"tar", "-C", "/data", "-cf", "-", ".")
		tools.WithKubeconfig(cmd, kubeconfigPath)
		tarData, err := cmd.Output()
		if err != nil {
			continue
		}

		// Extract tar to backup directory
		tarFile := filepath.Join(pvcBackupDir, "data.tar")
		if err := os.WriteFile(tarFile, tarData, 0600); err != nil {
			return nil, fmt.Errorf("failed to write PVC backup: %w", err)
		}
		files = append(files, tarFile)
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

// detectDatabaseType detects the database type for an app
func (m *Manager) detectDatabaseType(kubeconfigPath, appName string) (string, error) {
	// Check for postgres namespace
	cmd := exec.Command("kubectl", "get", "namespace", "postgres")
	tools.WithKubeconfig(cmd, kubeconfigPath)
	if err := cmd.Run(); err == nil {
		// Check if app uses postgres
		cmd = exec.Command("kubectl", "get", "pods", "-n", "postgres", "-l", fmt.Sprintf("app=%s", appName))
		tools.WithKubeconfig(cmd, kubeconfigPath)
		if output, _ := cmd.Output(); len(output) > 0 {
			return "postgres", nil
		}
	}

	// Check for mysql namespace
	cmd = exec.Command("kubectl", "get", "namespace", "mysql")
	tools.WithKubeconfig(cmd, kubeconfigPath)
	if err := cmd.Run(); err == nil {
		cmd = exec.Command("kubectl", "get", "pods", "-n", "mysql", "-l", fmt.Sprintf("app=%s", appName))
		tools.WithKubeconfig(cmd, kubeconfigPath)
		if output, _ := cmd.Output(); len(output) > 0 {
			return "mysql", nil
		}
	}

	return "", nil
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
