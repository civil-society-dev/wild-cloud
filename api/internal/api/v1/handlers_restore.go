package v1

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/wild-cloud/wild-central/daemon/internal/backup"
	"github.com/wild-cloud/wild-central/daemon/internal/operations"
)

// Note: RestoreFromSnapshot uses custom async logic (needs snapshotID in response)

// ListSnapshots lists all snapshots for an instance
func (api *API) ListSnapshots(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)

	snapshots, err := backup.ListSnapshotsForInstance(api.dataDir, instanceName)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list snapshots: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"snapshots": snapshots,
		},
	})
}

// ListAppSnapshots lists all snapshots for a specific app
func (api *API) ListAppSnapshots(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	appName := GetAppName(r)

	snapshots, err := backup.ListSnapshotsForApp(api.dataDir, instanceName, appName)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list app snapshots: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"snapshots": snapshots,
		},
	})
}

// RestoreFromSnapshot restores an app from a restic snapshot
func (api *API) RestoreFromSnapshot(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	appName := GetAppName(r)
	snapshotID := mux.Vars(r)["snapshot"]

	// Parse request body for restore options
	var opts backup.RestoreOptions
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		// Use defaults if no body provided
		opts = backup.RestoreOptions{}
	}

	mgr := backup.NewManager(api.dataDir)

	// Custom async handling needed to include snapshot_id in response
	opMgr := operations.NewManager(api.dataDir)
	opID, err := opMgr.Start(instanceName, "restore-snapshot", appName)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to start restore operation: "+err.Error())
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				_ = opMgr.Update(instanceName, opID, "failed", "Internal error during restore", 100)
			}
		}()

		_ = opMgr.UpdateProgress(instanceName, opID, 10, "Validating snapshot")

		if err := mgr.RestoreFromSnapshot(instanceName, appName, snapshotID, opts); err != nil {
			_ = opMgr.Update(instanceName, opID, "failed", err.Error(), 100)
			return
		}

		_ = opMgr.Update(instanceName, opID, "completed", "Restore from snapshot completed", 100)
	}()

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"success":      true,
		"operation_id": opID,
		"message":      "Restore from snapshot started",
		"snapshot_id":  snapshotID,
	})
}
