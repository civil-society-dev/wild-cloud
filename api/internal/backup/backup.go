// Package backup provides backup and restore operations for apps
package backup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/wild-cloud/wild-central/daemon/internal/apps"
	"github.com/wild-cloud/wild-central/daemon/internal/backup/destinations"
	"github.com/wild-cloud/wild-central/daemon/internal/backup/strategies"
	btypes "github.com/wild-cloud/wild-central/daemon/internal/backup/types"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
	"gopkg.in/yaml.v3"
)

type BackupInfo = btypes.BackupInfo
type ComponentBackup = btypes.ComponentBackup
type RestoreOptions = btypes.RestoreOptions
type Strategy = btypes.Strategy
type BackupDestination = btypes.BackupDestination
type BackupObject = btypes.BackupObject
type VerificationResult = btypes.VerificationResult
type ComponentVerification = btypes.ComponentVerification
type ProgressCallback = btypes.ProgressCallback
type BackupConfiguration = btypes.BackupConfiguration
type DestinationConfig = btypes.DestinationConfig
type S3Config = btypes.S3Config
type AzureConfig = btypes.AzureConfig
type NFSConfig = btypes.NFSConfig
type LocalConfig = btypes.LocalConfig
type RetentionPolicy = btypes.RetentionPolicy
type VerificationConfig = btypes.VerificationConfig

// Manager handles backup and restore operations
type Manager struct {
	dataDir          string
	appsDir          string
	strategies       map[string]Strategy
	destination      BackupDestination // Will be loaded per-instance
	progressCallback ProgressCallback   // Optional callback for progress updates
}

// NewManager creates a new backup manager
func NewManager(dataDir string) *Manager {
	return &Manager{
		dataDir:    dataDir,
		appsDir:    os.Getenv("WILD_DIRECTORY"),
		strategies: initStrategies(dataDir),
	}
}

// NewManagerWithProgress creates a new backup manager with progress callback
func NewManagerWithProgress(dataDir string, progressCallback ProgressCallback) *Manager {
	return &Manager{
		dataDir:          dataDir,
		appsDir:          os.Getenv("WILD_DIRECTORY"),
		strategies:       initStrategies(dataDir),
		progressCallback: progressCallback,
	}
}

// reportProgress reports progress if a callback is set
func (m *Manager) reportProgress(progress int, message string) {
	if m.progressCallback != nil {
		m.progressCallback(progress, message)
	}
}

// initStrategies initializes all available backup strategies
func initStrategies(dataDir string) map[string]Strategy {
	strats := map[string]Strategy{
		"postgres": strategies.NewPostgreSQLStrategy(dataDir),
		"mysql":    strategies.NewMySQLStrategy(dataDir),
		"config":   strategies.NewConfigStrategy(dataDir),
	}

	longhornStrategy := strategies.NewLonghornNativeStrategy(dataDir)
	strats["pvc"] = longhornStrategy
	strats["longhorn-native"] = longhornStrategy

	return strats
}

// GetBackupDir returns the backup directory for an instance
func (m *Manager) GetBackupDir(instanceName string) string {
	return tools.GetInstanceBackupsPath(m.dataDir, instanceName)
}

// BackupApp creates a backup of an app's data
func (m *Manager) BackupApp(instanceName, appName string) (*BackupInfo, error) {
	m.reportProgress(20, "Loading backup configuration")

	// Load instance config to get backup destination
	destination, err := m.loadDestination(instanceName)
	if err != nil {
		return nil, fmt.Errorf("failed to load backup destination: %w", err)
	}
	m.destination = destination

	m.reportProgress(30, "Loading app manifest")

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

	m.reportProgress(40, fmt.Sprintf("Backing up %d components", len(strategies)))

	// Calculate progress per strategy
	progressStart := 40
	progressEnd := 90
	progressPerStrategy := (progressEnd - progressStart) / len(strategies)

	for i, strategy := range strategies {
		currentProgress := progressStart + (i * progressPerStrategy)
		m.reportProgress(currentProgress, fmt.Sprintf("Backing up %s", strategy.Name()))

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
		m.reportProgress(95, "Saving backup metadata")
	}

	// Save backup metadata to instance directory
	if err := m.saveBackupMeta(instanceName, appName, timestamp, info); err != nil {
		return nil, fmt.Errorf("failed to save backup metadata: %w", err)
	}

	m.reportProgress(100, "Backup completed")
	return info, nil
}

