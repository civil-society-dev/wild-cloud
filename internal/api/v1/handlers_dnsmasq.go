package v1

import (
	"fmt"
	"log"
	"net/http"

	"github.com/wild-cloud/wild-central/daemon/internal/config"
)

// DnsmasqStatus returns the status of the dnsmasq service
func (api *API) DnsmasqStatus(w http.ResponseWriter, r *http.Request) {
	status, err := api.dnsmasq.GetStatus()
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get dnsmasq status: %v", err))
		return
	}

	if status.Status != "active" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

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

// DnsmasqGenerate generates the dnsmasq configuration without applying it (dry-run)
func (api *API) DnsmasqGenerate(w http.ResponseWriter, r *http.Request) {
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

	// Generate config without writing or restarting
	configContent := api.dnsmasq.Generate(globalCfg, instanceConfigs)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "dnsmasq configuration generated (dry-run mode)",
		"config":  configContent,
	})
}

// DnsmasqUpdate regenerates and updates the dnsmasq configuration with all instances
func (api *API) DnsmasqUpdate(w http.ResponseWriter, r *http.Request) {
	if err := api.updateDnsmasqForAllInstances(); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update dnsmasq: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "dnsmasq configuration updated successfully",
	})
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

	// Regenerate and write dnsmasq config
	return api.dnsmasq.UpdateConfig(globalCfg, instanceConfigs)
}

// getGlobalConfigPath returns the path to the global config file
func (api *API) getGlobalConfigPath() string {
	// This should match the structure from data.Paths
	return api.dataDir + "/config.yaml"
}
