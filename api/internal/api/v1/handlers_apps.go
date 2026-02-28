package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"

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
	instanceName := GetInstanceName(r)

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
	instanceName := GetInstanceName(r)

	var req AppAddRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "app name is required")
		return
	}

	appsMgr := apps.NewManager(api.dataDir, api.appsDir)
	if err := appsMgr.Add(instanceName, req.Name, req.Config, req.RequiredAppMappings); err != nil {
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
	api.StartAsyncOperationWithMessage(w, instanceName, operationType, appName, successMessage,
		func(opsMgr *operations.Manager, opID string) error {
			appsMgr := apps.NewManager(api.dataDir, api.appsDir)
			return operation(appsMgr, instanceName, appName)
		})
}

// AppsDeploy deploys an app to the cluster
func (api *API) AppsDeploy(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	appName := GetAppName(r)

	api.startAppOperation(w, instanceName, appName, "deploy_app", "App deployed",
		func(mgr *apps.Manager, instance, app string) error {
			return mgr.Deploy(instance, app)
		})
}

// AppsRestart performs a rolling restart of an app's pods
func (api *API) AppsRestart(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	appName := GetAppName(r)

	api.startAppOperation(w, instanceName, appName, "restart_app", "App restarted",
		func(mgr *apps.Manager, instance, app string) error {
			return mgr.Restart(instance, app)
		})
}

// AppsDelete deletes an app
func (api *API) AppsDelete(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	appName := GetAppName(r)

	api.startAppOperation(w, instanceName, appName, "delete_app", "App deleted",
		func(mgr *apps.Manager, instance, app string) error {
			return mgr.Delete(instance, app)
		})
}

// AppsUpdate updates an app from its source
func (api *API) AppsUpdate(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	appName := GetAppName(r)

	api.startAppOperation(w, instanceName, appName, "update_app", "App updated",
		func(mgr *apps.Manager, instance, app string) error {
			return mgr.Update(instance, app)
		})
}

// AppsEject converts an app from package-managed to custom
func (api *API) AppsEject(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	appName := GetAppName(r)

	appsMgr := apps.NewManager(api.dataDir, api.appsDir)
	if err := appsMgr.Eject(instanceName, appName); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to eject app: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "App converted to custom",
		"app":     appName,
	})
}

// AppsGetConfig returns current config values for an app instance
func (api *API) AppsGetConfig(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	appName := GetAppName(r)

	configPath := tools.GetInstanceConfigPath(api.dataDir, instanceName)
	configData, err := os.ReadFile(configPath)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to read instance config: %v", err))
		return
	}

	var instanceConfig map[string]interface{}
	if err := yaml.Unmarshal(configData, &instanceConfig); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to parse instance config: %v", err))
		return
	}

	// Extract app config
	var appConfig map[string]interface{}
	if appsSection, ok := instanceConfig["apps"].(map[string]interface{}); ok {
		if appConfigData, ok := appsSection[appName].(map[string]interface{}); ok {
			appConfig = appConfigData
		}
	}

	if appConfig == nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Config not found for app: %s", appName))
		return
	}

	respondJSON(w, http.StatusOK, appConfig)
}

// AppsUpdateConfig updates an app's configuration
func (api *API) AppsUpdateConfig(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	appName := GetAppName(r)

	var req AppConfigUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	appsMgr := apps.NewManager(api.dataDir, api.appsDir)
	if err := appsMgr.UpdateConfig(instanceName, appName, req.Config); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update config: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "App configuration updated",
		"app":     appName,
	})
}

// AppsGetStatus returns app status
func (api *API) AppsGetStatus(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	appName := GetAppName(r)

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
	instanceName := GetInstanceName(r)
	appName := GetAppName(r)

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
	instanceName := GetInstanceName(r)
	appName := GetAppName(r)

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
	instanceName := GetInstanceName(r)
	appName := GetAppName(r)

	tailStr := r.URL.Query().Get("tail")
	sinceSecondsStr := r.URL.Query().Get("sinceSeconds")
	podName := r.URL.Query().Get("pod")
	containerName := r.URL.Query().Get("container")

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

	kubeconfigPath := tools.GetKubeconfigPath(api.dataDir, instanceName)
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
		Container:    containerName,
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
	instanceName := GetInstanceName(r)
	appName := GetAppName(r)

	limitStr := r.URL.Query().Get("limit")
	limit := 20 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	kubeconfigPath := tools.GetKubeconfigPath(api.dataDir, instanceName)
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
	instanceName := GetInstanceName(r)
	appName := GetAppName(r)

	if err := ValidateAppName(appName); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
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
