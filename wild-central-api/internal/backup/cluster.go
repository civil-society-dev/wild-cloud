package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/wild-cloud/wild-central/daemon/internal/storage"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
)

// ClusterBackupComponents represents which components to include in backup
type ClusterBackupComponents struct {
	Etcd    bool `json:"etcd"`
	Config  bool `json:"config"`
	Secrets bool `json:"secrets"`
}

// ClusterBackupInfo extends BackupInfo for cluster-level backups
type ClusterBackupInfo struct {
	*BackupInfo
	InstanceName string                  `json:"instance_name"`
	Components   ClusterBackupComponents `json:"components"`
}

// BackupCluster creates a backup of cluster components (etcd, config, secrets)
func (m *Manager) BackupCluster(instanceName string, components ClusterBackupComponents) (*ClusterBackupInfo, error) {
	stagingDir := m.GetStagingDir(instanceName)
	if err := storage.EnsureDir(stagingDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create staging directory: %w", err)
	}

	timestamp := time.Now().UTC().Format("20060102T150405Z")
	clusterBackupDir := filepath.Join(stagingDir, "cluster", timestamp)

	if err := os.RemoveAll(clusterBackupDir); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to clean backup directory: %w", err)
	}
	if err := storage.EnsureDir(clusterBackupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	info := &ClusterBackupInfo{
		BackupInfo: &BackupInfo{
			AppName:   "cluster",
			Timestamp: timestamp,
			Type:      "cluster",
			Status:    "in_progress",
			Files:     []string{},
			CreatedAt: time.Now(),
		},
		InstanceName: instanceName,
		Components:   components,
	}

	// Save initial in_progress metadata immediately so it's visible in list operations
	metaFile := filepath.Join(clusterBackupDir, "cluster-backup.json")
	if err := m.saveClusterBackupMeta(metaFile, info); err != nil {
		return nil, fmt.Errorf("failed to save initial backup metadata: %w", err)
	}

	// Backup etcd if requested
	if components.Etcd {
		etcdFile, err := m.backupEtcd(instanceName, clusterBackupDir, timestamp, stagingDir)
		if err != nil {
			info.Status = "failed"
			info.Error = fmt.Sprintf("etcd backup failed: %v", err)
		} else if etcdFile != "" {
			info.Files = append(info.Files, etcdFile)
		}
	}

	// Backup config if requested
	if components.Config {
		configFile, err := m.backupConfig(instanceName, clusterBackupDir, stagingDir)
		if err != nil && info.Status != "failed" {
			info.Status = "failed"
			info.Error = fmt.Sprintf("config backup failed: %v", err)
		} else if configFile != "" {
			info.Files = append(info.Files, configFile)
		}
	}

	// Backup secrets if requested
	if components.Secrets {
		secretsFile, err := m.backupSecrets(instanceName, clusterBackupDir, stagingDir)
		if err != nil && info.Status != "failed" {
			info.Status = "failed"
			info.Error = fmt.Sprintf("secrets backup failed: %v", err)
		} else if secretsFile != "" {
			info.Files = append(info.Files, secretsFile)
		}
	}

	if info.Status != "failed" {
		info.Status = "completed"
	}

	// Calculate backup size
	info.Size = m.calculateBackupSize(clusterBackupDir)

	// Update metadata with final status (overwrites the in_progress version)
	if err := m.saveClusterBackupMeta(metaFile, info); err != nil {
		return nil, fmt.Errorf("failed to save final backup metadata: %w", err)
	}

	return info, nil
}

// RestoreCluster restores cluster from backup
func (m *Manager) RestoreCluster(instanceName, timestamp string, components ClusterBackupComponents) error {
	stagingDir := m.GetStagingDir(instanceName)
	backupDir := filepath.Join(stagingDir, "cluster", timestamp)

	if !storage.FileExists(backupDir) {
		return fmt.Errorf("no cluster backup found for timestamp %s", timestamp)
	}

	// Restore etcd if requested and available
	if components.Etcd {
		if err := m.restoreEtcd(instanceName, backupDir); err != nil {
			return fmt.Errorf("etcd restore failed: %w", err)
		}
	}

	// Restore config if requested and available
	if components.Config {
		if err := m.restoreConfig(instanceName, backupDir); err != nil {
			return fmt.Errorf("config restore failed: %w", err)
		}
	}

	// Restore secrets if requested and available
	if components.Secrets {
		if err := m.restoreSecrets(instanceName, backupDir); err != nil {
			return fmt.Errorf("secrets restore failed: %w", err)
		}
	}

	return nil
}

