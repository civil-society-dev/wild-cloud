package v1

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"

	"github.com/wild-cloud/wild-central/daemon/internal/operations"
)

// OperationGet returns operation status
func (api *API) OperationGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	opID := vars["id"]

	// Extract instance name from query param or header
	instanceName := r.URL.Query().Get("instance")
	if instanceName == "" {
		respondError(w, http.StatusBadRequest, "instance parameter is required")
		return
	}

	// Get operation
	opsMgr := operations.NewManager(api.dataDir)
	op, err := opsMgr.GetByInstance(instanceName, opID)
	if err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Operation not found: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, op)
}

// OperationList returns all operations for an instance
func (api *API) OperationList(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]

	// Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	// List operations
	opsMgr := operations.NewManager(api.dataDir)
	ops, err := opsMgr.List(instanceName)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list operations: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"operations": ops,
	})
}

// OperationCancel cancels an operation
func (api *API) OperationCancel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	opID := vars["id"]

	// Extract instance name from query param
	instanceName := r.URL.Query().Get("instance")
	if instanceName == "" {
		respondError(w, http.StatusBadRequest, "instance parameter is required")
		return
	}

	// Cancel operation
	opsMgr := operations.NewManager(api.dataDir)
	if err := opsMgr.Cancel(instanceName, opID); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to cancel operation: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Operation cancelled",
		"id":      opID,
	})
}

// OperationStream streams operation output via Server-Sent Events (SSE)
func (api *API) OperationStream(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	opID := vars["id"]

	// Extract instance name from query param
	instanceName := r.URL.Query().Get("instance")
	if instanceName == "" {
		respondError(w, http.StatusBadRequest, "instance parameter is required")
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		respondError(w, http.StatusInternalServerError, "Streaming not supported")
		return
	}

	// Check if operation is already completed
	statusFile := filepath.Join(api.dataDir, "instances", instanceName, "operations", opID+".json")
	isCompleted := false
	if data, err := os.ReadFile(statusFile); err == nil {
		var op map[string]interface{}
		if err := json.Unmarshal(data, &op); err == nil {
			if status, ok := op["status"].(string); ok {
				isCompleted = (status == "completed" || status == "failed")
			}
		}
	}

	// Send existing log file content first (if exists)
	logPath := filepath.Join(api.dataDir, "instances", instanceName, "operations", opID, "output.log")
	if _, err := os.Stat(logPath); err == nil {
		file, err := os.Open(logPath)
		if err == nil {
			defer file.Close()
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				fmt.Fprintf(w, "data: %s\n\n", line)
				flusher.Flush()
			}
		}
	}

	// If operation is already completed, send completion signal and return
	if isCompleted {
		// Send an event to signal completion
		fmt.Fprintf(w, "event: complete\ndata: Operation completed\n\n")
		flusher.Flush()
		return
	}

	// Subscribe to new output for ongoing operations
	ch := api.broadcaster.Subscribe(opID)
	defer api.broadcaster.Unsubscribe(opID, ch)

	// Stream new output as it arrives
	for data := range ch {
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
}
