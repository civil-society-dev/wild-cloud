package v1

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/wild-cloud/wild-central/daemon/internal/config"
)

// DnsmasqStatus returns the status of the dnsmasq service
func (api *API) DnsmasqStatus(w http.ResponseWriter, r *http.Request) {
	status, err := api.dnsmasq.GetStatus()
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get dnsmasq status: %v", err))
		return
	}

	// Always return 200 OK with status in body - let client handle inactive status
	respondJSON(w, http.StatusOK, status)
}

// DnsmasqGetConfig returns the current dnsmasq configuration
func (api *API) DnsmasqGetConfig(w http.ResponseWriter, r *http.Request) {
	configContent, err := api.dnsmasq.ReadConfig()
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to read dnsmasq config: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"config_file": api.dnsmasq.GetConfigPath(),
		"content":     configContent,
	})
}

// DnsmasqRestart restarts the dnsmasq service
func (api *API) DnsmasqRestart(w http.ResponseWriter, r *http.Request) {
	if err := api.dnsmasq.RestartService(); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to restart dnsmasq: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "dnsmasq service restarted successfully",
	})
}

// DnsmasqGenerate generates the dnsmasq configuration from all instances
// Query param ?overwrite=true will write the config and restart the service
func (api *API) DnsmasqGenerate(w http.ResponseWriter, r *http.Request) {
	// Check if overwrite flag is set
	overwrite := r.URL.Query().Get("overwrite") == "true"

	// Get all instances
	instanceNames, err := api.instance.ListInstances()
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list instances: %v", err))
		return
	}

	// Load global config
	globalConfigPath := api.getGlobalConfigPath()
	globalCfg, err := config.LoadGlobalConfig(globalConfigPath)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to load global config: %v", err))
		return
	}

	// Load all instance configs
	var instanceConfigs []config.InstanceConfig
	for _, name := range instanceNames {
		instanceConfigPath := api.instance.GetInstanceConfigPath(name)
		instanceCfg, err := config.LoadCloudConfig(instanceConfigPath)
		if err != nil {
			log.Printf("Warning: Could not load instance config for %s: %v", name, err)
			continue
		}
		instanceConfigs = append(instanceConfigs, *instanceCfg)
	}

	// Generate config
	configContent := api.dnsmasq.Generate(globalCfg, instanceConfigs)

	if overwrite {
		// Check if this is the first time dnsmasq is being started
		status, err := api.dnsmasq.GetStatus()
		isFirstStart := err != nil || status.Status != "active"

		// Write config and restart service
		if err := api.dnsmasq.UpdateConfig(globalCfg, instanceConfigs, true); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update dnsmasq: %v", err))
			return
		}

		// Configure system DNS to use local dnsmasq on first start
		if isFirstStart {
			if err := api.dnsmasq.ConfigureSystemDNS(); err != nil {
				log.Printf("Warning: Failed to configure system DNS: %v", err)
				// Don't fail the request - dnsmasq is still running
			}
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"message": "dnsmasq configuration generated and applied successfully",
			"config":  configContent,
		})
	} else {
		// Just return the generated config
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"message": "dnsmasq configuration generated (preview mode)",
			"config":  configContent,
		})
	}
}

// DnsmasqWriteConfig writes custom config content to the dnsmasq config file
func (api *API) DnsmasqWriteConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	if req.Content == "" {
		respondError(w, http.StatusBadRequest, "Config content is required")
		return
	}

	// Write the config directly using the dnsmasq config generator's WriteConfig
	configPath := api.dnsmasq.GetConfigPath()
	if err := writeConfigFile(configPath, req.Content); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to write config: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "dnsmasq configuration written successfully",
	})
}

// writeConfigFile writes content to a file
func writeConfigFile(path, content string) error {
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

// updateDnsmasqForAllInstances helper regenerates dnsmasq config from all instances
func (api *API) updateDnsmasqForAllInstances() error {
	// Get all instances
	instanceNames, err := api.instance.ListInstances()
	if err != nil {
		return fmt.Errorf("listing instances: %w", err)
	}

	// Load global config
	globalConfigPath := api.getGlobalConfigPath()
	globalCfg, err := config.LoadGlobalConfig(globalConfigPath)
	if err != nil {
		return fmt.Errorf("loading global config: %w", err)
	}

	// Load all instance configs
	var instanceConfigs []config.InstanceConfig
	for _, name := range instanceNames {
		instanceConfigPath := api.instance.GetInstanceConfigPath(name)
		instanceCfg, err := config.LoadCloudConfig(instanceConfigPath)
		if err != nil {
			log.Printf("Warning: Could not load instance config for %s: %v", name, err)
			continue
		}
		instanceConfigs = append(instanceConfigs, *instanceCfg)
	}

	// Regenerate and write dnsmasq config with restart
	return api.dnsmasq.UpdateConfig(globalCfg, instanceConfigs, true)
}

// getGlobalConfigPath returns the path to the global config file
func (api *API) getGlobalConfigPath() string {
	// This should match the structure from data.Paths
	return api.dataDir + "/config.yaml"
}