// ListClusterBackups returns all cluster backups for an instance
func (m *Manager) ListClusterBackups(instanceName string) ([]*ClusterBackupInfo, error) {
	stagingDir := m.GetStagingDir(instanceName)
	clusterBackupsDir := filepath.Join(stagingDir, "cluster")

	if !storage.FileExists(clusterBackupsDir) {
		return []*ClusterBackupInfo{}, nil
	}

	entries, err := os.ReadDir(clusterBackupsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read cluster backups directory: %w", err)
	}

	var backups []*ClusterBackupInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		metaFile := filepath.Join(clusterBackupsDir, entry.Name(), "cluster-backup.json")
		if storage.FileExists(metaFile) {
			info, err := m.loadClusterBackupMeta(metaFile)
			if err == nil {
				backups = append(backups, info)
			}
		}
	}

	return backups, nil
}

// DeleteClusterBackup deletes a specific cluster backup
func (m *Manager) DeleteClusterBackup(instanceName, timestamp string) error {
	stagingDir := m.GetStagingDir(instanceName)
	backupDir := filepath.Join(stagingDir, "cluster", timestamp)

	if !storage.FileExists(backupDir) {
		return fmt.Errorf("backup not found: %s", timestamp)
	}

	if err := os.RemoveAll(backupDir); err != nil {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	return nil
}

// ListAllBackups returns both app and cluster backups
func (m *Manager) ListAllBackups(instanceName string) (map[string]interface{}, error) {
	// Get cluster backups
	clusterBackups, err := m.ListClusterBackups(instanceName)
	if err != nil {
		return nil, fmt.Errorf("failed to list cluster backups: %w", err)
	}

	// Get app backups
	stagingDir := m.GetStagingDir(instanceName)
	appsBackupsDir := filepath.Join(stagingDir, "apps")

	appBackups := make(map[string][]*BackupInfo)
	if storage.FileExists(appsBackupsDir) {
		entries, err := os.ReadDir(appsBackupsDir)
		if err != nil {
			return nil, fmt.Errorf("failed to read app backups directory: %w", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			appName := entry.Name()
			backups, err := m.ListBackups(instanceName, appName)
			if err == nil && len(backups) > 0 {
				appBackups[appName] = backups
			}
		}
	}

	return map[string]interface{}{
		"cluster": clusterBackups,
		"apps":    appBackups,
	}, nil
}

// backupEtcd creates an etcd snapshot
func (m *Manager) backupEtcd(instanceName, backupDir, timestamp, stagingDir string) (string, error) {
	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)
	etcdFile := filepath.Join(backupDir, fmt.Sprintf("etcd_%s.snapshot", timestamp))

	// Get control plane node IP
	configPath := tools.GetInstanceConfigPath(m.dataDir, instanceName)
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read instance config: %w", err)
	}

	// Simple extraction of first control plane IP (could be improved with proper YAML parsing)
	// For now, assuming talosctl is available and configured
	cmd := exec.Command("kubectl", "get", "nodes", "-l", "node-role.kubernetes.io/control-plane",
		"-o", "jsonpath={.items[0].status.addresses[?(@.type==\"InternalIP\")].address}")
	tools.WithKubeconfig(cmd, kubeconfigPath)

	ipOutput, err := cmd.Output()
	if err != nil || len(ipOutput) == 0 {
		return "", fmt.Errorf("failed to get control plane node IP: %w", err)
	}

	nodeIP := string(ipOutput)

	// Use talosctl to create etcd snapshot
	talosConfigPath := tools.GetTalosconfigPath(m.dataDir, instanceName)
	cmd = exec.Command("talosctl", "-n", nodeIP, "--talosconfig", talosConfigPath,
		"etcd", "snapshot", etcdFile)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("etcd snapshot failed: %w, output: %s", err, string(output))
	}

	_ = configData // Used for potential future enhancements

	return m.cleanBackupPath(etcdFile, stagingDir), nil
}