// RestoreApp restores an app from backup
func (m *Manager) RestoreApp(instanceName, appName string, opts RestoreOptions) error {
	// Debug logging
	fmt.Printf("RestoreApp called with opts: %+v\n", opts)

	m.reportProgress(20, "Loading backup configuration")

	// Load instance config to get backup destination
	destination, err := m.loadDestination(instanceName)
	if err != nil {
		return fmt.Errorf("failed to load backup destination: %w", err)
	}
	m.destination = destination

	m.reportProgress(30, "Finding available backups")

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

	m.reportProgress(40, fmt.Sprintf("Restoring from backup %s", latestBackup.Timestamp))

	// Calculate progress per component
	progressStart := 40
	progressEnd := 80
	componentsToRestore := 0
	for _, component := range latestBackup.Components {
		if len(opts.Components) == 0 || contains(opts.Components, component.Type) {
			componentsToRestore++
		}
	}

	progressPerComponent := 0
	if componentsToRestore > 0 {
		progressPerComponent = (progressEnd - progressStart) / componentsToRestore
	}

	restoredCount := 0

	// Restore each component with blue-green flag
	for _, component := range latestBackup.Components {
		// Skip if specific components requested and this isn't one of them
		if len(opts.Components) > 0 && !contains(opts.Components, component.Type) {
			continue
		}

		strategy, exists := m.strategies[component.Type]
		if !exists {
			continue // Skip unknown component types
		}

		currentProgress := progressStart + (restoredCount * progressPerComponent)
		m.reportProgress(currentProgress, fmt.Sprintf("Restoring %s", component.Type))

		// Create a copy of component with blue-green flag
		componentCopy := component
		if componentCopy.Metadata == nil {
			componentCopy.Metadata = make(map[string]interface{})
		}
		componentCopy.Metadata["blueGreen"] = opts.BlueGreen

		// Debug logging
		fmt.Printf("Restoring component %s with metadata: %+v\n", component.Type, componentCopy.Metadata)

		if err := strategy.Restore(&componentCopy, m.destination); err != nil {
			return fmt.Errorf("failed to restore %s: %w", component.Type, err)
		}

		restoredCount++
	}

	// If blue-green restore, deploy the app to the restore namespace
	if opts.BlueGreen {
		m.reportProgress(85, "Deploying app to restore namespace")
		fmt.Printf("Deploying app to restore namespace for blue-green restore\n")
		if err := m.deployToRestoreNamespace(instanceName, appName); err != nil {
			return fmt.Errorf("failed to deploy app to restore namespace: %w", err)
		}
	}

	m.reportProgress(100, "Restore completed")
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

	// Sort backups by timestamp (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp > backups[j].Timestamp
	})

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
		return destinations.NewS3Destination(config.Destination.S3)

	case "azure":
		if config.Destination.Azure == nil {
			return nil, fmt.Errorf("Azure configuration missing")
		}
		return destinations.NewAzureDestination(config.Destination.Azure)

	case "nfs":
		if config.Destination.NFS == nil {
			return nil, fmt.Errorf("NFS configuration missing")
		}
		return destinations.NewNFSDestination(config.Destination.NFS)

	case "local":
		if config.Destination.Local == nil {
			// Default local path if not specified
			config.Destination.Local = &LocalConfig{
				Path: filepath.Join(m.dataDir, "instances", instanceName, "backups"),
			}
		}
		return destinations.NewLocalDestination(config.Destination.Local)

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

