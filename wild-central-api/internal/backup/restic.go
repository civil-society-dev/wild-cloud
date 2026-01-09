package backup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type ResticClient struct {
	config  *BackupConfig
	secrets *BackupSecrets
	dataDir string
}

type ResticSnapshot struct {
	ID       string    `json:"id"`
	ShortID  string    `json:"short_id"`
	Time     time.Time `json:"time"`
	Hostname string    `json:"hostname"`
	Username string    `json:"username"`
	Paths    []string  `json:"paths"`
	Tags     []string  `json:"tags"`
}

type ResticStats struct {
	TotalSize      uint64 `json:"total_size"`
	TotalFileCount uint64 `json:"total_file_count"`
}

type ResticStatus struct {
	Initialized bool   `json:"initialized"`
	Snapshots   int    `json:"snapshots"`
	TotalSize   uint64 `json:"totalSize"`
	LastBackup  string `json:"lastBackup,omitempty"`
}

func NewResticClient(dataDir string) (*ResticClient, error) {
	config, err := LoadBackupConfig(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if config == nil || config.Repository == "" {
		return nil, fmt.Errorf("backup not configured")
	}

	secrets, err := LoadBackupSecrets(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load secrets: %w", err)
	}

	if secrets == nil || secrets.Password == "" {
		return nil, fmt.Errorf("backup password not configured")
	}

	return &ResticClient{
		config:  config,
		secrets: secrets,
		dataDir: dataDir,
	}, nil
}

func (r *ResticClient) detectBackend() string {
	repo := r.config.Repository

	if strings.HasPrefix(repo, "s3:") {
		return "s3"
	}
	if strings.HasPrefix(repo, "sftp:") {
		return "sftp"
	}
	if strings.HasPrefix(repo, "azure:") {
		return "azure"
	}
	if strings.HasPrefix(repo, "gs:") {
		return "gcs"
	}

	return "local"
}

func (r *ResticClient) buildEnv() []string {
	env := os.Environ()

	env = append(env, fmt.Sprintf("RESTIC_REPOSITORY=%s", r.config.Repository))
	env = append(env, fmt.Sprintf("RESTIC_PASSWORD=%s", r.secrets.Password))

	backend := r.detectBackend()

	switch backend {
	case "s3":
		if r.secrets.Credentials.S3 != nil {
			env = append(env, fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", r.secrets.Credentials.S3.AccessKeyID))
			env = append(env, fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", r.secrets.Credentials.S3.SecretAccessKey))
		}
		if r.config.Backend.Endpoint != "" {
			env = append(env, fmt.Sprintf("AWS_S3_ENDPOINT=%s", r.config.Backend.Endpoint))
		}
		if r.config.Backend.Region != "" {
			env = append(env, fmt.Sprintf("AWS_DEFAULT_REGION=%s", r.config.Backend.Region))
		}

	case "azure":
		if r.secrets.Credentials.Azure != nil {
			env = append(env, fmt.Sprintf("AZURE_ACCOUNT_NAME=%s", r.secrets.Credentials.Azure.AccountName))
			env = append(env, fmt.Sprintf("AZURE_ACCOUNT_KEY=%s", r.secrets.Credentials.Azure.AccountKey))
		}

	case "gcs":
		if r.secrets.Credentials.GCS != nil {
			env = append(env, fmt.Sprintf("GOOGLE_PROJECT_ID=%s", r.secrets.Credentials.GCS.ProjectID))
			env = append(env, fmt.Sprintf("GOOGLE_APPLICATION_CREDENTIALS=%s", r.secrets.Credentials.GCS.CredentialsJSON))
		}
	}

	return env
}

func (r *ResticClient) runCommand(args ...string) (string, error) {
	cmd := exec.Command("restic", args...)
	cmd.Env = r.buildEnv()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("restic command failed: %w\nStderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

func (r *ResticClient) Init() error {
	_, err := r.runCommand("init")
	return err
}

func (r *ResticClient) IsInitialized() bool {
	_, err := r.runCommand("snapshots", "--json")
	return err == nil
}

func (r *ResticClient) Backup(sourcePath string, tags ...string) (string, error) {
	args := []string{"backup", sourcePath, "--json"}

	for _, tag := range tags {
		args = append(args, "--tag", tag)
	}

	output, err := r.runCommand(args...)
	if err != nil {
		return "", err
	}

	var result struct {
		SnapshotID string `json:"snapshot_id"`
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	lastLine := lines[len(lines)-1]

	if err := json.Unmarshal([]byte(lastLine), &result); err != nil {
		return "", fmt.Errorf("failed to parse backup output: %w", err)
	}

	return result.SnapshotID, nil
}

func (r *ResticClient) Snapshots(tags ...string) ([]ResticSnapshot, error) {
	args := []string{"snapshots", "--json"}

	for _, tag := range tags {
		args = append(args, "--tag", tag)
	}

	output, err := r.runCommand(args...)
	if err != nil {
		return nil, err
	}

	var snapshots []ResticSnapshot
	if err := json.Unmarshal([]byte(output), &snapshots); err != nil {
		return nil, fmt.Errorf("failed to parse snapshots: %w", err)
	}

	return snapshots, nil
}

func (r *ResticClient) Stats() (*ResticStats, error) {
	output, err := r.runCommand("stats", "--json")
	if err != nil {
		return nil, err
	}

	var stats ResticStats
	if err := json.Unmarshal([]byte(output), &stats); err != nil {
		return nil, fmt.Errorf("failed to parse stats: %w", err)
	}

	return &stats, nil
}

func (r *ResticClient) Check() error {
	_, err := r.runCommand("check")
	return err
}

func (r *ResticClient) Forget(snapshotID string) error {
	_, err := r.runCommand("forget", snapshotID)
	return err
}

func (r *ResticClient) Prune() error {
	_, err := r.runCommand("prune")
	return err
}

func (r *ResticClient) ForgetWithPolicy() error {
	args := []string{"forget", "--prune"}

	if r.config.Retention.KeepDaily > 0 {
		args = append(args, "--keep-daily", fmt.Sprintf("%d", r.config.Retention.KeepDaily))
	}
	if r.config.Retention.KeepWeekly > 0 {
		args = append(args, "--keep-weekly", fmt.Sprintf("%d", r.config.Retention.KeepWeekly))
	}
	if r.config.Retention.KeepMonthly > 0 {
		args = append(args, "--keep-monthly", fmt.Sprintf("%d", r.config.Retention.KeepMonthly))
	}
	if r.config.Retention.KeepYearly > 0 {
		args = append(args, "--keep-yearly", fmt.Sprintf("%d", r.config.Retention.KeepYearly))
	}

	_, err := r.runCommand(args...)
	return err
}

func (r *ResticClient) Status() (*ResticStatus, error) {
	status := &ResticStatus{
		Initialized: r.IsInitialized(),
	}

	if !status.Initialized {
		return status, nil
	}

	snapshots, err := r.Snapshots()
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshots: %w", err)
	}

	status.Snapshots = len(snapshots)

	if len(snapshots) > 0 {
		latest := snapshots[0]
		for _, s := range snapshots {
			if s.Time.After(latest.Time) {
				latest = s
			}
		}
		status.LastBackup = latest.Time.Format(time.RFC3339)
	}

	stats, err := r.Stats()
	if err == nil {
		status.TotalSize = stats.TotalSize
	}

	return status, nil
}

func (r *ResticClient) Restore(snapshotID, targetPath string) error {
	_, err := r.runCommand("restore", snapshotID, "--target", targetPath)
	return err
}

// GetSnapshot retrieves details for a specific snapshot
func (r *ResticClient) GetSnapshot(id string) (*ResticSnapshot, error) {
	output, err := r.runCommand("snapshots", id, "--json")
	if err != nil {
		return nil, err
	}

	var snapshots []ResticSnapshot
	if err := json.Unmarshal([]byte(output), &snapshots); err != nil {
		return nil, fmt.Errorf("failed to parse snapshot: %w", err)
	}

	if len(snapshots) == 0 {
		return nil, fmt.Errorf("snapshot not found: %s", id)
	}

	return &snapshots[0], nil
}

// ListSnapshotFiles lists files in a snapshot
func (r *ResticClient) ListSnapshotFiles(id string) ([]string, error) {
	output, err := r.runCommand("ls", id, "--json")
	if err != nil {
		return nil, err
	}

	var files []string
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		var entry struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry.Path != "" {
			files = append(files, entry.Path)
		}
	}

	return files, nil
}

// RestoreSnapshot downloads a snapshot to a target path
func (r *ResticClient) RestoreSnapshot(id, targetPath string) error {
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	_, err := r.runCommand("restore", id, "--target", targetPath)
	return err
}

func UploadToRestic(dataDir, instanceName, appName, backupPath string) (string, error) {
	client, err := NewResticClient(dataDir)
	if err != nil {
		return "", err
	}

	if !client.IsInitialized() {
		if err := client.Init(); err != nil {
			return "", fmt.Errorf("failed to initialize repository: %w", err)
		}
	}

	tags := []string{
		fmt.Sprintf("instance:%s", instanceName),
		fmt.Sprintf("app:%s", appName),
	}

	snapshotID, err := client.Backup(backupPath, tags...)
	if err != nil {
		return "", fmt.Errorf("failed to backup to restic: %w", err)
	}

	if err := os.RemoveAll(backupPath); err != nil {
		return snapshotID, fmt.Errorf("backup uploaded but failed to clean staging: %w", err)
	}

	return snapshotID, nil
}

func CleanStagingDirectory(stagingDir string) error {
	entries, err := os.ReadDir(stagingDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(stagingDir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}

	return nil
}
