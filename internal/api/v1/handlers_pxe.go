package v1

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/wild-cloud/wild-central/daemon/internal/pxe"
)

// PXEListAssets lists all PXE assets for an instance
func (api *API) PXEListAssets(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]

	// Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	// List assets
	pxeMgr := pxe.NewManager(api.dataDir)
	assets, err := pxeMgr.ListAssets(instanceName)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list assets: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"assets": assets,
	})
}

// PXEDownloadAsset downloads a PXE asset
func (api *API) PXEDownloadAsset(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]

	// Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

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

	if req.URL == "" {
		respondError(w, http.StatusBadRequest, "url is required")
		return
	}

	// Download asset
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
}

// PXEGetAsset returns information about a specific asset
func (api *API) PXEGetAsset(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]
	assetType := vars["type"]

	// Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	// Get asset path
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
}

// PXEDeleteAsset deletes a PXE asset
func (api *API) PXEDeleteAsset(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]
	assetType := vars["type"]

	// Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	// Delete asset
	pxeMgr := pxe.NewManager(api.dataDir)
	if err := pxeMgr.DeleteAsset(instanceName, assetType); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete asset: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Asset deleted successfully",
		"type":    assetType,
	})
}
