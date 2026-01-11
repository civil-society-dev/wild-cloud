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
	instanceName := GetInstanceName(r)
	appName := GetAppName(r)

	api.StartAsyncOperation(w, instanceName, "backup", appName,
		func(opsMgr *operations.Manager, opID string) error {
			_ = opsMgr.UpdateProgress(instanceName, opID, 10, "Starting backup")
			mgr := backup.NewManager(api.dataDir)
			_, err := mgr.BackupApp(instanceName, appName)
			return err
		})
}

// BackupAppList lists all backups for an app
func (api *API) BackupAppList(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	appName := GetAppName(r)

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
	instanceName := GetInstanceName(r)
	appName := GetAppName(r)

	var opts backup.RestoreOptions
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		opts = backup.RestoreOptions{}
	}

	api.StartAsyncOperation(w, instanceName, "restore", appName,
		func(opsMgr *operations.Manager, opID string) error {
			_ = opsMgr.UpdateProgress(instanceName, opID, 10, "Starting restore")
			mgr := backup.NewManager(api.dataDir)
			return mgr.RestoreApp(instanceName, appName, opts)
		})
}

// BackupAppDelete deletes a specific app backup
func (api *API) BackupAppDelete(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	appName := GetAppName(r)
	timestamp := mux.Vars(r)["timestamp"]

	mgr := backup.NewManager(api.dataDir)
	if err := mgr.DeleteAppBackup(instanceName, appName, timestamp); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete backup")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Backup deleted successfully",
	})
}
