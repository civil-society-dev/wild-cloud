package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"github.com/wild-cloud/wild-central/daemon/internal/apps"
	"github.com/wild-cloud/wild-central/daemon/internal/operations"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
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
		Name   string                 `json:"name"`
		Config map[string]interface{} `json:"config"`
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

// AppsGetEnhanced returns enhanced app details with runtime status
func (api *API) AppsGetEnhanced(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]
	appName := vars["app"]

	// Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	// Get enhanced app details
	appsMgr := apps.NewManager(api.dataDir, api.appsDir)
	enhanced, err := appsMgr.GetEnhanced(instanceName, appName)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get app details: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, enhanced)
}

// AppsGetEnhancedStatus returns just runtime status for an app
func (api *API) AppsGetEnhancedStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]
	appName := vars["app"]

	// Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	// Get runtime status
	appsMgr := apps.NewManager(api.dataDir, api.appsDir)
	status, err := appsMgr.GetEnhancedStatus(instanceName, appName)
	if err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Failed to get runtime status: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, status)
}

// AppsGetLogs returns logs for an app (from first pod)
func (api *API) AppsGetLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]
	appName := vars["app"]

	// Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	// Parse query parameters
	tailStr := r.URL.Query().Get("tail")
	sinceSecondsStr := r.URL.Query().Get("sinceSeconds")
	podName := r.URL.Query().Get("pod")

	tail := 100 // default
	if tailStr != "" {
		if t, err := strconv.Atoi(tailStr); err == nil && t > 0 {
			tail = t
		}
	}

	sinceSeconds := 0
	if sinceSecondsStr != "" {
		if s, err := strconv.Atoi(sinceSecondsStr); err == nil && s > 0 {
			sinceSeconds = s
		}
	}

	// Get logs
	kubeconfigPath := api.dataDir + "/instances/" + instanceName + "/kubeconfig"
	kubectl := tools.NewKubectl(kubeconfigPath)

	// If no pod specified, get the first pod
	if podName == "" {
		pods, err := kubectl.GetPods(appName, true)
		if err != nil || len(pods) == 0 {
			respondError(w, http.StatusNotFound, "No pods found for app")
			return
		}
		podName = pods[0].Name
	}

	logOpts := tools.LogOptions{
		Tail:         tail,
		SinceSeconds: sinceSeconds,
	}
	logs, err := kubectl.GetLogs(appName, podName, logOpts)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get logs: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"pod":  podName,
		"logs": logs,
	})
}

// AppsGetEvents returns kubernetes events for an app
func (api *API) AppsGetEvents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]
	appName := vars["app"]

	// Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 20 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Get events
	kubeconfigPath := api.dataDir + "/instances/" + instanceName + "/kubeconfig"
	kubectl := tools.NewKubectl(kubeconfigPath)

	events, err := kubectl.GetRecentEvents(appName, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get events: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"events": events,
	})
}

// AppsGetReadme returns the README.md content for an app
func (api *API) AppsGetReadme(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]
	appName := vars["app"]

	// Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	// Validate app name to prevent path traversal
	if appName == "" || appName == "." || appName == ".." ||
		strings.Contains(appName, "/") || strings.Contains(appName, "\\") {
		respondError(w, http.StatusBadRequest, "Invalid app name")
		return
	}

	// Try instance-specific README first
	instancePath := filepath.Join(api.dataDir, "instances", instanceName, "apps", appName, "README.md")
	content, err := os.ReadFile(instancePath)
	if err == nil {
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Write(content)
		return
	}

	// Fall back to global directory
	globalPath := filepath.Join(api.appsDir, appName, "README.md")
	content, err = os.ReadFile(globalPath)
	if err != nil {
		if os.IsNotExist(err) {
			respondError(w, http.StatusNotFound, fmt.Sprintf("README not found for app '%s' in instance '%s'", appName, instanceName))
		} else {
			respondError(w, http.StatusInternalServerError, "Failed to read README file")
		}
		return
	}

	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Write(content)
}
