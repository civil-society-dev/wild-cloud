package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/wild-cloud/wild-central/daemon/internal/backup"
	"github.com/wild-cloud/wild-central/daemon/internal/operations"
	"github.com/wild-cloud/wild-central/daemon/internal/sse"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
	"gopkg.in/yaml.v3"
)

// BackupAppStart starts a backup operation for an app
func (api *API) BackupAppStart(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	appName := GetAppName(r)

	// Publish backup started event
	api.sseManager.Broadcast(&sse.Event{
		Type:         "backup:started",
		InstanceName: instanceName,
		Data: map[string]interface{}{
			"app": appName,
		},
		Metadata: map[string]interface{}{
			"app": appName,
		},
	})

	api.StartAsyncOperation(w, instanceName, "backup", appName,
		func(opsMgr *operations.Manager, opID string) error {
			_ = opsMgr.UpdateProgress(instanceName, opID, 10, "Starting backup")

			// Create progress callback for the backup manager
			progressCallback := func(progress int, message string) {
				_ = opsMgr.UpdateProgress(instanceName, opID, progress, message)
			}

			mgr := backup.NewManagerWithProgress(api.dataDir, progressCallback)
			backupInfo, err := mgr.BackupApp(instanceName, appName)

			// Publish backup completed or failed event
			if err != nil {
				api.sseManager.Broadcast(&sse.Event{
					Type:         "backup:failed",
					InstanceName: instanceName,
					Data: map[string]interface{}{
						"app":   appName,
						"error": err.Error(),
					},
					Metadata: map[string]interface{}{
						"app": appName,
					},
				})
			} else {
				api.sseManager.Broadcast(&sse.Event{
					Type:         "backup:completed",
					InstanceName: instanceName,
					Data: map[string]interface{}{
						"app":    appName,
						"backup": backupInfo,
					},
					Metadata: map[string]interface{}{
						"app": appName,
					},
				})
			}

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

// BackupAppLatest handles GET /api/v1/instances/{name}/apps/{app}/backup/latest
func (api *API) BackupAppLatest(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	appName := GetAppName(r)
	mgr := backup.NewManager(api.dataDir)
	backups, err := mgr.ListBackups(instanceName, appName)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list backups")
		return
	}

	// Return only the latest backup (backups are already sorted newest first)
	var latestBackup interface{}
	if len(backups) > 0 {
		latestBackup = backups[0]
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    latestBackup,
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

	// Default to blue-green restore for safety
	if opts.BlueGreen == false && !opts.SkipData {
		opts.BlueGreen = true
	}

	// Publish restore started event
	api.sseManager.Broadcast(&sse.Event{
		Type:         "restore:started",
		InstanceName: instanceName,
		Data: map[string]interface{}{
			"app": appName,
		},
		Metadata: map[string]interface{}{
			"app": appName,
		},
	})

	api.StartAsyncOperation(w, instanceName, "restore", appName,
		func(opsMgr *operations.Manager, opID string) error {
			_ = opsMgr.UpdateProgress(instanceName, opID, 10, "Starting restore")

			// Create progress callback for the backup manager
			progressCallback := func(progress int, message string) {
				_ = opsMgr.UpdateProgress(instanceName, opID, progress, message)
			}

			mgr := backup.NewManagerWithProgress(api.dataDir, progressCallback)
			err := mgr.RestoreApp(instanceName, appName, opts)

			// Publish restore completed or failed event
			if err != nil {
				api.sseManager.Broadcast(&sse.Event{
					Type:         "restore:failed",
					InstanceName: instanceName,
					Data: map[string]interface{}{
						"app":   appName,
						"error": err.Error(),
					},
					Metadata: map[string]interface{}{
						"app": appName,
					},
				})
			} else {
				api.sseManager.Broadcast(&sse.Event{
					Type:         "restore:completed",
					InstanceName: instanceName,
					Data: map[string]interface{}{
						"app": appName,
					},
					Metadata: map[string]interface{}{
						"app": appName,
					},
				})
			}

			return err
		})
}

// BackupAppDelete deletes a specific app backup
func (api *API) BackupAppDelete(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	appName := GetAppName(r)
	timestamp := mux.Vars(r)["timestamp"]

	mgr := backup.NewManager(api.dataDir)
	if err := mgr.DeleteAppBackup(instanceName, appName, timestamp); err != nil {
		// Publish delete failed event
		api.sseManager.Broadcast(&sse.Event{
			Type:         "backup:delete:failed",
			InstanceName: instanceName,
			Data: map[string]interface{}{
				"app":       appName,
				"timestamp": timestamp,
				"error":     err.Error(),
			},
			Metadata: map[string]interface{}{
				"app": appName,
			},
		})
		respondError(w, http.StatusInternalServerError, "Failed to delete backup")
		return
	}

	// Publish delete completed event
	api.sseManager.Broadcast(&sse.Event{
		Type:         "backup:deleted",
		InstanceName: instanceName,
		Data: map[string]interface{}{
			"app":       appName,
			"timestamp": timestamp,
		},
		Metadata: map[string]interface{}{
			"app": appName,
		},
	})

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Backup deleted successfully",
	})
}

// BackupAppVerify verifies a specific app backup can be restored
func (api *API) BackupAppVerify(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	appName := GetAppName(r)
	timestamp := mux.Vars(r)["timestamp"]

	// Publish verify started event
	api.sseManager.Broadcast(&sse.Event{
		Type:         "backup:verify:started",
		InstanceName: instanceName,
		Data: map[string]interface{}{
			"app":       appName,
			"timestamp": timestamp,
		},
		Metadata: map[string]interface{}{
			"app": appName,
		},
	})

	mgr := backup.NewManager(api.dataDir)
	result, err := mgr.VerifyBackup(instanceName, appName, timestamp)
	if err != nil {
		// Publish verify failed event
		api.sseManager.Broadcast(&sse.Event{
			Type:         "backup:verify:failed",
			InstanceName: instanceName,
			Data: map[string]interface{}{
				"app":       appName,
				"timestamp": timestamp,
				"error":     err.Error(),
			},
			Metadata: map[string]interface{}{
				"app": appName,
			},
		})
		respondError(w, http.StatusInternalServerError, "Failed to verify backup")
		return
	}

	// Publish verify completed event
	api.sseManager.Broadcast(&sse.Event{
		Type:         "backup:verified",
		InstanceName: instanceName,
		Data: map[string]interface{}{
			"app":       appName,
			"timestamp": timestamp,
			"result":    result,
		},
		Metadata: map[string]interface{}{
			"app": appName,
		},
	})

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    result,
	})
}

