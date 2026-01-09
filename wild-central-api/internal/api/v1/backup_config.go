package v1

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/wild-cloud/wild-central/daemon/internal/backup"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
)

type BackupConfigRequest struct {
	Repository string                          `json:"repository"`
	Staging    string                          `json:"staging,omitempty"`
	Retention  *backup.ResticRetentionPolicy   `json:"retention,omitempty"`
	Backend    *backup.BackendConfig           `json:"backend,omitempty"`
	Secrets    *BackupSecretsRequest           `json:"secrets,omitempty"`
}

type BackupSecretsRequest struct {
	Password    string                    `json:"password"`
	Credentials *backup.BackupCredentials `json:"credentials,omitempty"`
}

type BackupConfigResponse struct {
	Repository string                       `json:"repository"`
	Staging    string                       `json:"staging"`
	Retention  backup.ResticRetentionPolicy `json:"retention"`
	Backend    backup.BackendConfig         `json:"backend"`
}

func (api *API) GetBackupConfig(w http.ResponseWriter, r *http.Request) {
	config, err := backup.LoadBackupConfig(api.dataDir)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if config == nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"configured": false,
		})
		return
	}

	response := BackupConfigResponse{
		Repository: config.Repository,
		Staging:    config.Staging,
		Retention:  config.Retention,
		Backend:    config.Backend,
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"configured": true,
		"config":     response,
	})
}

func (api *API) UpdateBackupConfig(w http.ResponseWriter, r *http.Request) {
	var req BackupConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.Repository == "" {
		respondError(w, http.StatusBadRequest, "repository is required")
		return
	}

	config := &backup.BackupConfig{
		Repository: req.Repository,
		Staging:    req.Staging,
	}

	if req.Retention != nil {
		config.Retention = *req.Retention
	} else {
		config.Retention = backup.ResticRetentionPolicy{
			KeepDaily:   7,
			KeepWeekly:  4,
			KeepMonthly: 6,
			KeepYearly:  2,
		}
	}

	if req.Backend != nil {
		config.Backend = *req.Backend
	}

	if err := backup.SaveBackupConfig(api.dataDir, config); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if req.Secrets != nil {
		secrets := &backup.BackupSecrets{
			Password: req.Secrets.Password,
		}

		if req.Secrets.Credentials != nil {
			secrets.Credentials = *req.Secrets.Credentials
		}

		if err := backup.SaveBackupSecrets(api.dataDir, secrets); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

func (api *API) TestBackupConfig(w http.ResponseWriter, r *http.Request) {
	client, err := backup.NewResticClient(api.dataDir)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	initialized := client.IsInitialized()

	result := map[string]interface{}{
		"success":     true,
		"initialized": initialized,
	}

	if initialized {
		status, err := client.Status()
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}

		result["status"] = status
	}

	respondJSON(w, http.StatusOK, result)
}

func (api *API) GetRepositoryStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]

	instanceDir := tools.GetInstancePath(api.dataDir, instanceName)

	client, err := backup.NewResticClient(instanceDir)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	status, err := client.Status()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, status)
}

func (api *API) InitRepository(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]

	instanceDir := tools.GetInstancePath(api.dataDir, instanceName)

	client, err := backup.NewResticClient(instanceDir)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if client.IsInitialized() {
		respondError(w, http.StatusBadRequest, "repository already initialized")
		return
	}

	if err := client.Init(); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}
