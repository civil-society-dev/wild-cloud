package v1

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"

	"github.com/wild-cloud/wild-central/daemon/internal/config"
	"github.com/wild-cloud/wild-central/daemon/internal/context"
	"github.com/wild-cloud/wild-central/daemon/internal/dnsmasq"
	"github.com/wild-cloud/wild-central/daemon/internal/instance"
	"github.com/wild-cloud/wild-central/daemon/internal/operations"
	"github.com/wild-cloud/wild-central/daemon/internal/secrets"
)

// API holds all dependencies for API handlers
type API struct {
	dataDir     string
	appsDir     string // Path to external apps directory
	config      *config.Manager
	secrets     *secrets.Manager
	context     *context.Manager
	instance    *instance.Manager
	dnsmasq     *dnsmasq.ConfigGenerator
	broadcaster *operations.Broadcaster // SSE broadcaster for operation output
}

// NewAPI creates a new API handler with all dependencies
// Note: Setup files (cluster-services, cluster-nodes, etc.) are now embedded in the binary
func NewAPI(dataDir, appsDir string) (*API, error) {
	// Ensure base directories exist
	instancesDir := filepath.Join(dataDir, "instances")
	if err := os.MkdirAll(instancesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create instances directory: %w", err)
	}

	// Determine dnsmasq config path
	dnsmasqConfigPath := "/etc/dnsmasq.d/wild-cloud.conf"
	if os.Getenv("WILD_API_DNSMASQ_CONFIG_PATH") != "" {
		dnsmasqConfigPath = os.Getenv("WILD_API_DNSMASQ_CONFIG_PATH")
		log.Printf("Using custom dnsmasq config path: %s", dnsmasqConfigPath)
	}

	return &API{
		dataDir:     dataDir,
		appsDir:     appsDir,
		config:      config.NewManager(),
		secrets:     secrets.NewManager(),
		context:     context.NewManager(dataDir),
		instance:    instance.NewManager(dataDir),
		dnsmasq:     dnsmasq.NewConfigGenerator(dnsmasqConfigPath),
		broadcaster: operations.NewBroadcaster(),
	}, nil
}

