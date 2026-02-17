package v1

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/wild-cloud/wild-central/daemon/internal/config"
	"github.com/wild-cloud/wild-central/daemon/internal/sse"
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

	// Track if domains changed (for DNS update)
	var domainsChanged bool
	if fileType == "config" {
		oldCloud, _ := existingConfig["cloud"].(map[string]interface{})
		newCloud, _ := updates["cloud"].(map[string]interface{})
		if oldCloud != nil && newCloud != nil {
			oldDomain, _ := oldCloud["domain"].(string)
			newDomain, _ := newCloud["domain"].(string)
			oldInternalDomain, _ := oldCloud["internalDomain"].(string)
			newInternalDomain, _ := newCloud["internalDomain"].(string)

			if oldDomain != newDomain || oldInternalDomain != newInternalDomain {
				domainsChanged = true
			}
		}
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

	// Update DNS if domains changed
	if domainsChanged && fileType == "config" {
		go func() {
			log.Printf("Domain change detected for instance %s, updating DNS configuration...", instanceName)

			// Load the full instance config
			instanceConfigPath := api.instance.GetInstanceConfigPath(instanceName)
			instanceCfg, err := config.LoadCloudConfig(instanceConfigPath)
			if err != nil {
				log.Printf("Failed to load instance config for DNS update: %v", err)
				return
			}

			// Update the DNS configuration for this instance
			if err := api.dnsmasq.UpdateInstanceDNS(instanceName, *instanceCfg); err != nil {
				log.Printf("Failed to update DNS for instance %s: %v", instanceName, err)
				return
			}

			log.Printf("Successfully updated DNS configuration for instance %s", instanceName)
		}()
	}

	// Capitalize first letter for message
	fileTypeCap := strings.ToUpper(fileType[:1]) + fileType[1:]
	respondMessage(w, http.StatusOK, fileTypeCap+" updated successfully")
}

// broadcastDnsmasqEvent broadcasts SSE events for dnsmasq status changes
func (api *API) broadcastDnsmasqEvent(eventType string, message string) {
	if api.sseManager == nil {
		return
	}

	// Get current dnsmasq status
	status, err := api.dnsmasq.GetStatus()
	if err != nil {
		status = nil
	}

	event := &sse.Event{
		ID:           fmt.Sprintf("dnsmasq-%d", time.Now().UnixNano()),
		Type:         eventType,
		InstanceName: "global", // dnsmasq is global/central-level, not instance-specific
		Timestamp:    time.Now(),
		Data: map[string]interface{}{
			"message": message,
			"status":  status,
		},
	}

	api.sseManager.Broadcast(event)
}

// broadcastCentralStatusEvent broadcasts SSE events for central status changes
func (api *API) broadcastCentralStatusEvent(startTime time.Time) {
	if api.sseManager == nil {
		return
	}

	// Get list of instances
	instances, err := api.instance.ListInstances()
	if err != nil {
		instances = []string{}
	}

	// Calculate uptime
	uptime := time.Since(startTime)

	event := &sse.Event{
		ID:           fmt.Sprintf("central-%d", time.Now().UnixNano()),
		Type:         "central:status",
		InstanceName: "global", // Central status is global, not instance-specific
		Timestamp:    time.Now(),
		Data: map[string]interface{}{
			"status":        "running",
			"version":       "0.1.0",
			"uptime":        uptime.String(),
			"uptimeSeconds": int(uptime.Seconds()),
			"instances": map[string]interface{}{
				"count": len(instances),
				"names": instances,
			},
		},
	}

	api.sseManager.Broadcast(event)
}
