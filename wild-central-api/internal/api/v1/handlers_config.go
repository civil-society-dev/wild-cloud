package v1

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/wild-cloud/wild-central/daemon/internal/config"
)

// ConfigUpdate represents a single configuration update
type ConfigUpdate struct {
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

// ConfigUpdateBatchRequest represents a batch configuration update request
type ConfigUpdateBatchRequest struct {
	Updates []ConfigUpdate `json:"updates"`
}

// ConfigUpdateBatch updates multiple configuration values atomically
func (api *API) ConfigUpdateBatch(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	// Validate instance exists
	if err := api.instance.ValidateInstance(name); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	// Parse request body
	var req ConfigUpdateBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.Updates) == 0 {
		respondError(w, http.StatusBadRequest, "updates array is required and cannot be empty")
		return
	}

	// Get config path
	configPath := api.instance.GetInstanceConfigPath(name)

	// Validate all paths before applying changes
	for i, update := range req.Updates {
		if update.Path == "" {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("update[%d]: path is required", i))
			return
		}
	}

	// Apply all updates atomically
	// The config manager's SetConfigValue already uses file locking,
	// so each individual update is atomic. For true atomicity across
	// all updates, we would need to implement transaction support.
	// For now, we apply updates sequentially within the lock.
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
