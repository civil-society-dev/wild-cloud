package backup

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/wild-cloud/wild-central/daemon/internal/apps"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
	"gopkg.in/yaml.v3"
)

// PostgreSQLStrategy implements backup strategy for PostgreSQL databases
type PostgreSQLStrategy struct {
	dataDir string
}

// NewPostgreSQLStrategy creates a new PostgreSQL backup strategy
func NewPostgreSQLStrategy(dataDir string) *PostgreSQLStrategy {
	return &PostgreSQLStrategy{
		dataDir: dataDir,
	}
}

// Name returns the strategy identifier
func (p *PostgreSQLStrategy) Name() string {
	return "postgres"
}

// Backup creates a PostgreSQL database backup using direct streaming
func (p *PostgreSQLStrategy) Backup(instanceName, appName string, manifest *apps.AppManifest, dest BackupDestination) (*ComponentBackup, error) {
	kubeconfigPath := tools.GetKubeconfigPath(p.dataDir, instanceName)

	// Determine database name from manifest or default to app name
	dbName := p.getDatabaseName(instanceName, appName)

	timestamp := time.Now().Format("20060102-150405")
	key := fmt.Sprintf("postgres/%s/%s/%s.dump", instanceName, appName, timestamp)

	// Get the postgres pod name
	podName, err := p.getPostgresPod(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find postgres pod: %w", err)
	}

	// Create pg_dump command that streams to stdout
	cmd := exec.Command("kubectl", "exec", "-n", "postgres",
		podName, "--", "pg_dump",
		"-U", "postgres",
		"--format=custom",
		"--no-owner",
		"--compress=9",
		"--dbname", dbName,
	)
	tools.WithKubeconfig(cmd, kubeconfigPath)

	// Use io.Pipe to stream directly from pg_dump to destination
	reader, writer := io.Pipe()

	// Capture stderr for error messages
	var stderr bytes.Buffer
	cmd.Stdout = writer
	cmd.Stderr = &stderr

	// Start pg_dump in goroutine
	errChan := make(chan error, 1)
	go func() {
		defer writer.Close()
		err := cmd.Run()
		if err != nil {
			errChan <- fmt.Errorf("pg_dump failed: %v, stderr: %s", err, stderr.String())
		} else {
			errChan <- nil
		}
	}()

	// Stream to destination
	size, err := dest.Put(key, reader)
	if err != nil {
		cmd.Process.Kill() // Stop pg_dump if upload fails
		return nil, fmt.Errorf("failed to upload backup: %w", err)
	}

	// Wait for pg_dump to complete
	if pgErr := <-errChan; pgErr != nil {
		return nil, pgErr
	}

	// Also backup globals (users, roles, etc)
	globalsKey := fmt.Sprintf("postgres/%s/%s/%s-globals.sql", instanceName, appName, timestamp)
	if err := p.backupGlobals(kubeconfigPath, dest, globalsKey); err != nil {
		// Globals backup is optional, log but don't fail
		fmt.Printf("Warning: failed to backup PostgreSQL globals: %v\n", err)
	}

	return &ComponentBackup{
		Type:     "postgres",
		Name:     fmt.Sprintf("postgres.%s", dbName),
		Size:     size,
		Location: key,
		Metadata: map[string]interface{}{
			"database": dbName,
			"format":   "custom",
			"globals":  globalsKey,
		},
	}, nil
}

// Restore restores a PostgreSQL database from backup
func (p *PostgreSQLStrategy) Restore(component *ComponentBackup, dest BackupDestination) error {
	// Get instance and app name from component location
	// Format: postgres/{instance}/{app}/{timestamp}.dump
	parts := strings.Split(component.Location, "/")
	if len(parts) < 3 {
		return fmt.Errorf("invalid backup location format")
	}
	instanceName := parts[1]

	kubeconfigPath := tools.GetKubeconfigPath(p.dataDir, instanceName)
	dbName, _ := component.Metadata["database"].(string)
	if dbName == "" {
		return fmt.Errorf("database name not found in backup metadata")
	}

	// Get the postgres pod name
	podName, err := p.getPostgresPod(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to find postgres pod: %w", err)
	}

	// Get backup from destination
	reader, err := dest.Get(component.Location)
	if err != nil {
		return fmt.Errorf("failed to retrieve backup: %w", err)
	}
	defer reader.Close()

	// Drop database first (must be done separately, can't run in transaction)
	dropCmd := exec.Command("kubectl", "exec", "-n", "postgres", podName, "--",
		"psql", "-U", "postgres", "-d", "postgres", "-c",
		fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
	tools.WithKubeconfig(dropCmd, kubeconfigPath)

	if output, err := dropCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to drop database: %w, output: %s", err, output)
	}

	// Create new database
	createCmd := exec.Command("kubectl", "exec", "-n", "postgres", podName, "--",
		"psql", "-U", "postgres", "-d", "postgres", "-c",
		fmt.Sprintf("CREATE DATABASE %s", dbName))
	tools.WithKubeconfig(createCmd, kubeconfigPath)

	if output, err := createCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create database: %w, output: %s", err, output)
	}

	// Restore database using pg_restore
	restoreCmd := exec.Command("kubectl", "exec", "-i", "-n", "postgres", podName, "--",
		"pg_restore", "-U", "postgres", "-d", dbName, "--no-owner", "--clean", "--if-exists")
	tools.WithKubeconfig(restoreCmd, kubeconfigPath)
	restoreCmd.Stdin = reader

	var stderr bytes.Buffer
	restoreCmd.Stderr = &stderr

	if err := restoreCmd.Run(); err != nil {
		// pg_restore returns non-zero for warnings, check if database was actually restored
		checkCmd := exec.Command("kubectl", "exec", "-n", "postgres", podName, "--",
			"psql", "-U", "postgres", "-d", dbName, "-c", "\\dt")
		tools.WithKubeconfig(checkCmd, kubeconfigPath)

		if checkOutput, checkErr := checkCmd.Output(); checkErr != nil || !strings.Contains(string(checkOutput), "table") {
			return fmt.Errorf("pg_restore failed: %w, stderr: %s", err, stderr.String())
		}
		// Restore succeeded with warnings, continue
	}

	// Restore globals if present
	if globalsKey, ok := component.Metadata["globals"].(string); ok && globalsKey != "" {
		if err := p.restoreGlobals(kubeconfigPath, dest, globalsKey); err != nil {
			// Globals restore is optional, log but don't fail
			fmt.Printf("Warning: failed to restore PostgreSQL globals: %v\n", err)
		}
	}

	return nil
}

