package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/wild-cloud/wild-central/daemon/internal/discovery"
	"github.com/wild-cloud/wild-central/daemon/internal/node"
)

// NodeDiscover initiates node discovery
// Accepts optional subnet parameter. If no subnet provided, auto-detects local networks.
func (api *API) NodeDiscover(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)

	var req SubnetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	var ipList []string
	var err error

	if req.Subnet != "" {
		ipList, err = discovery.ExpandSubnet(req.Subnet)
		if err != nil {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid subnet: %v", err))
			return
		}
	} else {
		networks, err := discovery.GetLocalNetworks()
		if err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to detect local networks: %v", err))
			return
		}

		if len(networks) == 0 {
			respondError(w, http.StatusNotFound, "No local networks found")
			return
		}

		for _, network := range networks {
			ips, err := discovery.ExpandSubnet(network)
			if err != nil {
				continue
			}
			ipList = append(ipList, ips...)
		}
	}

	discoveryMgr := discovery.NewManager(api.dataDir, instanceName)
	if err := discoveryMgr.StartDiscovery(instanceName, ipList); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to start discovery: %v", err))
		return
	}

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"message":     "Discovery started",
		"status":      "running",
		"ips_to_scan": len(ipList),
	})
}

// NodeDiscoveryStatus returns discovery status
func (api *API) NodeDiscoveryStatus(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)

	discoveryMgr := discovery.NewManager(api.dataDir, instanceName)
	status, err := discoveryMgr.GetDiscoveryStatus(instanceName)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get status: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, status)
}

// NodeHardware returns hardware info for a specific node
func (api *API) NodeHardware(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	nodeIP := mux.Vars(r)["ip"]

	nodeMgr := node.NewManager(api.dataDir, instanceName)
	hwInfo, err := nodeMgr.DetectHardware(nodeIP)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to detect hardware: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, hwInfo)
}

// NodeDetect detects hardware on a single node (POST with IP in body)
// IP address is required.
func (api *API) NodeDetect(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)

	var req IPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.IP == "" {
		respondError(w, http.StatusBadRequest, "IP address is required")
		return
	}

	nodeMgr := node.NewManager(api.dataDir, instanceName)
	hwInfo, err := nodeMgr.DetectHardware(req.IP)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to detect hardware: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, hwInfo)
}

// NodeAdd registers a new node
func (api *API) NodeAdd(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)

	var nodeData node.Node
	if err := json.NewDecoder(r.Body).Decode(&nodeData); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	nodeMgr := node.NewManager(api.dataDir, instanceName)
	if err := nodeMgr.Add(instanceName, &nodeData); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to add node: %v", err))
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "Node added successfully",
		"node":    nodeData,
	})
}

// NodeList returns all nodes for an instance
func (api *API) NodeList(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)

	nodeMgr := node.NewManager(api.dataDir, instanceName)
	nodes, err := nodeMgr.List(instanceName)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list nodes: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"nodes": nodes,
	})
}

// NodeGet returns a specific node
func (api *API) NodeGet(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	nodeIdentifier := GetNodeName(r)

	nodeMgr := node.NewManager(api.dataDir, instanceName)
	nodeData, err := nodeMgr.Get(instanceName, nodeIdentifier)
	if err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Node not found: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, nodeData)
}

// NodeApply generates configuration and applies it to node
func (api *API) NodeApply(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	nodeIdentifier := GetNodeName(r)

	opts := node.ApplyOptions{}

	nodeMgr := node.NewManager(api.dataDir, instanceName)
	if err := nodeMgr.Apply(instanceName, nodeIdentifier, opts); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to apply node configuration: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Node configuration applied successfully",
		"node":    nodeIdentifier,
	})
}

// NodeUpdate modifies existing node configuration
func (api *API) NodeUpdate(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	nodeIdentifier := GetNodeName(r)

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	nodeMgr := node.NewManager(api.dataDir, instanceName)
	if err := nodeMgr.Update(instanceName, nodeIdentifier, updates); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update node: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Node updated successfully",
		"node":    nodeIdentifier,
	})
}

// NodeFetchTemplates copies patch templates from directory to instance
func (api *API) NodeFetchTemplates(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)

	nodeMgr := node.NewManager(api.dataDir, instanceName)
	if err := nodeMgr.FetchTemplates(instanceName); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch templates: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Templates fetched successfully",
	})
}

// NodeDelete removes a node
// Query parameter: skip_reset=true to force delete without resetting
func (api *API) NodeDelete(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	nodeIdentifier := GetNodeName(r)

	skipReset := r.URL.Query().Get("skip_reset") == "true"

	nodeMgr := node.NewManager(api.dataDir, instanceName)
	if err := nodeMgr.Delete(instanceName, nodeIdentifier, skipReset); err != nil {
		errMsg := err.Error()
		if !skipReset && (strings.Contains(errMsg, "reset") || strings.Contains(errMsg, "timed out")) {
			respondError(w, http.StatusConflict, errMsg)
			return
		}
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete node: %v", err))
		return
	}

	message := "Node deleted successfully"
	if !skipReset {
		message = "Node reset and removed successfully"
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": message,
	})
}

// NodeDiscoveryCancel cancels an in-progress discovery operation
func (api *API) NodeDiscoveryCancel(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)

	discoveryMgr := discovery.NewManager(api.dataDir, instanceName)
	if err := discoveryMgr.CancelDiscovery(instanceName); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("Failed to cancel discovery: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Discovery cancelled successfully",
	})
}

// NodeReset resets a node to maintenance mode
func (api *API) NodeReset(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	nodeIdentifier := GetNodeName(r)

	nodeMgr := node.NewManager(api.dataDir, instanceName)
	if err := nodeMgr.Reset(instanceName, nodeIdentifier); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to reset node: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Node reset successfully - now in maintenance mode",
		"node":    nodeIdentifier,
	})
}
