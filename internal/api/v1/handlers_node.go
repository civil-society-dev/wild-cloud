package v1

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/wild-cloud/wild-central/daemon/internal/discovery"
	"github.com/wild-cloud/wild-central/daemon/internal/node"
)

// NodeDiscover initiates node discovery
func (api *API) NodeDiscover(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]

	// Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	// Parse request body
	var req struct {
		IPList []string `json:"ip_list"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.IPList) == 0 {
		respondError(w, http.StatusBadRequest, "ip_list is required")
		return
	}

	// Start discovery
	discoveryMgr := discovery.NewManager(api.dataDir, instanceName)
	if err := discoveryMgr.StartDiscovery(instanceName, req.IPList); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to start discovery: %v", err))
		return
	}

	respondJSON(w, http.StatusAccepted, map[string]string{
		"message": "Discovery started",
		"status":  "running",
	})
}

// NodeDiscoveryStatus returns discovery status
func (api *API) NodeDiscoveryStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]

	// Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

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
	vars := mux.Vars(r)
	instanceName := vars["name"]
	nodeIP := vars["ip"]

	// Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	// Detect hardware
	nodeMgr := node.NewManager(api.dataDir)
	hwInfo, err := nodeMgr.DetectHardware(nodeIP)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to detect hardware: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, hwInfo)
}

// NodeDetect detects hardware on a single node (POST with IP in body)
func (api *API) NodeDetect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]

	// Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	// Parse request body
	var req struct {
		IP string `json:"ip"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.IP == "" {
		respondError(w, http.StatusBadRequest, "ip is required")
		return
	}

	// Detect hardware
	nodeMgr := node.NewManager(api.dataDir)
	hwInfo, err := nodeMgr.DetectHardware(req.IP)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to detect hardware: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, hwInfo)
}

// NodeAdd registers a new node
func (api *API) NodeAdd(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]

	// Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	// Parse node data
	var nodeData node.Node
	if err := json.NewDecoder(r.Body).Decode(&nodeData); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Add node
	nodeMgr := node.NewManager(api.dataDir)
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
	vars := mux.Vars(r)
	instanceName := vars["name"]

	// Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	// List nodes
	nodeMgr := node.NewManager(api.dataDir)
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
	vars := mux.Vars(r)
	instanceName := vars["name"]
	nodeIdentifier := vars["node"]

	// Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	// Get node
	nodeMgr := node.NewManager(api.dataDir)
	nodeData, err := nodeMgr.Get(instanceName, nodeIdentifier)
	if err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Node not found: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, nodeData)
}

// NodeApply generates configuration and applies it to node
func (api *API) NodeApply(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]
	nodeIdentifier := vars["node"]

	// Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	// Apply always uses default options (no body needed)
	opts := node.ApplyOptions{}

	// Apply node configuration
	nodeMgr := node.NewManager(api.dataDir)
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
	vars := mux.Vars(r)
	instanceName := vars["name"]
	nodeIdentifier := vars["node"]

	// Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	// Parse update data
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Update node
	nodeMgr := node.NewManager(api.dataDir)
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
	vars := mux.Vars(r)
	instanceName := vars["name"]

	// Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	// Fetch templates
	nodeMgr := node.NewManager(api.dataDir)
	if err := nodeMgr.FetchTemplates(instanceName); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch templates: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Templates fetched successfully",
	})
}

// NodeDelete removes a node
func (api *API) NodeDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]
	nodeIdentifier := vars["node"]

	// Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	// Delete node
	nodeMgr := node.NewManager(api.dataDir)
	if err := nodeMgr.Delete(instanceName, nodeIdentifier); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete node: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Node deleted successfully",
	})
}
