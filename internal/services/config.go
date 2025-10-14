package services

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/wild-cloud/wild-central/daemon/internal/contracts"
	"github.com/wild-cloud/wild-central/daemon/internal/operations"
	"github.com/wild-cloud/wild-central/daemon/internal/storage"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
)

// UpdateConfig updates service configuration and optionally redeploys
func (m *Manager) UpdateConfig(instanceName, serviceName string, update contracts.ServiceConfigUpdate, broadcaster *operations.Broadcaster) (*contracts.ServiceConfigResponse, error) {
	// 1. Validate service exists
	manifest, err := m.GetManifest(serviceName)
	if err != nil {
		return nil, fmt.Errorf("service not found: %w", err)
	}

	namespace := manifest.Namespace
	if deployment, ok := serviceDeployments[serviceName]; ok {
		namespace = deployment.namespace
	}

	// 2. Load instance config
	configPath := tools.GetInstanceConfigPath(m.dataDir, instanceName)

	if !storage.FileExists(configPath) {
		return nil, fmt.Errorf("config file not found for instance %s", instanceName)
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// 3. Validate config keys against service manifest
	validPaths := make(map[string]bool)
	for _, path := range manifest.ConfigReferences {
		validPaths[path] = true
	}
	for _, cfg := range manifest.ServiceConfig {
		validPaths[cfg.Path] = true
	}

	for key := range update.Config {
		if !validPaths[key] {
			return nil, fmt.Errorf("invalid config key '%s' for service %s", key, serviceName)
		}
	}

	// 4. Update config values
	for key, value := range update.Config {
		if err := setNestedValue(config, key, value); err != nil {
			return nil, fmt.Errorf("failed to set config key '%s': %w", key, err)
		}
	}

	// 5. Write updated config
	updatedData, err := yaml.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, updatedData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write config: %w", err)
	}

	// 6. Redeploy if requested
	redeployed := false
	if update.Redeploy {
		// Fetch fresh templates if requested
		if update.Fetch {
			if err := m.Fetch(instanceName, serviceName); err != nil {
				return nil, fmt.Errorf("failed to fetch templates: %w", err)
			}
		}

		// Recompile templates
		if err := m.Compile(instanceName, serviceName); err != nil {
			return nil, fmt.Errorf("failed to recompile templates: %w", err)
		}

		// Redeploy service
		if err := m.Deploy(instanceName, serviceName, "", broadcaster); err != nil {
			return nil, fmt.Errorf("failed to redeploy service: %w", err)
		}

		redeployed = true
	}

	// 7. Build response
	message := "Service configuration updated successfully"
	if redeployed {
		message = "Service configuration updated and redeployed successfully"
	}

	return &contracts.ServiceConfigResponse{
		Service:    serviceName,
		Namespace:  namespace,
		Config:     update.Config,
		Redeployed: redeployed,
		Message:    message,
	}, nil
}

// setNestedValue sets a value in a nested map using dot notation
func setNestedValue(data map[string]interface{}, path string, value interface{}) error {
	keys := strings.Split(path, ".")
	current := data

	for i, key := range keys {
		if i == len(keys)-1 {
			// Last key - set the value
			current[key] = value
			return nil
		}

		// Navigate to the next level
		if next, ok := current[key].(map[string]interface{}); ok {
			current = next
		} else if current[key] == nil {
			// Create intermediate map if it doesn't exist
			next := make(map[string]interface{})
			current[key] = next
			current = next
		} else {
			return fmt.Errorf("path '%s' conflicts with existing non-map value at '%s'", path, key)
		}
	}

	return nil
}
