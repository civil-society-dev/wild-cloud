package v1

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/wild-cloud/wild-central/daemon/internal/apps"
	"github.com/wild-cloud/wild-central/daemon/internal/operations"
)

// AppsListAvailable lists all available apps
func (api *API) AppsListAvailable(w http.ResponseWriter, r *http.Request) {
	// List available apps from apps directory
	appsMgr := apps.NewManager(api.dataDir, api.appsDir)
	appList, err := appsMgr.ListAvailable()
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list apps: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"apps": appList,
	})
}

// AppsGetAvailable returns details for an available app
func (api *API) AppsGetAvailable(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appName := vars["app"]

	// Get app details
	appsMgr := apps.NewManager(api.dataDir, api.appsDir)
	app, err := appsMgr.Get(appName)
	if err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("App not found: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, app)
}

// AppsListDeployed lists deployed apps for an instance
func (api *API) AppsListDeployed(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]

	// Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	// List deployed apps
	appsMgr := apps.NewManager(api.dataDir, api.appsDir)
	deployedApps, err := appsMgr.ListDeployed(instanceName)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list apps: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"apps": deployedApps,
	})
}

// AppsAdd adds an app to instance configuration
func (api *API) AppsAdd(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]

	// Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	// Parse request
	var req struct {
		Name   string            `json:"name"`
		Config map[string]string `json:"config"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "app name is required")
		return
	}

	// Add app
	appsMgr := apps.NewManager(api.dataDir, api.appsDir)
	if err := appsMgr.Add(instanceName, req.Name, req.Config); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to add app: %v", err))
		return
	}

	respondJSON(w, http.StatusCreated, map[string]string{
		"message": "App added to configuration",
		"app":     req.Name,
	})
}

// startAppOperation starts an app operation (deploy or delete) in the background
func (api *API) startAppOperation(w http.ResponseWriter, instanceName, appName, operationType, successMessage string, operation func(*apps.Manager, string, string) error) {
	// Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	// Start operation
	opsMgr := operations.NewManager(api.dataDir)
	opID, err := opsMgr.Start(instanceName, operationType, appName)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to start operation: %v", err))
		return
	}

	// Execute operation in background
	go func() {
		appsMgr := apps.NewManager(api.dataDir, api.appsDir)
		_ = opsMgr.UpdateStatus(instanceName, opID, "running")

		if err := operation(appsMgr, instanceName, appName); err != nil {
			_ = opsMgr.Update(instanceName, opID, "failed", err.Error(), 0)
		} else {
			_ = opsMgr.Update(instanceName, opID, "completed", successMessage, 100)
		}
	}()

	respondJSON(w, http.StatusAccepted, map[string]string{
		"operation_id": opID,
		"message":      fmt.Sprintf("App %s initiated", operationType),
	})
}

// AppsDeploy deploys an app to the cluster
func (api *API) AppsDeploy(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]
	appName := vars["app"]

	api.startAppOperation(w, instanceName, appName, "deploy_app", "App deployed",
		func(mgr *apps.Manager, instance, app string) error {
			return mgr.Deploy(instance, app)
		})
}

// AppsDelete deletes an app
func (api *API) AppsDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]
	appName := vars["app"]

	api.startAppOperation(w, instanceName, appName, "delete_app", "App deleted",
		func(mgr *apps.Manager, instance, app string) error {
			return mgr.Delete(instance, app)
		})
}

// AppsGetStatus returns app status
func (api *API) AppsGetStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]
	appName := vars["app"]

	// Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	// Get status
	appsMgr := apps.NewManager(api.dataDir, api.appsDir)
	status, err := appsMgr.GetStatus(instanceName, appName)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get status: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, status)
}
