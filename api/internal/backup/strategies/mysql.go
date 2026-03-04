package strategies

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/wild-cloud/wild-central/daemon/internal/apps"
	btypes "github.com/wild-cloud/wild-central/daemon/internal/backup/types"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
)

// MySQLStrategy implements backup strategy for MySQL databases
type MySQLStrategy struct {
	dataDir string
}

// NewMySQLStrategy creates a new MySQL backup strategy
func NewMySQLStrategy(dataDir string) *MySQLStrategy {
	return &MySQLStrategy{
		dataDir: dataDir,
	}
}

// Name returns the strategy identifier
func (m *MySQLStrategy) Name() string {
	return "mysql"
}

// Backup creates a MySQL database backup using direct streaming with compression
func (m *MySQLStrategy) Backup(instanceName, appName string, manifest *apps.AppManifest, dest btypes.BackupDestination) (*btypes.ComponentBackup, error) {
	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)

	// Determine database name from manifest or default to app name
	dbName := m.getDatabaseName(instanceName, appName)

	// Get MySQL root password from secret
	password, err := m.getMySQLPassword(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get MySQL password: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	key := fmt.Sprintf("mysql/%s/%s/%s.sql.gz", instanceName, appName, timestamp)

	// Create mysqldump command that streams to stdout
	cmd := exec.Command("kubectl", "exec", "-n", "mysql",
		"mysql-0", "--", "bash", "-c",
		fmt.Sprintf("mysqldump -uroot -p'%s' --single-transaction --routines --triggers --events --databases %s",
			password, dbName))
	tools.WithKubeconfig(cmd, kubeconfigPath)

	// Use io.Pipe to stream and compress on the fly
	reader, writer := io.Pipe()

	// Create gzip writer to compress the stream
	gzWriter := gzip.NewWriter(writer)

	// Capture stderr for error messages
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Start mysqldump and compression in goroutine
	errChan := make(chan error, 1)
	go func() {
		defer writer.Close()
		defer gzWriter.Close()

		// Get command output
		output, err := cmd.Output()
		if err != nil {
			errChan <- fmt.Errorf("mysqldump failed: %v, stderr: %s", err, stderr.String())
			return
		}

		// Write compressed data
		if _, err := gzWriter.Write(output); err != nil {
			errChan <- fmt.Errorf("compression failed: %w", err)
			return
		}

		errChan <- nil
	}()

	// Stream compressed data to destination
	size, err := dest.Put(key, reader)
	if err != nil {
		cmd.Process.Kill() // Stop mysqldump if upload fails
		return nil, fmt.Errorf("failed to upload backup: %w", err)
	}

	// Wait for mysqldump and compression to complete
	if dumpErr := <-errChan; dumpErr != nil {
		return nil, dumpErr
	}

	return &btypes.ComponentBackup{
		Type:     "mysql",
		Name:     fmt.Sprintf("mysql.%s", dbName),
		Size:     size,
		Location: key,
		Metadata: map[string]interface{}{
			"database":   dbName,
			"format":     "sql",
			"compressed": true,
		},
	}, nil
}

// Restore restores a MySQL database from backup
func (m *MySQLStrategy) Restore(component *btypes.ComponentBackup, dest btypes.BackupDestination) error {
	// Get instance name from component location
	// Format: mysql/{instance}/{app}/{timestamp}.sql.gz
	parts := strings.Split(component.Location, "/")
	if len(parts) < 3 {
		return fmt.Errorf("invalid backup location format")
	}
	instanceName := parts[1]

	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)
	dbName, _ := component.Metadata["database"].(string)
	if dbName == "" {
		return fmt.Errorf("database name not found in backup metadata")
	}

	// Get MySQL root password
	password, err := m.getMySQLPassword(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to get MySQL password: %w", err)
	}

	// Get backup from destination
	compressedReader, err := dest.Get(component.Location)
	if err != nil {
		return fmt.Errorf("failed to retrieve backup: %w", err)
	}
	defer compressedReader.Close()

	// Decompress if needed
	var reader io.Reader = compressedReader
	if compressed, _ := component.Metadata["compressed"].(bool); compressed {
		gzReader, err := gzip.NewReader(compressedReader)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	// Drop and recreate database
	dropCmd := exec.Command("kubectl", "exec", "-n", "mysql", "mysql-0", "--",
		"bash", "-c",
		fmt.Sprintf("mysql -uroot -p'%s' -e 'DROP DATABASE IF EXISTS %s; CREATE DATABASE %s;'",
			password, dbName, dbName))
	tools.WithKubeconfig(dropCmd, kubeconfigPath)

	if output, err := dropCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to recreate database: %w, output: %s", err, output)
	}

	// Restore database
	restoreCmd := exec.Command("kubectl", "exec", "-i", "-n", "mysql", "mysql-0", "--",
		"bash", "-c",
		fmt.Sprintf("mysql -uroot -p'%s' %s", password, dbName))
	tools.WithKubeconfig(restoreCmd, kubeconfigPath)
	restoreCmd.Stdin = reader

	var stderr bytes.Buffer
	restoreCmd.Stderr = &stderr

	if err := restoreCmd.Run(); err != nil {
		return fmt.Errorf("mysql restore failed: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

// Verify checks if a MySQL backup can be restored
func (m *MySQLStrategy) Verify(component *btypes.ComponentBackup, dest btypes.BackupDestination) error {
	// Check if backup exists in destination
	reader, err := dest.Get(component.Location)
	if err != nil {
		return fmt.Errorf("backup not found in destination: %w", err)
	}
	defer reader.Close()

	// If compressed, verify gzip header
	if compressed, _ := component.Metadata["compressed"].(bool); compressed {
		// gzip header starts with 0x1f 0x8b
		header := make([]byte, 2)
		if _, err := io.ReadFull(reader, header); err != nil {
			return fmt.Errorf("failed to read backup header: %w", err)
		}

		if header[0] != 0x1f || header[1] != 0x8b {
			return fmt.Errorf("invalid gzip format")
		}
	} else {
		// For uncompressed SQL, check for common SQL statements
		header := make([]byte, 100)
		n, _ := reader.Read(header)
		headerStr := strings.ToLower(string(header[:n]))

		if !strings.Contains(headerStr, "mysql") && !strings.Contains(headerStr, "--") &&
			!strings.Contains(headerStr, "database") {
			return fmt.Errorf("doesn't appear to be a MySQL dump")
		}
	}

	return nil
}

// Supports checks if this strategy can handle the app based on its manifest
func (m *MySQLStrategy) Supports(manifest *apps.AppManifest) bool {
	// Check if the app has MySQL/MariaDB in its dependencies
	for _, dep := range manifest.Requires {
		if dep.Name == "mysql" || dep.Name == "mariadb" {
			return true
		}
	}
	return false
}

// getMySQLPassword retrieves the MySQL root password from Kubernetes secret
func (m *MySQLStrategy) getMySQLPassword(kubeconfigPath string) (string, error) {
	cmd := exec.Command("kubectl", "get", "secret", "-n", "mysql", "mysql-secret",
		"-o", "jsonpath={.data.password}")
	tools.WithKubeconfig(cmd, kubeconfigPath)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get MySQL secret: %w", err)
	}

	// The password is base64 encoded in the secret
	decoded, err := base64.StdEncoding.DecodeString(string(output))
	if err != nil {
		return "", fmt.Errorf("failed to decode password: %w", err)
	}

	return string(decoded), nil
}

// getDatabaseName determines the database name for the app
func (m *MySQLStrategy) getDatabaseName(instanceName, appName string) string {
	// In production, this would read from config.yaml
	// For now, use app name as database name
	return appName
}