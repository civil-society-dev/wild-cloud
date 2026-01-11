package v1

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
	"github.com/wild-cloud/wild-central/daemon/internal/utilities"
)

// InstanceUtilitiesHealth returns cluster health status for a specific instance
func (api *API) InstanceUtilitiesHealth(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	kubeconfigPath := tools.GetKubeconfigPath(api.dataDir, instanceName)

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

// InstanceUtilitiesDashboardToken returns a Kubernetes dashboard token for a specific instance
func (api *API) UtilitiesDashboardToken(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	kubeconfigPath := tools.GetKubeconfigPath(api.dataDir, instanceName)

	token, err := utilities.GetDashboardToken(kubeconfigPath)
	if err != nil {
		token, err = utilities.GetDashboardTokenFromSecret(kubeconfigPath)
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
	instanceName := GetInstanceName(r)
	kubeconfigPath := tools.GetKubeconfigPath(api.dataDir, instanceName)

	nodes, err := utilities.GetNodeIPs(kubeconfigPath)
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
	instanceName := GetInstanceName(r)
	kubeconfigPath := tools.GetKubeconfigPath(api.dataDir, instanceName)

	ip, err := utilities.GetControlPlaneIP(kubeconfigPath)
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
	instanceName := GetInstanceName(r)
	secretName := mux.Vars(r)["secret"]

	var req SecretCopyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.SourceNamespace == "" || req.DestinationNamespace == "" {
		respondError(w, http.StatusBadRequest, "source_namespace and destination_namespace are required")
		return
	}

	kubeconfigPath := tools.GetKubeconfigPath(api.dataDir, instanceName)

	if err := utilities.CopySecretBetweenNamespaces(kubeconfigPath, secretName, req.SourceNamespace, req.DestinationNamespace); err != nil {
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
	instanceName := GetInstanceName(r)
	kubeconfigPath := tools.GetKubeconfigPath(api.dataDir, instanceName)

	k8sVersion, err := utilities.GetClusterVersion(kubeconfigPath)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get cluster version")
		return
	}

	talosVersion, _ := utilities.GetTalosVersion()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"kubernetes": k8sVersion,
			"talos":      talosVersion,
		},
	})
}
