package v1

import (
	"encoding/json"
	"net/http"

	"github.com/wild-cloud/wild-central/daemon/internal/assets"
)

// SchematicGetInstanceSchematic returns the schematic configuration for an instance
func (api *API) SchematicGetInstanceSchematic(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)

	configPath := api.instance.GetInstanceConfigPath(instanceName)

	// Get schematic ID from config
	schematicID, err := api.config.GetConfigValue(configPath, "cluster.nodes.talos.schematicId")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get schematic ID: "+err.Error())
		return
	}

	// Get version from config
	version, err := api.config.GetConfigValue(configPath, "cluster.nodes.talos.version")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get version: "+err.Error())
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
	instanceName := GetInstanceName(r)

	// Parse request body
	var req SchematicUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.SchematicID == "" {
		respondError(w, http.StatusBadRequest, "schematicId is required")
		return
	}

	if req.Version == "" {
		respondError(w, http.StatusBadRequest, "version is required")
		return
	}

	configPath := api.instance.GetInstanceConfigPath(instanceName)

	// Update schematic ID in config
	if err := api.config.SetConfigValue(configPath, "cluster.nodes.talos.schematicId", req.SchematicID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to set schematic ID: "+err.Error())
		return
	}

	// Update version in config
	if err := api.config.SetConfigValue(configPath, "cluster.nodes.talos.version", req.Version); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to set version: "+err.Error())
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
			response["download_warning"] = "Failed to download assets: " + err.Error()
		} else {
			response["download_status"] = "Assets downloaded successfully"
		}
	}

	respondJSON(w, http.StatusOK, response)
}