// Verify checks if a PostgreSQL backup can be restored
func (p *PostgreSQLStrategy) Verify(component *ComponentBackup, dest BackupDestination) error {
	// Check if backup exists in destination
	reader, err := dest.Get(component.Location)
	if err != nil {
		return fmt.Errorf("backup not found in destination: %w", err)
	}
	reader.Close()

	// Verify it's a valid pg_dump format by checking magic bytes
	reader, err = dest.Get(component.Location)
	if err != nil {
		return err
	}
	defer reader.Close()

	// pg_dump custom format starts with "PGDMP"
	magic := make([]byte, 5)
	if _, err := io.ReadFull(reader, magic); err != nil {
		return fmt.Errorf("failed to read backup header: %w", err)
	}

	if string(magic) != "PGDMP" {
		return fmt.Errorf("invalid PostgreSQL dump format")
	}

	return nil
}

// Supports checks if this strategy can handle the app based on its manifest
func (p *PostgreSQLStrategy) Supports(manifest *apps.AppManifest) bool {
	// Check if the app has PostgreSQL in its dependencies
	for _, dep := range manifest.Requires {
		if dep.Name == "postgres" || dep.Name == "postgresql" || dep.Name == "pg" {
			return true
		}
	}
	return false
}

// backupGlobals backs up PostgreSQL global objects (users, roles, etc)
func (p *PostgreSQLStrategy) backupGlobals(kubeconfigPath string, dest BackupDestination, key string) error {
	// Get the postgres pod name
	podName, err := p.getPostgresPod(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to find postgres pod: %w", err)
	}

	cmd := exec.Command("kubectl", "exec", "-n", "postgres", podName, "--",
		"pg_dumpall", "-U", "postgres", "--globals-only")
	tools.WithKubeconfig(cmd, kubeconfigPath)

	reader, writer := io.Pipe()
	cmd.Stdout = writer

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	errChan := make(chan error, 1)
	go func() {
		defer writer.Close()
		if err := cmd.Run(); err != nil {
			errChan <- fmt.Errorf("pg_dumpall failed: %v, stderr: %s", err, stderr.String())
		} else {
			errChan <- nil
		}
	}()

	if _, err := dest.Put(key, reader); err != nil {
		cmd.Process.Kill()
		return err
	}

	return <-errChan
}

// restoreGlobals restores PostgreSQL global objects
func (p *PostgreSQLStrategy) restoreGlobals(kubeconfigPath string, dest BackupDestination, key string) error {
	// Get the postgres pod name
	podName, err := p.getPostgresPod(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to find postgres pod: %w", err)
	}

	reader, err := dest.Get(key)
	if err != nil {
		return fmt.Errorf("failed to retrieve globals backup: %w", err)
	}
	defer reader.Close()

	cmd := exec.Command("kubectl", "exec", "-i", "-n", "postgres", podName, "--",
		"psql", "-U", "postgres", "-d", "postgres")
	tools.WithKubeconfig(cmd, kubeconfigPath)
	cmd.Stdin = reader

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restore globals: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

// getDatabaseName determines the database name for the app
func (p *PostgreSQLStrategy) getDatabaseName(instanceName, appName string) string {
	// Read from config.yaml to get the actual database name
	configPath := tools.GetInstanceConfigPath(p.dataDir, instanceName)
	data, err := os.ReadFile(configPath)
	if err != nil {
		// Fall back to app name if config can't be read
		return appName
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		// Fall back to app name if config can't be parsed
		return appName
	}

	// Extract app-specific configuration
	if apps, ok := config["apps"].(map[string]interface{}); ok {
		if appConfig, ok := apps[appName].(map[string]interface{}); ok {
			// Look for dbName field
			if dbName, ok := appConfig["dbName"].(string); ok && dbName != "" {
				return dbName
			}
		}
	}

	// Fall back to app name if dbName not found
	return appName
}
// getPostgresPod finds the first running postgres pod
func (p *PostgreSQLStrategy) getPostgresPod(kubeconfigPath string) (string, error) {
	cmd := exec.Command("kubectl", "get", "pods", "-n", "postgres",
		"-l", "app=postgres", "-o", "jsonpath={.items[0].metadata.name}")
	tools.WithKubeconfig(cmd, kubeconfigPath)

	output, err := cmd.Output()
	if err != nil {
		// Try without label selector in case labels are different
		cmd = exec.Command("kubectl", "get", "pods", "-n", "postgres",
			"-o", "jsonpath={.items[0].metadata.name}")
		tools.WithKubeconfig(cmd, kubeconfigPath)
		output, err = cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to get postgres pod: %w", err)
		}
	}

	podName := strings.TrimSpace(string(output))
	if podName == "" {
		return "", fmt.Errorf("no postgres pod found")
	}

	return podName, nil
}
