package backup

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wild-cloud/wild-central/daemon/internal/tools"
	"gopkg.in/yaml.v3"
)

// LoadInstanceBackupConfig loads backup configuration from instance config.yaml
func LoadInstanceBackupConfig(dataDir, instanceName string) (*BackupConfiguration, error) {
	configPath := tools.GetInstanceConfigPath(dataDir, instanceName)

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read instance config: %w", err)
	}

	var root struct {
		Backup *BackupConfiguration `yaml:"backup"`
	}

	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if root.Backup == nil {
		// No backup configuration, use defaults
		return &BackupConfiguration{
			Destination: DestinationConfig{
				Type: "local",
				Local: &LocalConfig{
					Path: filepath.Join(dataDir, "instances", instanceName, "backups"),
				},
			},
			Retention: RetentionPolicy{
				Daily:   7,
				Weekly:  4,
				Monthly: 6,
				Yearly:  1,
			},
			Verification: VerificationConfig{
				Enabled:      false,
				Schedule:     "@weekly",
				RandomSample: true,
			},
		}, nil
	}

	config := root.Backup

	// Apply defaults for retention if not specified
	if config.Retention.Daily == 0 {
		config.Retention.Daily = 7
	}
	if config.Retention.Weekly == 0 {
		config.Retention.Weekly = 4
	}
	if config.Retention.Monthly == 0 {
		config.Retention.Monthly = 6
	}
	if config.Retention.Yearly == 0 {
		config.Retention.Yearly = 1
	}

	// Load credentials from secrets.yaml if needed
	if err := loadBackupSecrets(dataDir, instanceName, config); err != nil {
		// Secrets are optional, log but don't fail
		fmt.Printf("Warning: failed to load backup secrets: %v\n", err)
	}

	return config, nil
}

// loadBackupSecrets loads backup credentials from instance secrets.yaml
func loadBackupSecrets(dataDir, instanceName string, config *BackupConfiguration) error {
	secretsPath := filepath.Join(dataDir, "instances", instanceName, "secrets.yaml")

	data, err := os.ReadFile(secretsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Secrets file is optional
		}
		return fmt.Errorf("failed to read secrets: %w", err)
	}

	var root struct {
		Backup struct {
			S3 struct {
				AccessKeyID     string `yaml:"accessKeyId"`
				SecretAccessKey string `yaml:"secretAccessKey"`
			} `yaml:"s3"`
			Azure struct {
				AccessKey string `yaml:"accessKey"`
			} `yaml:"azure"`
		} `yaml:"backup"`
	}

	if err := yaml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("failed to parse secrets: %w", err)
	}

	// Apply S3 credentials if present
	if config.Destination.S3 != nil && root.Backup.S3.AccessKeyID != "" {
		config.Destination.S3.AccessKeyID = root.Backup.S3.AccessKeyID
		config.Destination.S3.SecretAccessKey = root.Backup.S3.SecretAccessKey
	}

	// Apply Azure credentials if present
	if config.Destination.Azure != nil && root.Backup.Azure.AccessKey != "" {
		config.Destination.Azure.AccessKey = root.Backup.Azure.AccessKey
	}

	return nil
}