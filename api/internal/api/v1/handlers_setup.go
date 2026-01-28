package v1

import (
	"net/http"

	"github.com/wild-cloud/wild-central/daemon/internal/setup"
)

// GetSetupStatus returns the current setup status for an instance
func (api *API) GetSetupStatus(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)

	// Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, "Instance not found")
		return
	}

	// Detect setup status
	status, err := setup.DetectSetupStatus(instanceName, api.dataDir)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, status)
}
