package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	"github.com/wild-cloud/wild-central/daemon/internal/assets"
)

// AssetsList lists all available assets (schematic@version combinations)
func (api *API) AssetsList(w http.ResponseWriter, r *http.Request) {
	assetsMgr := assets.NewManager(api.dataDir)

	assetList, err := assetsMgr.ListAssets()
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list assets: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"assets": assetList,
	})
}

// AssetsGet returns details for a specific asset (schematic@version)
func (api *API) AssetsGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	schematicID := vars["schematicId"]
	version := vars["version"]

	assetsMgr := assets.NewManager(api.dataDir)

	asset, err := assetsMgr.GetAsset(schematicID, version)
	if err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Asset not found: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, asset)
}

// AssetsDownload downloads assets for a schematic@version
func (api *API) AssetsDownload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	schematicID := vars["schematicId"]
	version := vars["version"]

	// Parse request body
	var req struct {
		Platform   string   `json:"platform,omitempty"`
		AssetTypes []string `json:"asset_types,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Default platform to amd64 if not specified
	if req.Platform == "" {
		req.Platform = "amd64"
	}

	// Download assets
	assetsMgr := assets.NewManager(api.dataDir)
	if err := assetsMgr.DownloadAssets(schematicID, version, req.Platform, req.AssetTypes); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to download assets: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "Assets downloaded successfully",
		"schematic_id": schematicID,
		"version":      version,
		"platform":     req.Platform,
	})
}

// AssetsServePXE serves a PXE asset file
func (api *API) AssetsServePXE(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	schematicID := vars["schematicId"]
	version := vars["version"]
	assetType := vars["assetType"]

	assetsMgr := assets.NewManager(api.dataDir)

	// Get asset path
	assetPath, err := assetsMgr.GetAssetPath(schematicID, version, assetType)
	if err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Asset not found: %v", err))
		return
	}

	// Open file
	file, err := os.Open(assetPath)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to open asset: %v", err))
		return
	}
	defer file.Close()

	// Get file info for size
	info, err := file.Stat()
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to stat asset: %v", err))
		return
	}

	// Set appropriate content type
	var contentType string
	switch assetType {
	case "kernel":
		contentType = "application/octet-stream"
	case "initramfs":
		contentType = "application/x-xz"
	case "iso":
		contentType = "application/x-iso9660-image"
	default:
		contentType = "application/octet-stream"
	}

	// Set headers
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
	// Set Content-Disposition to suggest filename for download
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", info.Name()))

	// Serve file
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}

// AssetsGetStatus returns download status for a schematic@version
func (api *API) AssetsGetStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	schematicID := vars["schematicId"]
	version := vars["version"]

	assetsMgr := assets.NewManager(api.dataDir)

	status, err := assetsMgr.GetAssetStatus(schematicID, version)
	if err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Asset not found: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, status)
}

// AssetsDelete deletes an asset (schematic@version) and all its files
func (api *API) AssetsDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	schematicID := vars["schematicId"]
	version := vars["version"]

	assetsMgr := assets.NewManager(api.dataDir)

	if err := assetsMgr.DeleteAsset(schematicID, version); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete asset: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message":      "Asset deleted successfully",
		"schematic_id": schematicID,
		"version":      version,
	})
}
