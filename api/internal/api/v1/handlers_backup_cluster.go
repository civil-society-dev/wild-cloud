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
	instanceName := GetInstanceName(r)

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

	api.StartAsyncOperation(w, instanceName, "cluster-backup", "cluster",
		func(opsMgr *operations.Manager, opID string) error {
			_ = opsMgr.UpdateProgress(instanceName, opID, 10, "Starting cluster backup")
			mgr := backup.NewManager(api.dataDir)
			_, err := mgr.BackupCluster(instanceName, components)
			return err
		})
}

// BackupClusterList lists all cluster backups for an instance
func (api *API) BackupClusterList(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)

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
	instanceName := GetInstanceName(r)

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

	api.StartAsyncOperation(w, instanceName, "cluster-restore", "cluster",
		func(opsMgr *operations.Manager, opID string) error {
			_ = opsMgr.UpdateProgress(instanceName, opID, 10, "Starting cluster restore")
			mgr := backup.NewManager(api.dataDir)
			return mgr.RestoreCluster(instanceName, req.Timestamp, components)
		})
}

// BackupClusterDelete deletes a specific cluster backup
func (api *API) BackupClusterDelete(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	timestamp := mux.Vars(r)["timestamp"]

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
	instanceName := GetInstanceName(r)

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
