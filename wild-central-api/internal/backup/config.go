package backup

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type BackupConfig struct {
	Repository string                 `yaml:"repository"`
	Staging    string                 `yaml:"staging"`
	Retention  ResticRetentionPolicy  `yaml:"retention"`
	Backend    BackendConfig          `yaml:"backend"`
}

type ResticRetentionPolicy struct {
	KeepDaily   int `yaml:"keepDaily"`
	KeepWeekly  int `yaml:"keepWeekly"`
	KeepMonthly int `yaml:"keepMonthly"`
	KeepYearly  int `yaml:"keepYearly"`
}

type BackendConfig struct {
	Endpoint string `yaml:"endpoint"`
	Region   string `yaml:"region"`
}

type BackupSecrets struct {
	Password    string                `yaml:"password"`
	Credentials BackupCredentials     `yaml:"credentials"`
}

type BackupCredentials struct {
	S3    *S3Credentials    `yaml:"s3,omitempty"`
	SFTP  *SFTPCredentials  `yaml:"sftp,omitempty"`
	Azure *AzureCredentials `yaml:"azure,omitempty"`
	GCS   *GCSCredentials   `yaml:"gcs,omitempty"`
}

type S3Credentials struct {
	AccessKeyID     string `yaml:"accessKeyId"`
	SecretAccessKey string `yaml:"secretAccessKey"`
}

type SFTPCredentials struct {
	Username string `yaml:"username"`
	Password string `yaml:"password,omitempty"`
	KeyFile  string `yaml:"keyFile,omitempty"`
}

type AzureCredentials struct {
	AccountName string `yaml:"accountName"`
	AccountKey  string `yaml:"accountKey"`
}

type GCSCredentials struct {
	ProjectID      string `yaml:"projectId"`
	CredentialsJSON string `yaml:"credentialsJson"`
}

func LoadBackupConfig(dataDir string) (*BackupConfig, error) {
	configPath := filepath.Join(dataDir, "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var root struct {
		Cloud struct {
			Backup BackupConfig `yaml:"backup"`
		} `yaml:"cloud"`
	}

	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	config := &root.Cloud.Backup

	if config.Repository == "" {
		return nil, nil
	}

	if config.Staging == "" {
		config.Staging = filepath.Join(dataDir, "backup-staging")
	}

	if config.Retention.KeepDaily == 0 {
		config.Retention.KeepDaily = 7
	}
	if config.Retention.KeepWeekly == 0 {
		config.Retention.KeepWeekly = 4
	}
	if config.Retention.KeepMonthly == 0 {
		config.Retention.KeepMonthly = 6
	}
	if config.Retention.KeepYearly == 0 {
		config.Retention.KeepYearly = 2
	}

	return config, nil
}

func LoadBackupSecrets(dataDir string) (*BackupSecrets, error) {
	secretsPath := filepath.Join(dataDir, "secrets.yaml")

	data, err := os.ReadFile(secretsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read secrets: %w", err)
	}

	var root struct {
		Cloud struct {
			Backup BackupSecrets `yaml:"backup"`
		} `yaml:"cloud"`
	}

	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("failed to parse secrets: %w", err)
	}

	secrets := &root.Cloud.Backup

	if secrets.Password == "" {
		return nil, nil
	}

	return secrets, nil
}

func SaveBackupConfig(dataDir string, config *BackupConfig) error {
	configPath := filepath.Join(dataDir, "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	var root map[string]interface{}
	if err := yaml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if root["cloud"] == nil {
		root["cloud"] = make(map[string]interface{})
	}
	cloud := root["cloud"].(map[string]interface{})

	cloud["backup"] = map[string]interface{}{
		"repository": config.Repository,
		"staging":    config.Staging,
		"retention": map[string]interface{}{
			"keepDaily":   config.Retention.KeepDaily,
			"keepWeekly":  config.Retention.KeepWeekly,
			"keepMonthly": config.Retention.KeepMonthly,
			"keepYearly":  config.Retention.KeepYearly,
		},
		"backend": map[string]interface{}{
			"endpoint": config.Backend.Endpoint,
			"region":   config.Backend.Region,
		},
	}

	output, err := yaml.Marshal(root)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, output, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func SaveBackupSecrets(dataDir string, secrets *BackupSecrets) error {
	secretsPath := filepath.Join(dataDir, "secrets.yaml")

	var root map[string]interface{}

	data, err := os.ReadFile(secretsPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read secrets: %w", err)
	}

	if err == nil {
		if err := yaml.Unmarshal(data, &root); err != nil {
			return fmt.Errorf("failed to parse secrets: %w", err)
		}
	} else {
		root = make(map[string]interface{})
	}

	if root["cloud"] == nil {
		root["cloud"] = make(map[string]interface{})
	}
	cloud := root["cloud"].(map[string]interface{})

	backupSecrets := map[string]interface{}{
		"password": secrets.Password,
	}

	if secrets.Credentials.S3 != nil || secrets.Credentials.SFTP != nil ||
	   secrets.Credentials.Azure != nil || secrets.Credentials.GCS != nil {
		credentials := make(map[string]interface{})

		if secrets.Credentials.S3 != nil {
			credentials["s3"] = map[string]interface{}{
				"accessKeyId":     secrets.Credentials.S3.AccessKeyID,
				"secretAccessKey": secrets.Credentials.S3.SecretAccessKey,
			}
		}
		if secrets.Credentials.SFTP != nil {
			sftp := map[string]interface{}{
				"username": secrets.Credentials.SFTP.Username,
			}
			if secrets.Credentials.SFTP.Password != "" {
				sftp["password"] = secrets.Credentials.SFTP.Password
			}
			if secrets.Credentials.SFTP.KeyFile != "" {
				sftp["keyFile"] = secrets.Credentials.SFTP.KeyFile
			}
			credentials["sftp"] = sftp
		}
		if secrets.Credentials.Azure != nil {
			credentials["azure"] = map[string]interface{}{
				"accountName": secrets.Credentials.Azure.AccountName,
				"accountKey":  secrets.Credentials.Azure.AccountKey,
			}
		}
		if secrets.Credentials.GCS != nil {
			credentials["gcs"] = map[string]interface{}{
				"projectId":       secrets.Credentials.GCS.ProjectID,
				"credentialsJson": secrets.Credentials.GCS.CredentialsJSON,
			}
		}

		backupSecrets["credentials"] = credentials
	}

	cloud["backup"] = backupSecrets

	output, err := yaml.Marshal(root)
	if err != nil {
		return fmt.Errorf("failed to marshal secrets: %w", err)
	}

	if err := os.WriteFile(secretsPath, output, 0600); err != nil {
		return fmt.Errorf("failed to write secrets: %w", err)
	}

	return nil
}
