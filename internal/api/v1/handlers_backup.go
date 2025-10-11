package v1

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/wild-cloud/wild-central/daemon/internal/backup"
	"github.com/wild-cloud/wild-central/daemon/internal/operations"
)

// BackupAppStart starts a backup operation for an app
func (api *API) BackupAppStart(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]
	appName := vars["app"]

	mgr := backup.NewManager(api.dataDir)

	// Create operation for tracking
	opMgr := operations.NewManager(api.dataDir)
	opID, err := opMgr.Start(instanceName, "backup", appName)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to start backup operation")
		return
	}

	// Run backup in background
	go func() {
		opMgr.UpdateProgress(instanceName, opID, 10, "Starting backup")

		info, err := mgr.BackupApp(instanceName, appName)
		if err != nil {
			opMgr.Update(instanceName, opID, "failed", err.Error(), 100)
			return
		}

		opMgr.Update(instanceName, opID, "completed", "Backup completed", 100)
		_ = info // Metadata saved in backup.json
	}()

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"success":      true,
		"operation_id": opID,
		"message":      "Backup started",
	})
}

// BackupAppList lists all backups for an app
func (api *API) BackupAppList(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]
	appName := vars["app"]

	mgr := backup.NewManager(api.dataDir)
	backups, err := mgr.ListBackups(instanceName, appName)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list backups")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"backups": backups,
		},
	})
}

// BackupAppRestore restores an app from backup
func (api *API) BackupAppRestore(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]
	appName := vars["app"]

	// Parse request body for restore options
	var opts backup.RestoreOptions
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		// Use defaults if no body provided
		opts = backup.RestoreOptions{}
	}

	mgr := backup.NewManager(api.dataDir)

	// Create operation for tracking
	opMgr := operations.NewManager(api.dataDir)
	opID, err := opMgr.Start(instanceName, "restore", appName)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to start restore operation")
		return
	}

	// Run restore in background
	go func() {
		opMgr.UpdateProgress(instanceName, opID, 10, "Starting restore")

		if err := mgr.RestoreApp(instanceName, appName, opts); err != nil {
			opMgr.Update(instanceName, opID, "failed", err.Error(), 100)
			return
		}

		opMgr.Update(instanceName, opID, "completed", "Restore completed", 100)
	}()

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"success":      true,
		"operation_id": opID,
		"message":      "Restore started",
	})
}
