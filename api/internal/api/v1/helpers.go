package v1

import (
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/wild-cloud/wild-central/daemon/internal/storage"
	"gopkg.in/yaml.v3"
)

// getNestedValue retrieves a value from a nested map using dot notation path.
// For example, getNestedValue(data, "cluster.nodes.active") returns data["cluster"]["nodes"]["active"].
func getNestedValue(data map[string]interface{}, path string) interface{} {
	keys := strings.Split(path, ".")
	current := data

	for i, key := range keys {
		if i == len(keys)-1 {
			return current[key]
		}

		if next, ok := current[key].(map[string]interface{}); ok {
			current = next
		} else {
			return nil
		}
	}

	return nil
}

// setNestedValue sets a value in a nested map using dot notation path.
// Creates intermediate maps as needed.
func setNestedValue(data map[string]interface{}, path string, value interface{}) {
	keys := strings.Split(path, ".")
	current := data

	for i, key := range keys {
		if i == len(keys)-1 {
			current[key] = value
			return
		}

		if next, ok := current[key].(map[string]interface{}); ok {
			current = next
		} else {
			next := make(map[string]interface{})
			current[key] = next
			current = next
		}
	}
}

// updateYAMLFile updates a YAML file with the provided key-value pairs.
// It performs a shallow merge at the top level, preserving unmodified keys.
func (api *API) updateYAMLFile(w http.ResponseWriter, r *http.Request, instanceName, fileType string) {
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, "Instance not found")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	var updates map[string]interface{}
	if err := yaml.Unmarshal(body, &updates); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid YAML format")
		return
	}

	var filePath string
	if fileType == "config" {
		filePath = api.instance.GetInstanceConfigPath(instanceName)
	} else {
		filePath = api.instance.GetInstanceSecretsPath(instanceName)
	}

	// Read existing file
	existingContent, err := storage.ReadFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		respondError(w, http.StatusInternalServerError, "Failed to read existing file")
		return
	}

	// Parse existing content or initialize empty map
	var existingConfig map[string]interface{}
	if len(existingContent) > 0 {
		if err := yaml.Unmarshal(existingContent, &existingConfig); err != nil {
			respondError(w, http.StatusBadRequest, "Failed to parse existing file")
			return
		}
	} else {
		existingConfig = make(map[string]interface{})
	}

	// Merge updates into existing config (shallow merge for top-level keys)
	for key, value := range updates {
		existingConfig[key] = value
	}

	// Marshal and write
	yamlContent, err := yaml.Marshal(existingConfig)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to marshal YAML")
		return
	}

	lockPath := filePath + ".lock"
	if err := storage.WithLock(lockPath, func() error {
		return storage.WriteFile(filePath, yamlContent, 0644)
	}); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to write file")
		return
	}

	// Capitalize first letter for message
	fileTypeCap := strings.ToUpper(fileType[:1]) + fileType[1:]
	respondMessage(w, http.StatusOK, fileTypeCap+" updated successfully")
}
