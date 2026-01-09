package v1

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/wild-cloud/wild-central/daemon/internal/backup"
	"github.com/wild-cloud/wild-central/daemon/internal/operations"
)

// BackupClusterStart starts a cluster backup operation
func (api *API) BackupClusterStart(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]

	// Parse request body for backup components
	var req struct {
		Etcd    bool `json:"etcd"`
		Config  bool `json:"config"`
		Secrets bool `json:"secrets"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	components := backup.ClusterBackupComponents{
		Etcd:    req.Etcd,
		Config:  req.Config,
		Secrets: req.Secrets,
	}

	mgr := backup.NewManager(api.dataDir)

	// Create operation for tracking
	opMgr := operations.NewManager(api.dataDir)
	opID, err := opMgr.Start(instanceName, "cluster-backup", "cluster")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to start backup operation")
		return
	}

	// Run backup in background
	go func() {
		_ = opMgr.UpdateProgress(instanceName, opID, 10, "Starting cluster backup")

		info, err := mgr.BackupCluster(instanceName, components)
		if err != nil {
			_ = opMgr.Update(instanceName, opID, "failed", err.Error(), 100)
			return
		}

		_ = opMgr.Update(instanceName, opID, "completed", "Cluster backup completed", 100)
		_ = info // Metadata saved in cluster-backup.json
	}()

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"success":      true,
		"operation_id": opID,
		"message":      "Cluster backup started",
	})
}

// BackupClusterList lists all cluster backups for an instance
func (api *API) BackupClusterList(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]

	mgr := backup.NewManager(api.dataDir)
	backups, err := mgr.ListClusterBackups(instanceName)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list cluster backups")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"backups": backups,
		},
	})
}

// BackupClusterRestore restores cluster from backup
func (api *API) BackupClusterRestore(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]

	// Parse request body for restore options
	var req struct {
		Timestamp string `json:"timestamp"`
		Etcd      bool   `json:"etcd"`
		Config    bool   `json:"config"`
		Secrets   bool   `json:"secrets"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Timestamp == "" {
		respondError(w, http.StatusBadRequest, "Timestamp is required")
		return
	}

	components := backup.ClusterBackupComponents{
		Etcd:    req.Etcd,
		Config:  req.Config,
		Secrets: req.Secrets,
	}

	mgr := backup.NewManager(api.dataDir)

	// Create operation for tracking
	opMgr := operations.NewManager(api.dataDir)
	opID, err := opMgr.Start(instanceName, "cluster-restore", "cluster")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to start restore operation")
		return
	}

	// Run restore in background
	go func() {
		_ = opMgr.UpdateProgress(instanceName, opID, 10, "Starting cluster restore")

		if err := mgr.RestoreCluster(instanceName, req.Timestamp, components); err != nil {
			_ = opMgr.Update(instanceName, opID, "failed", err.Error(), 100)
			return
		}

		_ = opMgr.Update(instanceName, opID, "completed", "Cluster restore completed", 100)
	}()

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"success":      true,
		"operation_id": opID,
		"message":      "Cluster restore started",
	})
}

// BackupClusterDelete deletes a specific cluster backup
func (api *API) BackupClusterDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]
	timestamp := vars["timestamp"]

	mgr := backup.NewManager(api.dataDir)
	if err := mgr.DeleteClusterBackup(instanceName, timestamp); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete cluster backup")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Cluster backup deleted",
	})
}

// BackupListAll lists all backups (cluster and apps) for an instance
func (api *API) BackupListAll(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]

	mgr := backup.NewManager(api.dataDir)
	allBackups, err := mgr.ListAllBackups(instanceName)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list backups")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    allBackups,
	})
}
