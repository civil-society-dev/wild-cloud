package strategies

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wild-cloud/wild-central/daemon/internal/apps"
	btypes "github.com/wild-cloud/wild-central/daemon/internal/backup/types"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
	"gopkg.in/yaml.v3"
)

// ConfigStrategy implements backup strategy for app configuration files
type ConfigStrategy struct {
	dataDir string
}

// NewConfigStrategy creates a new config backup strategy
func NewConfigStrategy(dataDir string) *ConfigStrategy {
	return &ConfigStrategy{
		dataDir: dataDir,
	}
}

// Name returns the strategy identifier
func (c *ConfigStrategy) Name() string {
	return "config"
}

// Backup creates a backup of app configuration files
func (c *ConfigStrategy) Backup(instanceName, appName string, manifest *apps.AppManifest, dest btypes.BackupDestination) (*btypes.ComponentBackup, error) {
	instancePath := filepath.Join(c.dataDir, "instances", instanceName)
	appPath := filepath.Join(instancePath, "apps", appName)

	timestamp := time.Now().Format("20060102-150405")
	key := fmt.Sprintf("config/%s/%s/%s.tar.gz", instanceName, appName, timestamp)

	// Create tar.gz archive in memory
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	// Files to backup from app directory
	patterns := []string{
		"manifest.yaml",
		"kustomization.yaml",
		"*.yaml",
	}

	totalSize := int64(0)
	fileCount := 0

	// Add app files to archive
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(filepath.Join(appPath, pattern))
		for _, file := range matches {
			if err := c.addFileToTar(tarWriter, file, instancePath); err != nil {
				tarWriter.Close()
				gzWriter.Close()
				return nil, fmt.Errorf("failed to add file %s to archive: %w", file, err)
			}

			info, _ := os.Stat(file)
			if info != nil {
				totalSize += info.Size()
			}
			fileCount++
		}
	}

	// Add app configuration from config.yaml
	if err := c.addAppConfig(tarWriter, instanceName, appName); err != nil {
		tarWriter.Close()
		gzWriter.Close()
		return nil, fmt.Errorf("failed to add app config: %w", err)
	}
	fileCount++

	// Add app secrets from secrets.yaml (optional, might not exist)
	if err := c.addAppSecrets(tarWriter, instanceName, appName); err == nil {
		fileCount++
	}

	// Close the archive
	if err := tarWriter.Close(); err != nil {
		gzWriter.Close()
		return nil, fmt.Errorf("failed to close tar: %w", err)
	}

	if err := gzWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip: %w", err)
	}

	// Upload to destination
	reader := bytes.NewReader(buf.Bytes())
	size, err := dest.Put(key, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to upload config backup: %w", err)
	}

	return &btypes.ComponentBackup{
		Type:     "config",
		Name:     fmt.Sprintf("config.%s", appName),
		Size:     size,
		Location: key,
		Metadata: map[string]interface{}{
			"files":     fileCount,
			"format":    "tar.gz",
			"totalSize": totalSize,
		},
	}, nil
}

// Restore restores app configuration from backup
func (c *ConfigStrategy) Restore(component *btypes.ComponentBackup, dest btypes.BackupDestination) error {
	// Get instance and app name from component location
	// Format: config/{instance}/{app}/{timestamp}.tar.gz
	parts := strings.Split(component.Location, "/")
	if len(parts) < 3 {
		return fmt.Errorf("invalid backup location format")
	}
	instanceName := parts[1]
	appName := parts[2]

	instancePath := filepath.Join(c.dataDir, "instances", instanceName)
	appPath := filepath.Join(instancePath, "apps", appName)

	// Get backup from destination
	reader, err := dest.Get(component.Location)
	if err != nil {
		return fmt.Errorf("failed to retrieve config backup: %w", err)
	}
	defer reader.Close()

	// Decompress and extract
	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	// Create app directory if it doesn't exist
	if err := os.MkdirAll(appPath, 0755); err != nil {
		return fmt.Errorf("failed to create app directory: %w", err)
	}

	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		// Determine target path
		targetPath := filepath.Join(instancePath, header.Name)

		// Security check: ensure path is within instance directory
		if !strings.HasPrefix(targetPath, instancePath) {
			return fmt.Errorf("invalid file path in archive: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}

		case tar.TypeReg:
			// Create directory if needed
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			// Handle special files (config.yaml and secrets.yaml)
			if strings.HasSuffix(header.Name, ".config.yaml.part") {
				// Merge with existing config.yaml
				if err := c.mergeConfig(tarReader, instancePath, appName); err != nil {
					return fmt.Errorf("failed to merge config: %w", err)
				}
			} else if strings.HasSuffix(header.Name, ".secrets.yaml.part") {
				// Merge with existing secrets.yaml
				if err := c.mergeSecrets(tarReader, instancePath, appName); err != nil {
					return fmt.Errorf("failed to merge secrets: %w", err)
				}
			} else {
				// Regular file
				file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
				if err != nil {
					return fmt.Errorf("failed to create file: %w", err)
				}

				if _, err := io.Copy(file, tarReader); err != nil {
					file.Close()
					return fmt.Errorf("failed to extract file: %w", err)
				}
				file.Close()
			}
		}
	}

	return nil
}

