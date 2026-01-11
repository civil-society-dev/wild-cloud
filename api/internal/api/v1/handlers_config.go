package v1

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/wild-cloud/wild-central/daemon/internal/config"
)

// ConfigUpdateBatch updates multiple configuration values atomically
func (api *API) ConfigUpdateBatch(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)

	var req ConfigBatchUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.Updates) == 0 {
		respondError(w, http.StatusBadRequest, "updates array is required and cannot be empty")
		return
	}

	configPath := api.instance.GetInstanceConfigPath(instanceName)

	for i, update := range req.Updates {
		if update.Path == "" {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("update[%d]: path is required", i))
			return
		}
	}

	updateCount := 0
	for _, update := range req.Updates {
		valueStr := fmt.Sprintf("%v", update.Value)
		if err := api.config.SetConfigValue(configPath, update.Path, valueStr); err != nil {
			respondError(w, http.StatusInternalServerError,
				fmt.Sprintf("Failed to update config path %s: %v", update.Path, err))
			return
		}
		updateCount++
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Configuration updated successfully",
		"updated": updateCount,
	})
}

// GetGlobalConfig returns the global configuration
func (api *API) GetGlobalConfig(w http.ResponseWriter, r *http.Request) {
	globalConfigPath := api.dataDir + "/config.yaml"

	// Load global config
	globalCfg, err := config.LoadGlobalConfig(globalConfigPath)
	if err != nil {
		// If config doesn't exist, return empty config with configured=false
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"configured": false,
			"config":     nil,
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"configured": !globalCfg.IsEmpty(),
		"config":     globalCfg,
	})
}

// UpdateGlobalConfig updates the global configuration
func (api *API) UpdateGlobalConfig(w http.ResponseWriter, r *http.Request) {
	globalConfigPath := api.dataDir + "/config.yaml"

	// Parse request body
	var globalCfg config.GlobalConfig
	if err := json.NewDecoder(r.Body).Decode(&globalCfg); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Save global config
	if err := config.SaveGlobalConfig(&globalCfg, globalConfigPath); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to save config: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Global configuration updated successfully",
		"config":  globalCfg,
	})
}
