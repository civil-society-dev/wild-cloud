package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GetWildCLIDataDir returns the Wild CLI data directory
func GetWildCLIDataDir() string {
	if dir := os.Getenv("WILD_CLI_DATA"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".wildcloud"
	}
	return filepath.Join(home, ".wildcloud")
}

// GetCurrentInstance resolves the current instance using the priority cascade:
// 1. --instance flag (passed as parameter)
// 2. $WILD_CLI_DATA/current_instance file
// 3. Auto-select first instance from API
func GetCurrentInstance(flagInstance string, apiClient InstanceLister) (string, string, error) {
	// Priority 1: --instance flag
	if flagInstance != "" {
		return flagInstance, "flag", nil
	}

	// Priority 2: current_instance file
	dataDir := GetWildCLIDataDir()
	currentFile := filepath.Join(dataDir, "current_instance")

	if data, err := os.ReadFile(currentFile); err == nil {
		instance := strings.TrimSpace(string(data))
		if instance != "" {
			return instance, "file", nil
		}
	}

	// Priority 3: Auto-select first instance from API
	if apiClient != nil {
		instances, err := apiClient.ListInstances()
		if err != nil {
			return "", "", fmt.Errorf("no instance configured and failed to list instances: %w", err)
		}

		if len(instances) == 0 {
			return "", "", fmt.Errorf("no instance configured and no instances available (create one with: wild instance create <name>)")
		}

		// Auto-select first instance
		return instances[0], "auto", nil
	}

	return "", "", fmt.Errorf("no instance configured (use --instance flag or run: wild instance use <name>)")
}

// SetCurrentInstance persists the instance selection to file
func SetCurrentInstance(instance string) error {
	dataDir := GetWildCLIDataDir()

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	currentFile := filepath.Join(dataDir, "current_instance")

	// Write instance name to file
	if err := os.WriteFile(currentFile, []byte(instance), 0644); err != nil {
		return fmt.Errorf("failed to write current instance file: %w", err)
	}

	return nil
}

// InstanceLister is an interface for listing instances (allows for testing and dependency injection)
type InstanceLister interface {
	ListInstances() ([]string, error)
}