// Verify checks if a config backup exists and is valid
func (c *ConfigStrategy) Verify(component *btypes.ComponentBackup, dest btypes.BackupDestination) error {
	// Check if backup exists
	reader, err := dest.Get(component.Location)
	if err != nil {
		return fmt.Errorf("backup not found in destination: %w", err)
	}
	defer reader.Close()

	// Verify it's a valid gzip file
	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("invalid gzip format: %w", err)
	}
	defer gzReader.Close()

	// Verify it's a valid tar archive
	tarReader := tar.NewReader(gzReader)
	hasManifest := false

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("invalid tar format: %w", err)
		}

		// Check for essential file
		if strings.Contains(header.Name, "manifest.yaml") {
			hasManifest = true
		}
	}

	if !hasManifest {
		return fmt.Errorf("backup missing essential file: manifest.yaml")
	}

	return nil
}

// addFileToTar adds a file to the tar archive
func (c *ConfigStrategy) addFileToTar(tw *tar.Writer, filePath, baseDir string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	// Create tar header
	header, err := tar.FileInfoHeader(stat, "")
	if err != nil {
		return err
	}

	// Use relative path in archive
	relPath, err := filepath.Rel(baseDir, filePath)
	if err != nil {
		return err
	}
	header.Name = relPath

	// Write header
	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	// Copy file content
	_, err = io.Copy(tw, file)
	return err
}

// addAppConfig extracts and adds app configuration from config.yaml
func (c *ConfigStrategy) addAppConfig(tw *tar.Writer, instanceName, appName string) error {
	configPath := tools.GetInstanceConfigPath(c.dataDir, instanceName)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config.yaml: %w", err)
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config.yaml: %w", err)
	}

	// Extract app-specific configuration
	appConfig := make(map[string]interface{})
	if apps, ok := config["apps"].(map[string]interface{}); ok {
		if ac, ok := apps[appName].(map[string]interface{}); ok {
			appConfig = ac
		}
	}

	// Marshal app config
	configData, err := yaml.Marshal(appConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal app config: %w", err)
	}

	// Add to archive
	header := &tar.Header{
		Name:    fmt.Sprintf("apps/%s/.config.yaml.part", appName),
		Mode:    0644,
		Size:    int64(len(configData)),
		ModTime: time.Now(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	_, err = tw.Write(configData)
	return err
}

// addAppSecrets extracts and adds app secrets from secrets.yaml
func (c *ConfigStrategy) addAppSecrets(tw *tar.Writer, instanceName, appName string) error {
	secretsPath := filepath.Join(c.dataDir, "instances", instanceName, "secrets.yaml")
	data, err := os.ReadFile(secretsPath)
	if err != nil {
		// Secrets file might not exist
		return err
	}

	var secrets map[string]interface{}
	if err := yaml.Unmarshal(data, &secrets); err != nil {
		return fmt.Errorf("failed to parse secrets.yaml: %w", err)
	}

	// Extract app-specific secrets
	appSecrets := make(map[string]interface{})
	if apps, ok := secrets["apps"].(map[string]interface{}); ok {
		if as, ok := apps[appName].(map[string]interface{}); ok {
			appSecrets = as
		}
	}

	// Marshal app secrets
	secretsData, err := yaml.Marshal(appSecrets)
	if err != nil {
		return fmt.Errorf("failed to marshal app secrets: %w", err)
	}

	// Add to archive
	header := &tar.Header{
		Name:    fmt.Sprintf("apps/%s/.secrets.yaml.part", appName),
		Mode:    0600,
		Size:    int64(len(secretsData)),
		ModTime: time.Now(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	_, err = tw.Write(secretsData)
	return err
}

// mergeConfig merges restored config with existing config.yaml
func (c *ConfigStrategy) mergeConfig(reader io.Reader, instancePath, appName string) error {
	configPath := filepath.Join(instancePath, "config.yaml")

	// Read restored app config
	restoredData, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	var restoredConfig map[string]interface{}
	if err := yaml.Unmarshal(restoredData, &restoredConfig); err != nil {
		return err
	}

	// Read existing config
	var config map[string]interface{}
	if data, err := os.ReadFile(configPath); err == nil {
		yaml.Unmarshal(data, &config)
	}
	if config == nil {
		config = make(map[string]interface{})
	}

	// Merge app config
	if config["apps"] == nil {
		config["apps"] = make(map[string]interface{})
	}
	apps := config["apps"].(map[string]interface{})
	apps[appName] = restoredConfig

	// Write back
	output, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, output, 0644)
}

// mergeSecrets merges restored secrets with existing secrets.yaml
func (c *ConfigStrategy) mergeSecrets(reader io.Reader, instancePath, appName string) error {
	secretsPath := filepath.Join(instancePath, "secrets.yaml")

	// Read restored app secrets
	restoredData, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	var restoredSecrets map[string]interface{}
	if err := yaml.Unmarshal(restoredData, &restoredSecrets); err != nil {
		return err
	}

	// Read existing secrets
	var secrets map[string]interface{}
	if data, err := os.ReadFile(secretsPath); err == nil {
		yaml.Unmarshal(data, &secrets)
	}
	if secrets == nil {
		secrets = make(map[string]interface{})
	}

	// Merge app secrets
	if secrets["apps"] == nil {
		secrets["apps"] = make(map[string]interface{})
	}
	apps := secrets["apps"].(map[string]interface{})
	apps[appName] = restoredSecrets

	// Write back
	output, err := yaml.Marshal(secrets)
	if err != nil {
		return err
	}

	return os.WriteFile(secretsPath, output, 0600)
}
// Supports checks if this strategy can handle the app based on its manifest
func (c *ConfigStrategy) Supports(manifest *apps.AppManifest) bool {
	// Config strategy supports all apps - every app has configuration to backup
	return true
}
