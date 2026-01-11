package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"

	"github.com/wild-cloud/wild-central/daemon/internal/contracts"
	"github.com/wild-cloud/wild-central/daemon/internal/operations"
	"github.com/wild-cloud/wild-central/daemon/internal/services"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
)

// ServicesList lists all base services
func (api *API) ServicesList(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)

	servicesMgr := services.NewManager(api.dataDir)
	svcList, err := servicesMgr.List(instanceName)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list services: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"services": svcList,
	})
}

// ServicesGet returns a specific service
func (api *API) ServicesGet(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	serviceName := GetServiceName(r)

	servicesMgr := services.NewManager(api.dataDir)
	service, err := servicesMgr.Get(instanceName, serviceName)
	if err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Service not found: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, service)
}

// ServicesInstall installs a service
func (api *API) ServicesInstall(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)

	var req ServiceInstallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "service name is required")
		return
	}

	api.StartAsyncOperationWithBroadcaster(w, instanceName, "install_service", req.Name,
		func(opsMgr *operations.Manager, opID string, broadcaster *operations.Broadcaster) error {
			servicesMgr := services.NewManager(api.dataDir)
			return servicesMgr.Install(instanceName, req.Name, req.Fetch, req.Deploy, opID, broadcaster)
		})
}

// ServicesInstallAll installs all base services
func (api *API) ServicesInstallAll(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)

	var req ServiceInstallAllRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Deploy = true
	}

	api.StartAsyncOperationWithBroadcaster(w, instanceName, "install_all_services", "all",
		func(opsMgr *operations.Manager, opID string, broadcaster *operations.Broadcaster) error {
			servicesMgr := services.NewManager(api.dataDir)
			return servicesMgr.InstallAll(instanceName, req.Fetch, req.Deploy, opID, broadcaster)
		})
}

// ServicesDelete deletes a service
func (api *API) ServicesDelete(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	serviceName := GetServiceName(r)

	api.StartAsyncOperation(w, instanceName, "delete_service", serviceName,
		func(opsMgr *operations.Manager, opID string) error {
			servicesMgr := services.NewManager(api.dataDir)
			return servicesMgr.Delete(instanceName, serviceName)
		})
}

// ServicesGetStatus returns detailed service status
func (api *API) ServicesGetStatus(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	serviceName := GetServiceName(r)

	servicesMgr := services.NewManager(api.dataDir)
	status, err := servicesMgr.GetDetailedStatus(instanceName, serviceName)
	if err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Failed to get status: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, status)
}

// ServicesGetManifest returns the manifest for a service
func (api *API) ServicesGetManifest(w http.ResponseWriter, r *http.Request) {
	serviceName := mux.Vars(r)["service"]

	servicesMgr := services.NewManager(api.dataDir)
	manifest, err := servicesMgr.GetManifest(serviceName)
	if err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Service not found: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, manifest)
}

// ServicesGetConfig returns the service configuration schema
func (api *API) ServicesGetConfig(w http.ResponseWriter, r *http.Request) {
	serviceName := mux.Vars(r)["service"]

	servicesMgr := services.NewManager(api.dataDir)
	manifest, err := servicesMgr.GetManifest(serviceName)
	if err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Service not found: %v", err))
		return
	}

	response := map[string]interface{}{
		"configReferences": manifest.ConfigReferences,
		"serviceConfig":    manifest.ServiceConfig,
	}

	respondJSON(w, http.StatusOK, response)
}

// ServicesGetInstanceConfig returns current config values for a service instance
func (api *API) ServicesGetInstanceConfig(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	serviceName := GetServiceName(r)

	servicesMgr := services.NewManager(api.dataDir)
	manifest, err := servicesMgr.GetManifest(serviceName)
	if err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Service not found: %v", err))
		return
	}

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

	configValues := make(map[string]interface{})
	for _, path := range manifest.ConfigReferences {
		if value := getNestedValue(instanceConfig, path); value != nil {
			configValues[path] = value
		}
	}
	for _, cfg := range manifest.ServiceConfig {
		if value := getNestedValue(instanceConfig, cfg.Path); value != nil {
			configValues[cfg.Path] = value
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"config": configValues,
	})
}

// ServicesFetch handles fetching service files to instance
func (api *API) ServicesFetch(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	serviceName := GetServiceName(r)

	servicesMgr := services.NewManager(api.dataDir)
	if err := servicesMgr.Fetch(instanceName, serviceName); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch service: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Service %s files fetched successfully", serviceName),
	})
}

// ServicesCompile handles template compilation
func (api *API) ServicesCompile(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	serviceName := GetServiceName(r)

	servicesMgr := services.NewManager(api.dataDir)
	if err := servicesMgr.Compile(instanceName, serviceName); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to compile templates: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Templates compiled successfully for %s", serviceName),
	})
}

// ServicesDeploy handles service deployment
func (api *API) ServicesDeploy(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	serviceName := GetServiceName(r)

	servicesMgr := services.NewManager(api.dataDir)
	if err := servicesMgr.Deploy(instanceName, serviceName, "", nil); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to deploy service: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Service %s deployed successfully", serviceName),
	})
}

// ServicesGetLogs retrieves or streams service logs
func (api *API) ServicesGetLogs(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	serviceName := GetServiceName(r)

	query := r.URL.Query()
	logsReq := contracts.ServiceLogsRequest{
		Container: query.Get("container"),
		Follow:    query.Get("follow") == "true",
		Previous:  query.Get("previous") == "true",
		Since:     query.Get("since"),
	}

	if tailStr := query.Get("tail"); tailStr != "" {
		var tail int
		if _, err := fmt.Sscanf(tailStr, "%d", &tail); err == nil {
			logsReq.Tail = tail
		}
	}

	if logsReq.Tail < 0 {
		respondError(w, http.StatusBadRequest, "tail parameter must be positive")
		return
	}
	if logsReq.Tail > 5000 {
		respondError(w, http.StatusBadRequest, "tail parameter cannot exceed 5000")
		return
	}
	if logsReq.Previous && logsReq.Follow {
		respondError(w, http.StatusBadRequest, "previous and follow cannot be used together")
		return
	}

	servicesMgr := services.NewManager(api.dataDir)

	if logsReq.Follow {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		if err := servicesMgr.StreamLogs(instanceName, serviceName, logsReq, w); err != nil {
			fmt.Printf("Error streaming logs: %v\n", err)
		}
		return
	}

	logsResp, err := servicesMgr.GetLogs(instanceName, serviceName, logsReq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get logs: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, logsResp)
}

// ServicesUpdateConfig updates service configuration
func (api *API) ServicesUpdateConfig(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	serviceName := GetServiceName(r)

	var update contracts.ServiceConfigUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	if len(update.Config) == 0 {
		respondError(w, http.StatusBadRequest, "config field is required and must not be empty")
		return
	}

	servicesMgr := services.NewManager(api.dataDir)
	response, err := servicesMgr.UpdateConfig(instanceName, serviceName, update, api.broadcaster)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update config: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, response)
}