// backupConfig backs up instance config.yaml
func (m *Manager) backupConfig(instanceName, backupDir, stagingDir string) (string, error) {
	configPath := tools.GetInstanceConfigPath(m.dataDir, instanceName)
	if !storage.FileExists(configPath) {
		return "", fmt.Errorf("config file not found")
	}

	configFile := filepath.Join(backupDir, "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read config: %w", err)
	}

	if err := os.WriteFile(configFile, data, 0600); err != nil {
		return "", fmt.Errorf("failed to write config backup: %w", err)
	}

	return m.cleanBackupPath(configFile, stagingDir), nil
}

// backupSecrets backs up instance secrets.yaml
func (m *Manager) backupSecrets(instanceName, backupDir, stagingDir string) (string, error) {
	secretsPath := tools.GetInstanceSecretsPath(m.dataDir, instanceName)
	if !storage.FileExists(secretsPath) {
		return "", fmt.Errorf("secrets file not found")
	}

	secretsFile := filepath.Join(backupDir, "secrets.yaml")
	data, err := os.ReadFile(secretsPath)
	if err != nil {
		return "", fmt.Errorf("failed to read secrets: %w", err)
	}

	if err := os.WriteFile(secretsFile, data, 0600); err != nil {
		return "", fmt.Errorf("failed to write secrets backup: %w", err)
	}

	return m.cleanBackupPath(secretsFile, stagingDir), nil
}

// restoreEtcd restores etcd from snapshot
func (m *Manager) restoreEtcd(instanceName, backupDir string) error {
	// Find etcd snapshot file
	matches, err := filepath.Glob(filepath.Join(backupDir, "etcd_*.snapshot"))
	if err != nil || len(matches) == 0 {
		return fmt.Errorf("no etcd snapshot found in backup")
	}

	snapshotFile := matches[0]

	// Get control plane node IP
	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)
	cmd := exec.Command("kubectl", "get", "nodes", "-l", "node-role.kubernetes.io/control-plane",
		"-o", "jsonpath={.items[0].status.addresses[?(@.type==\"InternalIP\")].address}")
	tools.WithKubeconfig(cmd, kubeconfigPath)

	ipOutput, err := cmd.Output()
	if err != nil || len(ipOutput) == 0 {
		return fmt.Errorf("failed to get control plane node IP: %w", err)
	}

	nodeIP := string(ipOutput)

	// Use talosctl to bootstrap from snapshot
	talosConfigPath := tools.GetTalosconfigPath(m.dataDir, instanceName)
	cmd = exec.Command("talosctl", "-n", nodeIP, "--talosconfig", talosConfigPath,
		"bootstrap", "--recover-from", snapshotFile)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("etcd restore failed: %w, output: %s", err, string(output))
	}

	return nil
}

// restoreConfig restores instance config.yaml
func (m *Manager) restoreConfig(instanceName, backupDir string) error {
	configBackup := filepath.Join(backupDir, "config.yaml")
	if !storage.FileExists(configBackup) {
		return fmt.Errorf("config backup not found")
	}

	configPath := tools.GetInstanceConfigPath(m.dataDir, instanceName)
	data, err := os.ReadFile(configBackup)
	if err != nil {
		return fmt.Errorf("failed to read config backup: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to restore config: %w", err)
	}

	return nil
}

// restoreSecrets restores instance secrets.yaml
func (m *Manager) restoreSecrets(instanceName, backupDir string) error {
	secretsBackup := filepath.Join(backupDir, "secrets.yaml")
	if !storage.FileExists(secretsBackup) {
		return fmt.Errorf("secrets backup not found")
	}

	secretsPath := tools.GetInstanceSecretsPath(m.dataDir, instanceName)
	data, err := os.ReadFile(secretsBackup)
	if err != nil {
		return fmt.Errorf("failed to read secrets backup: %w", err)
	}

	if err := os.WriteFile(secretsPath, data, 0600); err != nil {
		return fmt.Errorf("failed to restore secrets: %w", err)
	}

	return nil
}

// saveClusterBackupMeta saves cluster backup metadata to JSON file
func (m *Manager) saveClusterBackupMeta(path string, info *ClusterBackupInfo) error {
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// loadClusterBackupMeta loads cluster backup metadata from JSON file
func (m *Manager) loadClusterBackupMeta(path string) (*ClusterBackupInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var info ClusterBackupInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	return &info, nil
}