// BackupResourceInfo contains information about a discovered backup resource
type BackupResourceInfo struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`  // "database", "pvc", "secret"
	Plugin       string                 `json:"plugin"` // "postgres", "mysql", "longhorn-pvc", etc.
	Source       map[string]interface{} `json:"source"` // Resource-specific info
	ShouldBackup bool                   `json:"shouldBackup"`
	Reason       string                 `json:"reason,omitempty"` // Why it's included/excluded
}

// BackupAppDiscoverResources auto-discovers backup resources for an app
func (api *API) BackupAppDiscoverResources(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)
	appName := GetAppName(r)

	// Validate instance
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, "Instance not found")
		return
	}

	// Check if app exists
	appPath := filepath.Join(api.dataDir, "instances", instanceName, "apps", appName)
	if _, err := os.Stat(appPath); os.IsNotExist(err) {
		respondError(w, http.StatusNotFound, fmt.Sprintf("App not found: %s", appName))
		return
	}

	resources := []BackupResourceInfo{}

	// 1. Use kustomize to build all resources and discover from there
	kustomizeResources := discoverFromKustomize(appPath)
	resources = append(resources, kustomizeResources...)

	// 2. Discover databases from requires section (these aren't in kustomize output)
	manifestPath := filepath.Join(appPath, "manifest.yaml")
	databases := discoverDatabases(api.dataDir, instanceName, appName, manifestPath)
	resources = append(resources, databases...)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"app":       appName,
			"resources": resources,
		},
	})
}

// discoverFromKustomize uses kustomize build to discover all resources
func discoverFromKustomize(appPath string) []BackupResourceInfo {
	resources := []BackupResourceInfo{}

	// Run kustomize build to get all resources
	cmd := exec.Command("kubectl", "kustomize", appPath)
	output, err := cmd.Output()
	if err != nil {
		// Fallback to using kustomize directly if kubectl kustomize fails
		cmd = exec.Command("kustomize", "build", appPath)
		output, err = cmd.Output()
		if err != nil {
			// If kustomize isn't available either, return empty
			return resources
		}
	}

	// Parse the multi-document YAML output
	docs := bytes.Split(output, []byte("---"))

	for _, doc := range docs {
		doc = bytes.TrimSpace(doc)
		if len(doc) == 0 {
			continue
		}

		var resource map[string]interface{}
		if err := yaml.Unmarshal(doc, &resource); err != nil {
			continue
		}

		kind, _ := resource["kind"].(string)

		switch kind {
		case "PersistentVolumeClaim":
			resources = append(resources, parsePVC(resource))

		case "StatefulSet":
			// Extract volumeClaimTemplates
			if spec, ok := resource["spec"].(map[string]interface{}); ok {
				if vcts, ok := spec["volumeClaimTemplates"].([]interface{}); ok {
					metadata, _ := resource["metadata"].(map[string]interface{})
					ssName, _ := metadata["name"].(string)

					for _, vct := range vcts {
						if vctMap, ok := vct.(map[string]interface{}); ok {
							pvc := parseVolumeClaimTemplate(vctMap, ssName)
							resources = append(resources, pvc)
						}
					}
				}
			}
		}
	}

	return resources
}

// parsePVC extracts PVC info from a manifest
func parsePVC(pvc map[string]interface{}) BackupResourceInfo {
	metadata, _ := pvc["metadata"].(map[string]interface{})
	name, _ := metadata["name"].(string)

	spec, _ := pvc["spec"].(map[string]interface{})
	storageClass := "longhorn" // default
	if sc, ok := spec["storageClassName"].(string); ok {
		storageClass = sc
	}

	// Determine plugin based on storage class
	plugin := detectStoragePlugin(storageClass)

	// Check if it's a cache or temp volume
	shouldBackup := !strings.Contains(name, "-cache") && !strings.Contains(name, "-tmp")
	reason := ""
	if !shouldBackup {
		reason = "Cache or temporary storage"
	}

	// Get storage size
	size := "unknown"
	if resources, ok := spec["resources"].(map[string]interface{}); ok {
		if requests, ok := resources["requests"].(map[string]interface{}); ok {
			if storage, ok := requests["storage"].(string); ok {
				size = storage
			}
		}
	}

	return BackupResourceInfo{
		Name:         name,
		Type:         "pvc",
		Plugin:       plugin,
		Source: map[string]interface{}{
			"pvcName":      name,
			"storageClass": storageClass,
			"size":         size,
		},
		ShouldBackup: shouldBackup,
		Reason:       reason,
	}
}

// parseVolumeClaimTemplate parses a StatefulSet's volumeClaimTemplate
func parseVolumeClaimTemplate(vct map[string]interface{}, statefulSetName string) BackupResourceInfo {
	metadata, _ := vct["metadata"].(map[string]interface{})
	name, _ := metadata["name"].(string)

	spec, _ := vct["spec"].(map[string]interface{})
	storageClass := "longhorn" // default
	if sc, ok := spec["storageClassName"].(string); ok {
		storageClass = sc
	}

	// Get storage size
	size := "unknown"
	if resources, ok := spec["resources"].(map[string]interface{}); ok {
		if requests, ok := resources["requests"].(map[string]interface{}); ok {
			if storage, ok := requests["storage"].(string); ok {
				size = storage
			}
		}
	}

	// For StatefulSets, the actual PVC name follows the pattern: {template-name}-{statefulset-name}-{ordinal}
	pvcName := fmt.Sprintf("%s-%s-0", name, statefulSetName)

	// Determine if it should be backed up
	shouldBackup := !strings.Contains(name, "-cache") && !strings.Contains(name, "-tmp")
	reason := ""
	if !shouldBackup {
		reason = "Cache or temporary storage"
	}

	return BackupResourceInfo{
		Name:         pvcName,
		Type:         "pvc",
		Plugin:       detectStoragePlugin(storageClass),
		Source: map[string]interface{}{
			"pvcName":      pvcName,
			"storageClass": storageClass,
			"size":         size,
			"statefulSet":  true,
		},
		ShouldBackup: shouldBackup,
		Reason:       reason,
	}
}

// detectStoragePlugin returns the appropriate plugin for a storage class
func detectStoragePlugin(storageClass string) string {
	switch storageClass {
	case "longhorn":
		return "longhorn-pvc"
	case "nfs":
		return "nfs"
	case "local-path":
		return "local-path"
	default:
		return storageClass
	}
}

// discoverDatabases finds databases from app requirements
func discoverDatabases(dataDir, instanceName, appName, manifestPath string) []BackupResourceInfo {
	resources := []BackupResourceInfo{}

	// Read app manifest
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return resources
	}

	var manifest map[string]interface{}
	if err := yaml.Unmarshal(manifestData, &manifest); err != nil {
		return resources
	}

	// Check requires section
	if requires, ok := manifest["requires"].([]interface{}); ok {
		// Read config.yaml to get database names
		configPath := tools.GetInstanceConfigPath(dataDir, instanceName)
		configData, _ := os.ReadFile(configPath)
		var config map[string]interface{}
		yaml.Unmarshal(configData, &config)

		appConfig := map[string]interface{}{}
		if apps, ok := config["apps"].(map[string]interface{}); ok {
			if ac, ok := apps[appName].(map[string]interface{}); ok {
				appConfig = ac
			}
		}

		for _, req := range requires {
			if reqMap, ok := req.(map[string]interface{}); ok {
				depName, _ := reqMap["name"].(string)
				installedAs, _ := reqMap["installedAs"].(string)
				if installedAs == "" {
					installedAs = depName
				}

				// Check if it's a database
				if isDatabase(depName) {
					dbName := appName // default
					if dn, ok := appConfig["dbName"].(string); ok {
						dbName = dn
					} else if dn, ok := appConfig["db"].(map[string]interface{}); ok {
						if n, ok := dn["name"].(string); ok {
							dbName = n
						}
					}

					// Check if it's a cache (Redis/Memcached)
					shouldBackup := depName != "redis" && depName != "memcached"
					reason := ""
					if !shouldBackup {
						reason = "Cache database"
					}

					resources = append(resources, BackupResourceInfo{
						Name:   fmt.Sprintf("%s.%s", installedAs, dbName),
						Type:   "database",
						Plugin: depName,
						Source: map[string]interface{}{
							"database": dbName,
							"instance": installedAs,
							"type":     depName,
						},
						ShouldBackup: shouldBackup,
						Reason:       reason,
					})
				}
			}
		}
	}

	return resources
}

// isDatabase checks if a dependency is a database
func isDatabase(name string) bool {
	databases := []string{"postgres", "postgresql", "mysql", "mariadb", "redis", "memcached", "mongodb"}
	for _, db := range databases {
		if name == db {
			return true
		}
	}
	return false
}