// mergeConfigurations safely merges backup config with current config
func (m *Manager) mergeConfigurations(backupConfig, currentConfig map[string]interface{}) map[string]interface{} {
	// Start with current config as base (preserves user customizations)
	merged := make(map[string]interface{})
	for k, v := range currentConfig {
		merged[k] = v
	}

	// Overlay backup config for data-specific fields
	// These are fields that should be restored from backup
	dataFields := []string{
		"storage",      // PVC sizes
		"replicas",     // Scaling settings
		"resources",    // Resource limits
		"persistence",  // Persistence settings
	}

	for _, field := range dataFields {
		if backupValue, exists := backupConfig[field]; exists {
			merged[field] = backupValue
		}
	}

	// Handle nested app configurations
	if backupApps, ok := backupConfig["apps"].(map[string]interface{}); ok {
		if currentApps, ok := currentConfig["apps"].(map[string]interface{}); ok {
			mergedApps := make(map[string]interface{})

			// Merge each app's configuration
			for appName, currentAppConfig := range currentApps {
				if currentAppMap, ok := currentAppConfig.(map[string]interface{}); ok {
					mergedApps[appName] = currentAppMap

					// If this app exists in backup, merge data fields
					if backupAppConfig, exists := backupApps[appName]; exists {
						if backupAppMap, ok := backupAppConfig.(map[string]interface{}); ok {
							appMerged := make(map[string]interface{})
							// Start with current
							for k, v := range currentAppMap {
								appMerged[k] = v
							}
							// Overlay backup data fields
							for _, field := range dataFields {
								if backupValue, exists := backupAppMap[field]; exists {
									appMerged[field] = backupValue
								}
							}
							mergedApps[appName] = appMerged
						}
					}
				}
			}
			merged["apps"] = mergedApps
		}
	}

	return merged
}

// mergeSecrets safely merges backup secrets with current secrets
func (m *Manager) mergeSecrets(backupSecrets, currentSecrets map[string]interface{}) map[string]interface{} {
	// For secrets, we generally want to preserve current secrets
	// Only restore secrets that don't exist in current config
	merged := make(map[string]interface{})

	// Start with all current secrets
	for k, v := range currentSecrets {
		merged[k] = v
	}

	// Add any secrets from backup that don't exist in current
	// This handles cases where secrets were deleted accidentally
	for k, v := range backupSecrets {
		if _, exists := merged[k]; !exists {
			fmt.Printf("Restoring missing secret: %s\n", k)
			merged[k] = v
		}
	}

	return merged
}

// deployToRestoreNamespace deploys the app to the restore namespace for blue-green deployment
func (m *Manager) deployToRestoreNamespace(instanceName, appName string) error {
	kubeconfigPath := filepath.Join(m.dataDir, "instances", instanceName, "kubeconfig")
	restoreNamespace := appName + "-restore"

	// Source and destination paths
	srcAppDir := filepath.Join(m.dataDir, "instances", instanceName, "apps", appName)
	restoreAppDir := filepath.Join(m.dataDir, "instances", instanceName, "apps-restore", appName)

	// Create restore app directory
	if err := os.MkdirAll(restoreAppDir, 0755); err != nil {
		return fmt.Errorf("failed to create restore app directory: %w", err)
	}

	// Copy all manifest files
	if err := copyDirectory(srcAppDir, restoreAppDir); err != nil {
		return fmt.Errorf("failed to copy app manifests: %w", err)
	}

	// Update namespace in kustomization.yaml
	kustomizePath := filepath.Join(restoreAppDir, "kustomization.yaml")
	if err := updateKustomizeNamespace(kustomizePath, restoreNamespace); err != nil {
		return fmt.Errorf("failed to update kustomize namespace: %w", err)
	}

	// Update namespace.yaml
	namespacePath := filepath.Join(restoreAppDir, "namespace.yaml")
	if err := updateNamespaceManifest(namespacePath, restoreNamespace); err != nil {
		return fmt.Errorf("failed to update namespace manifest: %w", err)
	}

	// Update PVC references to use restored volumes
	if err := updatePVCReferences(restoreAppDir, restoreNamespace); err != nil {
		return fmt.Errorf("failed to update PVC references: %w", err)
	}

	// Update database references for blue-green restore
	if err := updateDatabaseReferences(restoreAppDir, appName); err != nil {
		return fmt.Errorf("failed to update database references: %w", err)
	}

	// Copy secrets from original namespace to restore namespace
	secretCmd := exec.Command("kubectl", "get", "secret", appName+"-secrets",
		"-n", appName, "-o", "yaml")
	tools.WithKubeconfig(secretCmd, kubeconfigPath)
	secretOutput, err := secretCmd.Output()
	if err == nil && len(secretOutput) > 0 {
		// Replace namespace in secret YAML
		secretYaml := string(secretOutput)
		secretYaml = strings.ReplaceAll(secretYaml, "namespace: "+appName, "namespace: "+restoreNamespace)

		// Apply to restore namespace
		applyCmd := exec.Command("kubectl", "apply", "-f", "-")
		tools.WithKubeconfig(applyCmd, kubeconfigPath)
		applyCmd.Stdin = strings.NewReader(secretYaml)
		if err := applyCmd.Run(); err != nil {
			fmt.Printf("Warning: failed to copy secrets to restore namespace: %v\n", err)
		}
	}

	// Deploy using kubectl apply -k
	deployCmd := exec.Command("kubectl", "apply", "-k", restoreAppDir)
	tools.WithKubeconfig(deployCmd, kubeconfigPath)

	var stdout, stderr bytes.Buffer
	deployCmd.Stdout = &stdout
	deployCmd.Stderr = &stderr

	if err := deployCmd.Run(); err != nil {
		return fmt.Errorf("failed to deploy app: %w, stderr: %s", err, stderr.String())
	}

	fmt.Printf("Successfully deployed app to restore namespace: %s\n", restoreNamespace)
	return nil
}

