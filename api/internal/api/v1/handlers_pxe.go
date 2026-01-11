package v1

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/wild-cloud/wild-central/daemon/internal/assets"
	"github.com/wild-cloud/wild-central/daemon/internal/pxe"
)

// PXEListAssets lists all PXE assets for an instance
// DEPRECATED: This endpoint is deprecated. Use GET /api/v1/assets/{schematicId} instead.
func (api *API) PXEListAssets(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)

	w.Header().Set("X-Deprecated", "This endpoint is deprecated. Use GET /api/v1/assets/{schematicId} instead.")
	log.Printf("Warning: Deprecated endpoint /api/v1/instances/%s/pxe/assets called", instanceName)

	// Get schematic ID from instance config
	configPath := api.instance.GetInstanceConfigPath(instanceName)
	schematicID, err := api.config.GetConfigValue(configPath, "cluster.nodes.talos.schematicId")
	if err != nil || schematicID == "" || schematicID == "null" {
		// Fall back to old PXE manager if no schematic configured
		pxeMgr := pxe.NewManager(api.dataDir)
		assets, err := pxeMgr.ListAssets(instanceName)
		if err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list assets: %v", err))
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"assets": assets,
		})
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"assets":  []interface{}{},
		"message": "Please use the new /api/v1/pxe/assets endpoint with both schematic ID and version",
	})
}

// PXEDownloadAsset downloads a PXE asset
// DEPRECATED: This endpoint is deprecated. Use POST /api/v1/assets/{schematicId}/download instead.
func (api *API) PXEDownloadAsset(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)

	w.Header().Set("X-Deprecated", "This endpoint is deprecated. Use POST /api/v1/assets/{schematicId}/download instead.")
	log.Printf("Warning: Deprecated endpoint /api/v1/instances/%s/pxe/assets/download called", instanceName)

	// Parse request
	var req struct {
		AssetType string `json:"asset_type"` // kernel, initramfs, iso
		Version   string `json:"version"`
		URL       string `json:"url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.AssetType == "" {
		respondError(w, http.StatusBadRequest, "asset_type is required")
		return
	}

	// Get schematic ID from instance config
	configPath := api.instance.GetInstanceConfigPath(instanceName)
	schematicID, err := api.config.GetConfigValue(configPath, "cluster.nodes.talos.schematicId")

	// If no schematic configured or URL provided (old behavior), fall back to old PXE manager
	if (err != nil || schematicID == "" || schematicID == "null") || req.URL != "" {
		if req.URL == "" {
			respondError(w, http.StatusBadRequest, "url is required when schematic is not configured")
			return
		}

		// Download asset using old PXE manager
		pxeMgr := pxe.NewManager(api.dataDir)
		if err := pxeMgr.DownloadAsset(instanceName, req.AssetType, req.Version, req.URL); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to download asset: %v", err))
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"message":    "Asset downloaded successfully",
			"asset_type": req.AssetType,
			"version":    req.Version,
		})
		return
	}

	// Proxy to new asset system
	if req.Version == "" {
		respondError(w, http.StatusBadRequest, "version is required")
		return
	}

	assetsMgr := assets.NewManager(api.dataDir)
	assetTypes := []string{req.AssetType}
	platform := "amd64" // Default platform for backward compatibility
	if err := assetsMgr.DownloadAssets(schematicID, req.Version, platform, assetTypes); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to download asset: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message":      "Asset downloaded successfully",
		"asset_type":   req.AssetType,
		"version":      req.Version,
		"schematic_id": schematicID,
	})
}

// PXEGetAsset returns information about a specific asset
// DEPRECATED: This endpoint is deprecated. Use GET /api/v1/assets/{schematicId}/pxe/{assetType} instead.
func (api *API) PXEGetAsset(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	assetType := mux.Vars(r)["type"]

	w.Header().Set("X-Deprecated", "This endpoint is deprecated. Use GET /api/v1/assets/{schematicId}/pxe/{assetType} instead.")
	log.Printf("Warning: Deprecated endpoint /api/v1/instances/%s/pxe/assets/%s called", instanceName, assetType)

	// Get schematic ID from instance config
	configPath := api.instance.GetInstanceConfigPath(instanceName)
	schematicID, err := api.config.GetConfigValue(configPath, "cluster.nodes.talos.schematicId")
	if err != nil || schematicID == "" || schematicID == "null" {
		// Fall back to old PXE manager if no schematic configured
		pxeMgr := pxe.NewManager(api.dataDir)
		assetPath, err := pxeMgr.GetAssetPath(instanceName, assetType)
		if err != nil {
			respondError(w, http.StatusNotFound, fmt.Sprintf("Asset not found: %v", err))
			return
		}

		// Verify asset
		valid, err := pxeMgr.VerifyAsset(instanceName, assetType)
		if err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to verify asset: %v", err))
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"type":  assetType,
			"path":  assetPath,
			"valid": valid,
		})
		return
	}

	respondError(w, http.StatusBadRequest, "This deprecated endpoint requires version. Please use /api/v1/pxe/assets/{schematicId}/{version}/pxe/{assetType}")
}

// PXEDeleteAsset deletes a PXE asset
// DEPRECATED: This endpoint is deprecated. Use DELETE /api/v1/assets/{schematicId} instead.
func (api *API) PXEDeleteAsset(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	assetType := mux.Vars(r)["type"]

	w.Header().Set("X-Deprecated", "This endpoint is deprecated. Use DELETE /api/v1/assets/{schematicId} instead.")
	log.Printf("Warning: Deprecated endpoint DELETE /api/v1/instances/%s/pxe/assets/%s called", instanceName, assetType)

	// Get schematic ID from instance config
	configPath := api.instance.GetInstanceConfigPath(instanceName)
	schematicID, err := api.config.GetConfigValue(configPath, "cluster.nodes.talos.schematicId")
	if err != nil || schematicID == "" || schematicID == "null" {
		// Fall back to old PXE manager if no schematic configured
		pxeMgr := pxe.NewManager(api.dataDir)
		if err := pxeMgr.DeleteAsset(instanceName, assetType); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete asset: %v", err))
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"message": "Asset deleted successfully",
			"type":    assetType,
		})
		return
	}

	// Note: In the new system, we don't delete individual assets, only entire schematics
	// For backward compatibility, we'll just report success without doing anything
	respondJSON(w, http.StatusOK, map[string]string{
		"message":      "Individual asset deletion not supported in schematic mode. Use DELETE /api/v1/assets/{schematicId} to delete all assets.",
		"type":         assetType,
		"schematic_id": schematicID,
	})
}