func (api *API) RegisterRoutes(r *mux.Router) {
	// Instance management
	r.HandleFunc("/api/v1/instances", api.CreateInstance).Methods("POST")
	r.HandleFunc("/api/v1/instances", api.ListInstances).Methods("GET")
	r.HandleFunc("/api/v1/instances/{name}", api.GetInstance).Methods("GET")
	r.HandleFunc("/api/v1/instances/{name}", api.DeleteInstance).Methods("DELETE")

	// Config management
	r.HandleFunc("/api/v1/instances/{name}/config", api.GetConfig).Methods("GET")
	r.HandleFunc("/api/v1/instances/{name}/config", api.UpdateConfig).Methods("PUT")
	r.HandleFunc("/api/v1/instances/{name}/config", api.ConfigUpdateBatch).Methods("PATCH")

	// Secrets management
	r.HandleFunc("/api/v1/instances/{name}/secrets", api.GetSecrets).Methods("GET")
	r.HandleFunc("/api/v1/instances/{name}/secrets", api.UpdateSecrets).Methods("PUT")

	// Context management
	r.HandleFunc("/api/v1/context", api.GetContext).Methods("GET")
	r.HandleFunc("/api/v1/context", api.SetContext).Methods("POST")

	// Node management
	r.HandleFunc("/api/v1/instances/{name}/nodes/discover", api.NodeDiscover).Methods("POST")
	r.HandleFunc("/api/v1/instances/{name}/nodes/detect", api.NodeDetect).Methods("POST")
	r.HandleFunc("/api/v1/instances/{name}/discovery", api.NodeDiscoveryStatus).Methods("GET")
	r.HandleFunc("/api/v1/instances/{name}/nodes/hardware/{ip}", api.NodeHardware).Methods("GET")
	r.HandleFunc("/api/v1/instances/{name}/nodes/fetch-templates", api.NodeFetchTemplates).Methods("POST")
	r.HandleFunc("/api/v1/instances/{name}/nodes", api.NodeAdd).Methods("POST")
	r.HandleFunc("/api/v1/instances/{name}/nodes", api.NodeList).Methods("GET")
	r.HandleFunc("/api/v1/instances/{name}/nodes/{node}", api.NodeGet).Methods("GET")
	r.HandleFunc("/api/v1/instances/{name}/nodes/{node}", api.NodeUpdate).Methods("PUT")
	r.HandleFunc("/api/v1/instances/{name}/nodes/{node}/apply", api.NodeApply).Methods("POST")
	r.HandleFunc("/api/v1/instances/{name}/nodes/{node}", api.NodeDelete).Methods("DELETE")

	// Asset management
	r.HandleFunc("/api/v1/assets", api.AssetsListSchematics).Methods("GET")
	r.HandleFunc("/api/v1/assets/{schematicId}", api.AssetsGetSchematic).Methods("GET")
	r.HandleFunc("/api/v1/assets/{schematicId}/download", api.AssetsDownload).Methods("POST")
	r.HandleFunc("/api/v1/assets/{schematicId}/pxe/{assetType}", api.AssetsServePXE).Methods("GET")
	r.HandleFunc("/api/v1/assets/{schematicId}/status", api.AssetsGetStatus).Methods("GET")
	r.HandleFunc("/api/v1/assets/{schematicId}", api.AssetsDeleteSchematic).Methods("DELETE")

	// Instance-schematic relationship
	r.HandleFunc("/api/v1/instances/{name}/schematic", api.SchematicGetInstanceSchematic).Methods("GET")
	r.HandleFunc("/api/v1/instances/{name}/schematic", api.SchematicUpdateInstanceSchematic).Methods("PUT")

	// Operations
	r.HandleFunc("/api/v1/instances/{name}/operations", api.OperationList).Methods("GET")
	r.HandleFunc("/api/v1/operations/{id}", api.OperationGet).Methods("GET")
	r.HandleFunc("/api/v1/operations/{id}/stream", api.OperationStream).Methods("GET")
	r.HandleFunc("/api/v1/operations/{id}/cancel", api.OperationCancel).Methods("POST")

	// Cluster operations
	r.HandleFunc("/api/v1/instances/{name}/cluster/config/generate", api.ClusterGenerateConfig).Methods("POST")
	r.HandleFunc("/api/v1/instances/{name}/cluster/bootstrap", api.ClusterBootstrap).Methods("POST")
	r.HandleFunc("/api/v1/instances/{name}/cluster/endpoints", api.ClusterConfigureEndpoints).Methods("POST")
	r.HandleFunc("/api/v1/instances/{name}/cluster/status", api.ClusterGetStatus).Methods("GET")
	r.HandleFunc("/api/v1/instances/{name}/cluster/health", api.ClusterHealth).Methods("GET")
	r.HandleFunc("/api/v1/instances/{name}/cluster/kubeconfig", api.ClusterGetKubeconfig).Methods("GET")
	r.HandleFunc("/api/v1/instances/{name}/cluster/kubeconfig/generate", api.ClusterGenerateKubeconfig).Methods("POST")
	r.HandleFunc("/api/v1/instances/{name}/cluster/talosconfig", api.ClusterGetTalosconfig).Methods("GET")
	r.HandleFunc("/api/v1/instances/{name}/cluster/reset", api.ClusterReset).Methods("POST")

	// Services
	r.HandleFunc("/api/v1/instances/{name}/services", api.ServicesList).Methods("GET")
	r.HandleFunc("/api/v1/instances/{name}/services", api.ServicesInstall).Methods("POST")
	r.HandleFunc("/api/v1/instances/{name}/services/install-all", api.ServicesInstallAll).Methods("POST")
	r.HandleFunc("/api/v1/instances/{name}/services/{service}", api.ServicesGet).Methods("GET")
	r.HandleFunc("/api/v1/instances/{name}/services/{service}", api.ServicesDelete).Methods("DELETE")
	r.HandleFunc("/api/v1/instances/{name}/services/{service}/status", api.ServicesGetStatus).Methods("GET")
	r.HandleFunc("/api/v1/services/{service}/manifest", api.ServicesGetManifest).Methods("GET")
	r.HandleFunc("/api/v1/services/{service}/config", api.ServicesGetConfig).Methods("GET")
	r.HandleFunc("/api/v1/instances/{name}/services/{service}/config", api.ServicesGetInstanceConfig).Methods("GET")

	// Service lifecycle endpoints
	r.HandleFunc("/api/v1/instances/{name}/services/{service}/fetch", api.ServicesFetch).Methods("POST")
	r.HandleFunc("/api/v1/instances/{name}/services/{service}/compile", api.ServicesCompile).Methods("POST")
	r.HandleFunc("/api/v1/instances/{name}/services/{service}/deploy", api.ServicesDeploy).Methods("POST")

	// Apps
	r.HandleFunc("/api/v1/apps", api.AppsListAvailable).Methods("GET")
	r.HandleFunc("/api/v1/apps/{app}", api.AppsGetAvailable).Methods("GET")
	r.HandleFunc("/api/v1/instances/{name}/apps", api.AppsListDeployed).Methods("GET")
	r.HandleFunc("/api/v1/instances/{name}/apps", api.AppsAdd).Methods("POST")
	r.HandleFunc("/api/v1/instances/{name}/apps/{app}/deploy", api.AppsDeploy).Methods("POST")
	r.HandleFunc("/api/v1/instances/{name}/apps/{app}", api.AppsDelete).Methods("DELETE")
	r.HandleFunc("/api/v1/instances/{name}/apps/{app}/status", api.AppsGetStatus).Methods("GET")

	// Backup & Restore
	r.HandleFunc("/api/v1/instances/{name}/apps/{app}/backup", api.BackupAppStart).Methods("POST")
	r.HandleFunc("/api/v1/instances/{name}/apps/{app}/backup", api.BackupAppList).Methods("GET")
	r.HandleFunc("/api/v1/instances/{name}/apps/{app}/restore", api.BackupAppRestore).Methods("POST")

	// Utilities
	r.HandleFunc("/api/v1/utilities/health", api.UtilitiesHealth).Methods("GET")
	r.HandleFunc("/api/v1/instances/{name}/utilities/health", api.InstanceUtilitiesHealth).Methods("GET")
	r.HandleFunc("/api/v1/utilities/dashboard/token", api.UtilitiesDashboardToken).Methods("GET")
	r.HandleFunc("/api/v1/utilities/nodes/ips", api.UtilitiesNodeIPs).Methods("GET")
	r.HandleFunc("/api/v1/utilities/controlplane/ip", api.UtilitiesControlPlaneIP).Methods("GET")
	r.HandleFunc("/api/v1/utilities/secrets/{secret}/copy", api.UtilitiesSecretCopy).Methods("POST")
	r.HandleFunc("/api/v1/utilities/version", api.UtilitiesVersion).Methods("GET")

	// dnsmasq management
	r.HandleFunc("/api/v1/dnsmasq/status", api.DnsmasqStatus).Methods("GET")
	r.HandleFunc("/api/v1/dnsmasq/config", api.DnsmasqGetConfig).Methods("GET")
	r.HandleFunc("/api/v1/dnsmasq/restart", api.DnsmasqRestart).Methods("POST")
	r.HandleFunc("/api/v1/dnsmasq/generate", api.DnsmasqGenerate).Methods("POST")
	r.HandleFunc("/api/v1/dnsmasq/update", api.DnsmasqUpdate).Methods("POST")
}

