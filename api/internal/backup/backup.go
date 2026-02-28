// Package backup provides backup and restore operations for apps
package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/wild-cloud/wild-central/daemon/internal/apps"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
	"gopkg.in/yaml.v3"
)

// BackupInfo represents metadata about a backup
type BackupInfo struct {
	AppName    string              `json:"app_name"`
	Timestamp  string              `json:"timestamp"`
	Type       string              `json:"type"` // "full"
	Size       int64               `json:"size,omitempty"`
	Status     string              `json:"status"` // "completed", "failed", "in_progress"
	Error      string              `json:"error,omitempty"`
	Components []ComponentBackup   `json:"components"`
	CreatedAt  time.Time           `json:"created_at"`
	Verified   bool                `json:"verified"`
	VerifiedAt *time.Time          `json:"verified_at,omitempty"`
}

// ComponentBackup represents a single backup component (db, pvc, config, etc)
type ComponentBackup struct {
	Type     string                 `json:"type"`     // "postgres", "mysql", "pvc", "config"
	Name     string                 `json:"name"`     // Component identifier
	Size     int64                  `json:"size"`
	Location string                 `json:"location"` // Path in destination
	Metadata map[string]interface{} `json:"metadata"`
}

// RestoreOptions configures restore behavior
type RestoreOptions struct {
	Components []string `json:"components,omitempty"` // Specific components to restore
	SkipData   bool     `json:"skip_data"`             // Skip data, restore only config
}

// Manager handles backup and restore operations
type Manager struct {
	dataDir     string
	appsDir     string
	strategies  map[string]Strategy
	destination BackupDestination // Will be loaded per-instance
}

// NewManager creates a new backup manager
func NewManager(dataDir string) *Manager {
	return &Manager{
		dataDir:    dataDir,
		appsDir:    os.Getenv("WILD_DIRECTORY"),
		strategies: initStrategies(dataDir),
	}
}

// initStrategies initializes all available backup strategies
func initStrategies(dataDir string) map[string]Strategy {
	return map[string]Strategy{
		"postgres": NewPostgreSQLStrategy(dataDir),
		"mysql":    NewMySQLStrategy(dataDir),
		"pvc":      NewLonghornStrategy(dataDir),
		"config":   NewConfigStrategy(dataDir),
	}
}

// GetBackupDir returns the backup directory for an instance
func (m *Manager) GetBackupDir(instanceName string) string {
	return tools.GetInstanceBackupsPath(m.dataDir, instanceName)
}

// BackupApp creates a backup of an app's data
func (m *Manager) BackupApp(instanceName, appName string) (*BackupInfo, error) {
	// Load instance config to get backup destination
	destination, err := m.loadDestination(instanceName)
	if err != nil {
		return nil, fmt.Errorf("failed to load backup destination: %w", err)
	}
	m.destination = destination

	// Load app manifest to determine what to backup
	manifest, err := m.loadAppManifest(instanceName, appName)
	if err != nil {
		return nil, fmt.Errorf("failed to load app manifest: %w", err)
	}

	timestamp := time.Now().UTC().Format("20060102T150405Z")

	info := &BackupInfo{
		AppName:    appName,
		Timestamp:  timestamp,
		Type:       "full",
		Status:     "in_progress",
		Components: []ComponentBackup{},
		CreatedAt:  time.Now(),
	}

	// Detect and execute appropriate strategies
	strategies := m.detectStrategies(manifest)

	for _, strategy := range strategies {
		component, err := strategy.Backup(instanceName, appName, manifest, m.destination)
		if err != nil {
			info.Status = "failed"
			info.Error = fmt.Sprintf("%s backup failed: %v", strategy.Name(), err)
			break
		}
		if component != nil {
			info.Components = append(info.Components, *component)
			info.Size += component.Size
		}
	}

	if info.Status != "failed" {
		info.Status = "completed"
	}

	// Save backup metadata to instance directory
	if err := m.saveBackupMeta(instanceName, appName, timestamp, info); err != nil {
		return nil, fmt.Errorf("failed to save backup metadata: %w", err)
	}

	return info, nil
}

// RestoreApp restores an app from backup
func (m *Manager) RestoreApp(instanceName, appName string, opts RestoreOptions) error {
	// Load instance config to get backup destination
	destination, err := m.loadDestination(instanceName)
	if err != nil {
		return fmt.Errorf("failed to load backup destination: %w", err)
	}
	m.destination = destination

	// Find the latest backup
	backups, err := m.ListBackups(instanceName, appName)
	if err != nil || len(backups) == 0 {
		return fmt.Errorf("no backups found for app %s", appName)
	}

	// Use the most recent backup
	latestBackup := backups[0]
	for _, backup := range backups {
		if backup.Timestamp > latestBackup.Timestamp {
			latestBackup = backup
		}
	}

	// Restore each component
	for _, component := range latestBackup.Components {
		// Skip if specific components requested and this isn't one of them
		if len(opts.Components) > 0 && !contains(opts.Components, component.Type) {
			continue
		}

		strategy, exists := m.strategies[component.Type]
		if !exists {
			continue // Skip unknown component types
		}

		if err := strategy.Restore(&component, m.destination); err != nil {
			return fmt.Errorf("failed to restore %s: %w", component.Type, err)
		}
	}

	return nil
}

// ListBackups returns all backups for an app
func (m *Manager) ListBackups(instanceName, appName string) ([]*BackupInfo, error) {
	backupDir := filepath.Join(m.GetBackupDir(instanceName), appName)

	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return []*BackupInfo{}, nil
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []*BackupInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		metaFile := filepath.Join(backupDir, entry.Name(), "metadata.json")
		if info, err := m.loadBackupMeta(metaFile); err == nil {
			backups = append(backups, info)
		}
	}

	return backups, nil
}

