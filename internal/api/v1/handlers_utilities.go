package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/wild-cloud/wild-central/daemon/internal/utilities"
)

// UtilitiesHealth returns cluster health status (legacy, no instance context)
func (api *API) UtilitiesHealth(w http.ResponseWriter, r *http.Request) {
	status, err := utilities.GetClusterHealth("")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get cluster health")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    status,
	})
}

// InstanceUtilitiesHealth returns cluster health status for a specific instance
func (api *API) InstanceUtilitiesHealth(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]

	// Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	// Get kubeconfig path for this instance
	kubeconfigPath := filepath.Join(api.dataDir, "instances", instanceName, "kubeconfig")

	status, err := utilities.GetClusterHealth(kubeconfigPath)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get cluster health")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    status,
	})
}

// UtilitiesDashboardToken returns a Kubernetes dashboard token
func (api *API) UtilitiesDashboardToken(w http.ResponseWriter, r *http.Request) {
	token, err := utilities.GetDashboardToken()
	if err != nil {
		// Try fallback method
		token, err = utilities.GetDashboardTokenFromSecret()
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to get dashboard token")
			return
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    token,
	})
}

// UtilitiesNodeIPs returns IP addresses for all cluster nodes
func (api *API) UtilitiesNodeIPs(w http.ResponseWriter, r *http.Request) {
	nodes, err := utilities.GetNodeIPs()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get node IPs")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"nodes": nodes,
		},
	})
}

// UtilitiesControlPlaneIP returns the control plane IP
func (api *API) UtilitiesControlPlaneIP(w http.ResponseWriter, r *http.Request) {
	ip, err := utilities.GetControlPlaneIP()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get control plane IP")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"ip": ip,
		},
	})
}

// UtilitiesSecretCopy copies a secret between namespaces
func (api *API) UtilitiesSecretCopy(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	secretName := vars["secret"]

	var req struct {
		SourceNamespace      string `json:"source_namespace"`
		DestinationNamespace string `json:"destination_namespace"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.SourceNamespace == "" || req.DestinationNamespace == "" {
		respondError(w, http.StatusBadRequest, "source_namespace and destination_namespace are required")
		return
	}

	if err := utilities.CopySecretBetweenNamespaces(secretName, req.SourceNamespace, req.DestinationNamespace); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to copy secret")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Secret copied successfully",
	})
}

// UtilitiesVersion returns cluster and Talos versions
func (api *API) UtilitiesVersion(w http.ResponseWriter, r *http.Request) {
	k8sVersion, err := utilities.GetClusterVersion()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get cluster version")
		return
	}

	talosVersion, _ := utilities.GetTalosVersion() // Don't fail if Talos check fails

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"kubernetes": k8sVersion,
			"talos":      talosVersion,
		},
	})
}