// CreateInstance creates a new instance
func (api *API) CreateInstance(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "Instance name is required")
		return
	}

	if err := api.instance.CreateInstance(req.Name); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create instance: %v", err))
		return
	}

	// Attempt to update dnsmasq configuration with all instances
	// This is non-critical - include warning in response if it fails
	response := map[string]interface{}{
		"name":    req.Name,
		"message": "Instance created successfully",
	}

	if err := api.updateDnsmasqForAllInstances(); err != nil {
		log.Printf("Warning: Could not update dnsmasq configuration: %v", err)
		response["warning"] = fmt.Sprintf("dnsmasq update failed: %v. Use POST /api/v1/dnsmasq/update to retry.", err)
	}

	respondJSON(w, http.StatusCreated, response)
}

// ListInstances lists all instances
func (api *API) ListInstances(w http.ResponseWriter, r *http.Request) {
	instances, err := api.instance.ListInstances()
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list instances: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"instances": instances,
	})
}

// GetInstance retrieves instance details
func (api *API) GetInstance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	if err := api.instance.ValidateInstance(name); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	// Get config
	configPath := api.instance.GetInstanceConfigPath(name)
	configData, err := os.ReadFile(configPath)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to read config: %v", err))
		return
	}

	var configMap map[string]interface{}
	if err := yaml.Unmarshal(configData, &configMap); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to parse config: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"name":   name,
		"config": configMap,
	})
}