// DeleteAppBackup deletes a specific app backup by timestamp
func (m *Manager) DeleteAppBackup(instanceName, appName, timestamp string) error {
	backupDir := filepath.Join(m.GetBackupDir(instanceName), appName, timestamp)

	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return fmt.Errorf("backup not found: %s", timestamp)
	}

	// Load backup metadata to get component locations
	metaFile := filepath.Join(backupDir, "metadata.json")
	info, err := m.loadBackupMeta(metaFile)
	if err != nil {
		return fmt.Errorf("failed to load backup metadata: %w", err)
	}

	// Load destination
	destination, err := m.loadDestination(instanceName)
	if err != nil {
		return fmt.Errorf("failed to load backup destination: %w", err)
	}

	// Delete each component from destination
	for _, component := range info.Components {
		if err := destination.Delete(component.Location); err != nil {
			// Log but continue deleting other components
			fmt.Printf("Warning: failed to delete %s from destination: %v\n", component.Location, err)
		}
	}

	// Delete local metadata
	if err := os.RemoveAll(backupDir); err != nil {
		return fmt.Errorf("failed to delete backup metadata: %w", err)
	}

	return nil
}

// VerifyBackup verifies that a backup can be restored
func (m *Manager) VerifyBackup(instanceName, appName, timestamp string) (*VerificationResult, error) {
	// Load destination
	destination, err := m.loadDestination(instanceName)
	if err != nil {
		return nil, fmt.Errorf("failed to load backup destination: %w", err)
	}

	// Load backup metadata
	backupDir := filepath.Join(m.GetBackupDir(instanceName), appName, timestamp)
	metaFile := filepath.Join(backupDir, "metadata.json")
	info, err := m.loadBackupMeta(metaFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load backup metadata: %w", err)
	}

	result := &VerificationResult{
		Success:    true,
		TestedAt:   time.Now(),
		Components: []ComponentVerification{},
	}

	// Verify each component
	for _, component := range info.Components {
		cv := ComponentVerification{
			Type: component.Type,
		}

		strategy, exists := m.strategies[component.Type]
		if !exists {
			cv.Success = false
			cv.Error = "Strategy not found"
		} else if err := strategy.Verify(&component, destination); err != nil {
			cv.Success = false
			cv.Error = err.Error()
		} else {
			cv.Success = true
		}

		result.Components = append(result.Components, cv)
		if !cv.Success {
			result.Success = false
		}
	}

	// Update backup metadata with verification status
	info.Verified = result.Success
	now := time.Now()
	info.VerifiedAt = &now
	m.saveBackupMeta(instanceName, appName, timestamp, info)

	return result, nil
}

// detectStrategies determines which backup strategies to use based on the app manifest
func (m *Manager) detectStrategies(manifest *apps.AppManifest) []Strategy {
	var strategies []Strategy

	// Always backup config
	if configStrategy, exists := m.strategies["config"]; exists {
		strategies = append(strategies, configStrategy)
	}

	// Check dependencies for database strategies
	for _, dep := range manifest.Requires {
		switch dep.Name {
		case "postgres", "postgresql":
			if s, exists := m.strategies["postgres"]; exists {
				strategies = append(strategies, s)
			}
		case "mysql", "mariadb":
			if s, exists := m.strategies["mysql"]; exists {
				strategies = append(strategies, s)
			}
		}
	}

	// Check for PVCs (will be detected by the strategy itself)
	if pvcStrategy, exists := m.strategies["pvc"]; exists {
		strategies = append(strategies, pvcStrategy)
	}

	return strategies
}

// loadAppManifest loads the app manifest from the instance directory
func (m *Manager) loadAppManifest(instanceName, appName string) (*apps.AppManifest, error) {
	manifestPath := filepath.Join(m.dataDir, "instances", instanceName, "apps", appName, "manifest.yaml")

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest apps.AppManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &manifest, nil
}

// loadDestination loads the backup destination configuration for an instance
func (m *Manager) loadDestination(instanceName string) (BackupDestination, error) {
	config, err := LoadInstanceBackupConfig(m.dataDir, instanceName)
	if err != nil {
		return nil, fmt.Errorf("failed to load backup config: %w", err)
	}

	switch config.Destination.Type {
	case "s3":
		if config.Destination.S3 == nil {
			return nil, fmt.Errorf("S3 configuration missing")
		}
		return NewS3Destination(config.Destination.S3)

	case "azure":
		if config.Destination.Azure == nil {
			return nil, fmt.Errorf("Azure configuration missing")
		}
		return NewAzureDestination(config.Destination.Azure)

	case "nfs":
		if config.Destination.NFS == nil {
			return nil, fmt.Errorf("NFS configuration missing")
		}
		return NewNFSDestination(config.Destination.NFS)

	case "local":
		if config.Destination.Local == nil {
			// Default local path if not specified
			config.Destination.Local = &LocalConfig{
				Path: filepath.Join(m.dataDir, "instances", instanceName, "backups"),
			}
		}
		return NewLocalDestination(config.Destination.Local)

	default:
		return nil, fmt.Errorf("unknown backup destination type: %s", config.Destination.Type)
	}
}

// saveBackupMeta saves backup metadata to JSON file
func (m *Manager) saveBackupMeta(instanceName, appName, timestamp string, info *BackupInfo) error {
	backupDir := filepath.Join(m.GetBackupDir(instanceName), appName, timestamp)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	metaFile := filepath.Join(backupDir, "metadata.json")
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(metaFile, data, 0600)
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

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, s := range slice {
		if s == value {
			return true
		}
	}
	return false
}