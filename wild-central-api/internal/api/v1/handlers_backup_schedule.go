package v1

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/wild-cloud/wild-central/daemon/internal/backup"
)

// ScheduleListHandler lists all backup schedules for an instance
func (api *API) ScheduleListHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]

	mgr := backup.NewManager(api.dataDir)
	schedules, err := mgr.LoadSchedules(instanceName)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to load schedules: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"schedules": schedules,
	})
}

// ScheduleCreateHandler creates a new backup schedule
func (api *API) ScheduleCreateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]

	var req backup.BackupSchedule
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	mgr := backup.NewManager(api.dataDir)

	// Validate configuration if scheduler is available
	if api.backupScheduler != nil {
		if err := api.backupScheduler.ValidateScheduleConfiguration(instanceName, &req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid schedule configuration: "+err.Error())
			return
		}
	}

	schedule, err := mgr.CreateSchedule(instanceName, &req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create schedule: "+err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"schedule": schedule,
	})
}

// ScheduleGetHandler retrieves a specific backup schedule
func (api *API) ScheduleGetHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]
	scheduleID := vars["schedule_id"]

	mgr := backup.NewManager(api.dataDir)
	schedule, err := mgr.GetSchedule(instanceName, scheduleID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Schedule not found: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"schedule": schedule,
	})
}

// ScheduleUpdateHandler updates an existing backup schedule
func (api *API) ScheduleUpdateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]
	scheduleID := vars["schedule_id"]

	var req backup.BackupSchedule
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	mgr := backup.NewManager(api.dataDir)

	// Validate configuration if scheduler is available
	if api.backupScheduler != nil {
		if err := api.backupScheduler.ValidateScheduleConfiguration(instanceName, &req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid schedule configuration: "+err.Error())
			return
		}
	}

	schedule, err := mgr.UpdateSchedule(instanceName, scheduleID, &req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update schedule: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"schedule": schedule,
	})
}

// ScheduleDeleteHandler deletes a backup schedule
func (api *API) ScheduleDeleteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]
	scheduleID := vars["schedule_id"]

	mgr := backup.NewManager(api.dataDir)
	if err := mgr.DeleteSchedule(instanceName, scheduleID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete schedule: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Schedule deleted",
	})
}

// ScheduleRunHandler manually triggers a scheduled backup
func (api *API) ScheduleRunHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]
	scheduleID := vars["schedule_id"]

	if api.backupScheduler == nil {
		respondError(w, http.StatusInternalServerError, "Backup scheduler not available")
		return
	}

	// Execute synchronously for manual trigger
	if err := api.backupScheduler.ExecuteScheduledBackup(instanceName, scheduleID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to execute backup: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Backup executed successfully",
	})
}

// ScheduleHistoryHandler returns backup history for a schedule
func (api *API) ScheduleHistoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]
	scheduleID := vars["schedule_id"]

	if api.backupScheduler == nil {
		respondError(w, http.StatusInternalServerError, "Backup scheduler not available")
		return
	}

	history, err := api.backupScheduler.GetScheduleHistory(instanceName, scheduleID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get history: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"history": history,
	})
}

// SchedulerStatusHandler returns the current scheduler status
func (api *API) SchedulerStatusHandler(w http.ResponseWriter, r *http.Request) {
	if api.backupScheduler == nil {
		respondError(w, http.StatusInternalServerError, "Backup scheduler not available")
		return
	}

	status := api.backupScheduler.GetSchedulerStatus()
	respondJSON(w, http.StatusOK, status)
}
