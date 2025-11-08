package v1

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/wild-cloud/wild-central/daemon/internal/assets"
)

// SchematicGetInstanceSchematic returns the schematic configuration for an instance
func (api *API) SchematicGetInstanceSchematic(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]

	// Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	configPath := api.instance.GetInstanceConfigPath(instanceName)

	// Get schematic ID from config
	schematicID, err := api.config.GetConfigValue(configPath, "cluster.nodes.talos.schematicId")
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get schematic ID: %v", err))
		return
	}

	// Get version from config
	version, err := api.config.GetConfigValue(configPath, "cluster.nodes.talos.version")
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get version: %v", err))
		return
	}

	// If schematic is configured, get asset status
	var assetStatus interface{}
	if schematicID != "" && schematicID != "null" && version != "" && version != "null" {
		assetsMgr := assets.NewManager(api.dataDir)
		status, err := assetsMgr.GetAssetStatus(schematicID, version)
		if err == nil {
			assetStatus = status
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"schematic_id": schematicID,
		"version":      version,
		"assets":       assetStatus,
	})
}

// SchematicUpdateInstanceSchematic updates the schematic configuration for an instance
func (api *API) SchematicUpdateInstanceSchematic(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]

	// Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	// Parse request body
	var req struct {
		SchematicID string `json:"schematic_id"`
		Version     string `json:"version"`
		Download    bool   `json:"download,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.SchematicID == "" {
		respondError(w, http.StatusBadRequest, "schematic_id is required")
		return
	}

	if req.Version == "" {
		respondError(w, http.StatusBadRequest, "version is required")
		return
	}

	configPath := api.instance.GetInstanceConfigPath(instanceName)

	// Update schematic ID in config
	if err := api.config.SetConfigValue(configPath, "cluster.nodes.talos.schematicId", req.SchematicID); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to set schematic ID: %v", err))
		return
	}

	// Update version in config
	if err := api.config.SetConfigValue(configPath, "cluster.nodes.talos.version", req.Version); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to set version: %v", err))
		return
	}

	response := map[string]interface{}{
		"message":      "Schematic configuration updated successfully",
		"schematic_id": req.SchematicID,
		"version":      req.Version,
	}

	// Optionally download assets
	if req.Download {
		assetsMgr := assets.NewManager(api.dataDir)
		platform := "amd64" // Default platform
		if err := assetsMgr.DownloadAssets(req.SchematicID, req.Version, platform, nil); err != nil {
			response["download_warning"] = fmt.Sprintf("Failed to download assets: %v", err)
		} else {
			response["download_status"] = "Assets downloaded successfully"
		}
	}

	respondJSON(w, http.StatusOK, response)
}
