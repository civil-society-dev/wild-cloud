// Package types provides shared types for the backup system
package types

import (
	"io"
	"time"

	"github.com/wild-cloud/wild-central/daemon/internal/apps"
)

// Strategy defines the interface for backup strategies
type Strategy interface {
	// Name returns the strategy identifier
	Name() string

	// Backup creates a backup of components handled by this strategy
	Backup(instanceName, appName string, manifest *apps.AppManifest, dest BackupDestination) (*ComponentBackup, error)

	// Restore restores components from a backup
	Restore(backup *ComponentBackup, dest BackupDestination) error

	// Verify checks if a backup component can be restored
	Verify(backup *ComponentBackup, dest BackupDestination) error
}

// BackupDestination defines the interface for backup storage destinations
type BackupDestination interface {
	// Put uploads data to the destination, returns size written
	Put(key string, reader io.Reader) (int64, error)

	// Get retrieves data from the destination
	Get(key string) (io.ReadCloser, error)

	// Delete removes data from the destination
	Delete(key string) error

	// List returns objects with the given prefix
	List(prefix string) ([]BackupObject, error)

	// GetURL returns a pre-signed URL for direct access (optional)
	GetURL(key string, expiry time.Duration) (string, error)

	// Type returns the destination type identifier
	Type() string
}

// BackupObject represents an object in backup storage
type BackupObject struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"lastModified"`
}

// BackupInfo represents metadata about a backup
type BackupInfo struct {
	AppName    string            `json:"app_name"`
	Timestamp  string            `json:"timestamp"`
	Type       string            `json:"type"` // "full"
	Size       int64             `json:"size,omitempty"`
	Status     string            `json:"status"` // "completed", "failed", "in_progress"
	Error      string            `json:"error,omitempty"`
	Components []ComponentBackup `json:"components"`
	CreatedAt  time.Time         `json:"created_at"`
	Verified   bool              `json:"verified"`
	VerifiedAt *time.Time        `json:"verified_at,omitempty"`
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
	SkipData   bool     `json:"skip_data"`            // Skip data, restore only config
	BlueGreen  bool     `json:"blue_green"`           // Use blue-green restore strategy
}

// VerificationResult represents the result of backup verification
type VerificationResult struct {
	Success    bool                    `json:"success"`
	Duration   float64                 `json:"duration"` // seconds
	TestedAt   time.Time               `json:"testedAt"`
	Components []ComponentVerification `json:"components"`
}

// ComponentVerification represents verification result for a single component
type ComponentVerification struct {
	Type    string `json:"type"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// ProgressCallback is a function type for reporting backup/restore progress
type ProgressCallback func(progress int, message string)

// BackupConfiguration represents instance-level backup configuration
type BackupConfiguration struct {
	Destination  DestinationConfig `yaml:"destination"`
	Retention    RetentionPolicy   `yaml:"retention"`
	Schedules    map[string]string `yaml:"schedules"` // app-name -> cron expression
	Verification VerificationConfig `yaml:"verification"`
}

// DestinationConfig configures where backups are stored
type DestinationConfig struct {
	Type  string       `yaml:"type"` // "s3", "azure", "nfs", "local"
	S3    *S3Config    `yaml:"s3,omitempty"`
	Azure *AzureConfig `yaml:"azure,omitempty"`
	NFS   *NFSConfig   `yaml:"nfs,omitempty"`
	Local *LocalConfig `yaml:"local,omitempty"`
}

// S3Config configures S3 backup destination
type S3Config struct {
	Bucket         string `yaml:"bucket"`
	Region         string `yaml:"region"`
	Endpoint       string `yaml:"endpoint,omitempty"` // For S3-compatible services
	AccessKeyID    string `yaml:"-"`                  // Loaded from secrets.yaml
	SecretAccessKey string `yaml:"-"`                 // Loaded from secrets.yaml
}

// AzureConfig configures Azure Blob Storage destination
type AzureConfig struct {
	Container      string `yaml:"container"`
	StorageAccount string `yaml:"storageAccount"`
	AccessKey      string `yaml:"-"` // Loaded from secrets.yaml
}

// NFSConfig configures NFS backup destination
type NFSConfig struct {
	Server       string `yaml:"server"`
	Path         string `yaml:"path"`
	MountPoint   string `yaml:"mountPoint,omitempty"`
	MountOptions string `yaml:"mountOptions,omitempty"`
}

// LocalConfig configures local filesystem backup destination
type LocalConfig struct {
	Path string `yaml:"path"`
}

// RetentionPolicy defines how long to keep backups
type RetentionPolicy struct {
	Daily   int `yaml:"daily"`
	Weekly  int `yaml:"weekly"`
	Monthly int `yaml:"monthly"`
	Yearly  int `yaml:"yearly"`
}

// VerificationConfig configures backup verification
type VerificationConfig struct {
	Enabled      bool   `yaml:"enabled"`
	Schedule     string `yaml:"schedule"`     // Cron expression
	RandomSample bool   `yaml:"randomSample"` // Test random backup each time
}