// DeleteInstance deletes an instance
func (api *API) DeleteInstance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	if err := api.instance.DeleteInstance(name); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete instance: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Instance deleted successfully",
	})
}

// GetConfig retrieves instance configuration
func (api *API) GetConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	if err := api.instance.ValidateInstance(name); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	configPath := api.instance.GetInstanceConfigPath(name)
	configData, err := os.ReadFile(configPath)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to read config: %v", err))
		return
	}

	var configMap map[string]interface{}
	if err := yaml.Unmarshal(configData, &configMap); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to parse config: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, configMap)
}

// UpdateConfig updates instance configuration
func (api *API) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	if err := api.instance.ValidateInstance(name); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	var updates map[string]interface{}
	if err := yaml.Unmarshal(body, &updates); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid YAML: %v", err))
		return
	}

	configPath := api.instance.GetInstanceConfigPath(name)

	// Update each key-value pair
	for key, value := range updates {
		valueStr := fmt.Sprintf("%v", value)
		if err := api.config.SetConfigValue(configPath, key, valueStr); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update config key %s: %v", key, err))
			return
		}
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Config updated successfully",
	})
}

// GetSecrets retrieves instance secrets (redacted by default)
func (api *API) GetSecrets(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	if err := api.instance.ValidateInstance(name); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	secretsPath := api.instance.GetInstanceSecretsPath(name)

	secretsData, err := os.ReadFile(secretsPath)
	if err != nil {
		if os.IsNotExist(err) {
			respondJSON(w, http.StatusOK, map[string]interface{}{})
			return
		}
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to read secrets: %v", err))
		return
	}

	var secretsMap map[string]interface{}
	if err := yaml.Unmarshal(secretsData, &secretsMap); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to parse secrets: %v", err))
		return
	}

	// Check if client wants raw secrets (dangerous!)
	showRaw := r.URL.Query().Get("raw") == "true"

	if !showRaw {
		// Redact secrets
		for key := range secretsMap {
			secretsMap[key] = "********"
		}
	}

	respondJSON(w, http.StatusOK, secretsMap)
}

// UpdateSecrets updates instance secrets
func (api *API) UpdateSecrets(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	if err := api.instance.ValidateInstance(name); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	var updates map[string]interface{}
	if err := yaml.Unmarshal(body, &updates); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid YAML: %v", err))
		return
	}

	// Get secrets file path
	secretsPath := api.instance.GetInstanceSecretsPath(name)

	// Update each secret
	for key, value := range updates {
		valueStr := fmt.Sprintf("%v", value)
		if err := api.secrets.SetSecret(secretsPath, key, valueStr); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update secret %s: %v", key, err))
			return
		}
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Secrets updated successfully",
	})
}

// GetContext retrieves current context
func (api *API) GetContext(w http.ResponseWriter, r *http.Request) {
	currentContext, err := api.context.GetCurrentContext()
	if err != nil {
		if os.IsNotExist(err) {
			respondJSON(w, http.StatusOK, map[string]interface{}{
				"context": nil,
			})
			return
		}
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get context: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"context": currentContext,
	})
}

// SetContext sets current context
func (api *API) SetContext(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Context string `json:"context"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Context == "" {
		respondError(w, http.StatusBadRequest, "Context name is required")
		return
	}

	if err := api.context.SetCurrentContext(req.Context); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("Failed to set context: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"context": req.Context,
		"message": "Context set successfully",
	})
}

// StatusHandler returns daemon status information
func (api *API) StatusHandler(w http.ResponseWriter, r *http.Request, startTime time.Time, dataDir, appsDir string) {
	// Get list of instances
	instances, err := api.instance.ListInstances()
	if err != nil {
		instances = []string{}
	}

	// Calculate uptime
	uptime := time.Since(startTime)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":        "running",
		"version":       "0.1.0", // TODO: Get from build info
		"uptime":        uptime.String(),
		"uptimeSeconds": int(uptime.Seconds()),
		"dataDir":       dataDir,
		"appsDir":       appsDir,
		"setupFiles":    "embedded", // Indicate that setup files are now embedded
		"instances": map[string]interface{}{
			"count": len(instances),
			"names": instances,
		},
	})
}

// Helper functions

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{
		"error": message,
	})
}