// Helper functions for deployment

func copyDirectory(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(dstPath, data, info.Mode())
	})
}

func updateKustomizeNamespace(path, namespace string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Simple replacement - in production would use proper YAML parsing
	content := string(data)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "namespace:") {
			lines[i] = "namespace: " + namespace
		}
	}

	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}

func updateNamespaceManifest(path, namespace string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)
	// Replace name in metadata
	content = regexp.MustCompile(`name:\s+\w+`).ReplaceAllString(content, "name: "+namespace)

	return os.WriteFile(path, []byte(content), 0644)
}

func updatePVCReferences(appDir, namespace string) error {
	// Update any PVC volume references in deployment files
	// This is simplified - in production would parse YAML properly
	return filepath.Walk(appDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			content := string(data)
			// Update PVC references if they exist
			if strings.Contains(content, "persistentVolumeClaim:") {
				// The PVC names should already be correct from the original manifests
				// Just ensure namespace is correct which we already did in kustomize
			}

			// Write back the file even if no changes (preserves permissions)
			os.WriteFile(path, []byte(content), info.Mode())
		}

		return nil
	})
}

func updateDatabaseReferences(appDir, appName string) error {
	// Generic function to update database references for blue-green restore
	// This works by finding common database-related patterns and adding _restore suffix
	return filepath.Walk(appDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		// Only process YAML files
		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		// Skip certain files that shouldn't be modified
		basename := filepath.Base(path)
		if basename == "namespace.yaml" || basename == "kustomization.yaml" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		content := string(data)
		modified := false

		// Common database name patterns to update
		// These patterns handle various ways apps specify database names
		patterns := []struct {
			pattern     string
			replacement string
			isRegex     bool
		}{
			// Environment variable patterns for database names (quoted values)
			{`value: "` + appName + `"`, `value: "` + appName + `_restore"`, false},
			{`value: '` + appName + `'`, `value: '` + appName + `_restore'`, false},

			// Common database environment variable names
			{`database: ` + appName, `database: ` + appName + `_restore`, false},
			{`dbName: ` + appName, `dbName: ` + appName + `_restore`, false},
			{`POSTGRES_DB: ` + appName, `POSTGRES_DB: ` + appName + `_restore`, false},
			{`MYSQL_DATABASE: ` + appName, `MYSQL_DATABASE: ` + appName + `_restore`, false},

			// Bare value pattern with word boundary (regex)
			{`value:\s+` + appName + `\b`, `value: ` + appName + `_restore`, true},

			// Database URLs (regex - be careful not to double-suffix)
			{`://[^/]+/` + appName + `(\?|$|")`, `://[^/]+/` + appName + `_restore$1`, true},
		}

		// Apply all patterns
		for _, p := range patterns {
			if strings.Contains(content, appName) && !strings.Contains(content, appName+"_restore") {
				if p.isRegex {
					// Use regexp for complex patterns
					re := regexp.MustCompile(p.pattern)
					newContent := re.ReplaceAllString(content, p.replacement)
					if newContent != content {
						content = newContent
						modified = true
					}
				} else {
					// Simple string replacement for exact matches
					newContent := strings.ReplaceAll(content, p.pattern, p.replacement)
					if newContent != content {
						content = newContent
						modified = true
					}
				}
			}
		}

		// Write back if modified
		if modified {
			if err := os.WriteFile(path, []byte(content), info.Mode()); err != nil {
				return fmt.Errorf("failed to write file %s: %w", path, err)
			}
			fmt.Printf("Updated database references in %s\n", basename)
		}

		return nil
	})
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