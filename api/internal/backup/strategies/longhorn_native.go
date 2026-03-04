package strategies

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/wild-cloud/wild-central/daemon/internal/apps"
	btypes "github.com/wild-cloud/wild-central/daemon/internal/backup/types"
	"github.com/wild-cloud/wild-central/daemon/internal/operations"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
)

// LonghornNativeStrategy implements backup strategy using Longhorn native backups to NFS
type LonghornNativeStrategy struct {
	dataDir   string
	opManager *operations.Manager
}

// NewLonghornNativeStrategy creates a new Longhorn native backup strategy
func NewLonghornNativeStrategy(dataDir string) *LonghornNativeStrategy {
	return &LonghornNativeStrategy{
		dataDir:   dataDir,
		opManager: operations.NewManager(dataDir),
	}
}

// Name returns the strategy identifier
func (l *LonghornNativeStrategy) Name() string {
	return "longhorn-native"
}

// LonghornBackup represents a Longhorn Backup CRD
type LonghornBackup struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		Name      string            `json:"name"`
		Namespace string            `json:"namespace"`
		Labels    map[string]string `json:"labels"`
	} `json:"metadata"`
	Spec struct {
		SnapshotName string            `json:"snapshotName"`
		Labels       map[string]string `json:"labels"`
	} `json:"spec"`
	Status struct {
		State           string `json:"state"`
		Progress        int    `json:"progress"`
		URL             string `json:"url"`
		VolumeSize      string `json:"volumeSize"`
		VolumeCreatedAt string `json:"volumeCreatedAt"`
		Messages        map[string]string `json:"messages"`
		Error           string `json:"error"`
	} `json:"status"`
}

// Backup creates Longhorn native backups of all PVCs for an app
func (l *LonghornNativeStrategy) Backup(instanceName, appName string, manifest *apps.AppManifest, dest btypes.BackupDestination) (*btypes.ComponentBackup, error) {
	kubeconfigPath := tools.GetKubeconfigPath(l.dataDir, instanceName)

	// Get all PVCs in the app namespace
	pvcs, err := l.getPVCs(kubeconfigPath, appName)
	if err != nil {
		return nil, fmt.Errorf("failed to get PVCs: %w", err)
	}

	if len(pvcs) == 0 {
		return nil, nil
	}

	timestamp := time.Now().Format("20060102-150405")
	backups := []map[string]string{}

	// Create a Longhorn backup for each PVC
	for _, pvcName := range pvcs {
		// Skip cache or temp volumes
		if strings.Contains(pvcName, "-cache") || strings.Contains(pvcName, "-tmp") {
			continue
		}

		// Get the actual Longhorn volume name from the PV
		volumeName, err := l.getVolumeNameFromPVC(kubeconfigPath, appName, pvcName)
		if err != nil {
			return nil, fmt.Errorf("failed to get volume name for PVC %s: %w", pvcName, err)
		}

		// Create snapshot via Longhorn API
		snapshotName := fmt.Sprintf("%s-%s-snapshot-%s", appName, pvcName, timestamp)
		if err := l.createSnapshot(kubeconfigPath, volumeName, snapshotName); err != nil {
			return nil, fmt.Errorf("failed to create snapshot for volume %s: %w", volumeName, err)
		}

		// Create backup from snapshot via Longhorn API
		backupID, err := l.createBackup(kubeconfigPath, volumeName, snapshotName)
		if err != nil {
			return nil, fmt.Errorf("failed to create backup for volume %s: %w", volumeName, err)
		}

		// Wait for backup to complete and get URL
		backupURL, err := l.waitForBackupComplete(kubeconfigPath, volumeName, backupID)
		if err != nil {
			return nil, fmt.Errorf("backup not ready for volume %s: %w", volumeName, err)
		}

		backups = append(backups, map[string]string{
			"pvc":        pvcName,
			"volume":     volumeName,
			"backupID":   backupID,
			"backupURL":  backupURL,
			"snapshot":   snapshotName,
		})

		// Clean up old backups (keep only latest)
		l.cleanupOldBackups(kubeconfigPath, volumeName, backupID)
	}

	if len(backups) == 0 {
		return nil, nil
	}

	// Save backup metadata to destination
	metadataKey := fmt.Sprintf("backups/%s/%s/%s.json", instanceName, appName, timestamp)
	metadata, _ := json.Marshal(map[string]interface{}{
		"backups":   backups,
		"timestamp": timestamp,
		"type":      "longhorn-native",
		"instance":  instanceName,
		"app":       appName,
	})

	if _, err := dest.Put(metadataKey, bytes.NewReader(metadata)); err != nil {
		return nil, fmt.Errorf("failed to save backup metadata: %w", err)
	}

	return &btypes.ComponentBackup{
		Type:     "longhorn-native",
		Name:     fmt.Sprintf("volumes.%s", appName),
		Size:     0,
		Location: fmt.Sprintf("backups/%s/%s/%s", instanceName, appName, timestamp),
		Metadata: map[string]interface{}{
			"backups": backups,
			"count":   len(backups),
			"type":    "longhorn-native",
		},
	}, nil
}

// Restore restores PVCs from Longhorn backups using blue-green deployment
func (l *LonghornNativeStrategy) Restore(component *btypes.ComponentBackup, dest btypes.BackupDestination) error {
	fmt.Printf("LonghornNativeStrategy.Restore called with component: %+v\n", component)

	// Get instance and app names from component location
	parts := strings.Split(component.Location, "/")
	if len(parts) < 3 {
		return fmt.Errorf("invalid backup location format")
	}
	instanceName := parts[1]
	appName := parts[2]

	kubeconfigPath := tools.GetKubeconfigPath(l.dataDir, instanceName)

	// Check if this is a blue-green restore
	isBlueGreen := component.Metadata["blueGreen"] == true
	targetNamespace := appName

	fmt.Printf("Longhorn restore: isBlueGreen=%v, targetNamespace=%s\n", isBlueGreen, targetNamespace)

	if isBlueGreen {
		// Create restore namespace for blue-green deployment
		targetNamespace = fmt.Sprintf("%s-restore", appName)
		fmt.Printf("Creating restore namespace: %s\n", targetNamespace)

		// Create the restore namespace if it doesn't exist
		nsYaml := fmt.Sprintf(`apiVersion: v1
kind: Namespace
metadata:
  name: %s
  labels:
    app: %s
    type: restore`, targetNamespace, appName)

		cmd := exec.Command("kubectl", "apply", "-f", "-")
		tools.WithKubeconfig(cmd, kubeconfigPath)
		cmd.Stdin = strings.NewReader(nsYaml)

		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create restore namespace: %w, stderr: %s", err, stderr.String())
		}
		fmt.Printf("Created restore namespace: %s\n", targetNamespace)
	}

	backups, ok := component.Metadata["backups"].([]interface{})
	if !ok {
		return fmt.Errorf("no backups found in metadata")
	}
	fmt.Printf("Found %d backups to restore\n", len(backups))

	// Get Longhorn API endpoint
	fmt.Println("Getting Longhorn API endpoint...")
	apiURL, err := l.getLonghornAPIEndpoint(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to get Longhorn API endpoint: %w", err)
	}
	fmt.Printf("Longhorn API endpoint: %s\n", apiURL)

	// Restore each PVC from its backup
	for _, b := range backups {
		backup, ok := b.(map[string]interface{})
		if !ok {
			continue
		}

		pvcName, _ := backup["pvc"].(string)
		volumeName, _ := backup["volume"].(string)
		backupURL, _ := backup["backupURL"].(string)

		fmt.Printf("Processing backup: pvcName=%s, volumeName=%s, backupURL=%s\n", pvcName, volumeName, backupURL)

		if pvcName == "" || volumeName == "" || backupURL == "" {
			fmt.Println("Skipping backup with missing data")
			continue
		}

		// Get PVC size from existing PVC or use default
		fmt.Printf("Getting PVC size for %s in namespace %s\n", pvcName, appName)
		pvcSize := l.getPVCSize(kubeconfigPath, appName, pvcName)
		fmt.Printf("PVC size: %s\n", pvcSize)

		// Create a new volume from backup via Longhorn API
		restoreVolumeName := fmt.Sprintf("%s-restore-%s", pvcName, time.Now().Format("20060102-150405"))
		fmt.Printf("Creating restore volume %s from backup %s\n", restoreVolumeName, backupURL)

		if err := l.createVolumeFromBackup(kubeconfigPath, apiURL, restoreVolumeName, backupURL, pvcSize); err != nil {
			return fmt.Errorf("failed to create volume from backup for %s: %w", pvcName, err)
		}
		fmt.Printf("Created restore volume %s successfully\n", restoreVolumeName)

		// Store volume mapping for later use by deployment
		// The deployment will create PVCs that reference these volumes
		if component.Metadata == nil {
			component.Metadata = make(map[string]interface{})
		}
		volumeMappings, ok := component.Metadata["volumeMappings"].(map[string]string)
		if !ok {
			volumeMappings = make(map[string]string)
		}
		volumeMappings[pvcName] = restoreVolumeName
		component.Metadata["volumeMappings"] = volumeMappings
		fmt.Printf("Mapped PVC %s to volume %s for deployment\n", pvcName, restoreVolumeName)
	}

	return nil
}

func (l *LonghornNativeStrategy) createVolumeFromBackup(kubeconfigPath, apiURL, volumeName, backupURL, size string) error {
	// Create volume from backup using Longhorn API
	url := fmt.Sprintf("%s/v1/volumes", apiURL)
	fmt.Printf("Creating volume via Longhorn API at: %s\n", url)

	// Parse size to bytes
	sizeBytes := "1073741824" // Default 1Gi
	if strings.HasSuffix(size, "Gi") {
		var sizeInt int
		if _, err := fmt.Sscanf(size, "%dGi", &sizeInt); err == nil {
			sizeBytes = fmt.Sprintf("%d", sizeInt*1024*1024*1024)
		}
	}

	payload := fmt.Sprintf(`{
		"name": "%s",
		"size": "%s",
		"fromBackup": "%s",
		"numberOfReplicas": 3
	}`, volumeName, sizeBytes, backupURL)

	fmt.Printf("Payload: %s\n", payload)

	// Use curl directly since we have port-forward to localhost:8080
	cmd := exec.Command("curl", "-X", "POST", url,
		"-H", "Content-Type: application/json",
		"-d", payload, "-s")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	fmt.Println("Creating volume from backup via Longhorn API...")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create volume from backup: %w, stderr: %s, stdout: %s", err, stderr.String(), stdout.String())
	}
	fmt.Printf("Volume %s creation initiated, response: %s\n", volumeName, stdout.String())

	// Wait for volume to be ready
	return l.waitForVolume(kubeconfigPath, apiURL, volumeName)
}

func (l *LonghornNativeStrategy) waitForVolume(kubeconfigPath, apiURL, volumeName string) error {
	maxRetries := 60 // 5 minutes with 5-second intervals
	for i := 0; i < maxRetries; i++ {
		// Check volume status
		url := fmt.Sprintf("%s/v1/volumes/%s", apiURL, volumeName)

		// Use curl directly since we have port-forward to localhost:8080
		cmd := exec.Command("curl", "-s", url)

		output, err := cmd.Output()
		if err == nil {
			var volume map[string]interface{}
			if err := json.Unmarshal(output, &volume); err == nil {
				// For restore, we just need the volume to exist and be in a stable state
				// It may remain detached until a workload uses it
				if state, _ := volume["state"].(string); state == "detached" || state == "attached" {
					// Check if restore is complete
					if restoreStatus, ok := volume["restoreStatus"].([]interface{}); ok && len(restoreStatus) > 0 {
						// If there are restore status entries, check if any are complete
						for _, rs := range restoreStatus {
							if status, ok := rs.(map[string]interface{}); ok {
								if isRestored, _ := status["isRestored"].(bool); isRestored {
									fmt.Printf("Volume %s restore completed\n", volumeName)
									return nil
								}
							}
						}
					} else {
						// No restore status entries - volume might be ready
						// For freshly created volumes from backup, they start detached but healthy
						if robustness, _ := volume["robustness"].(string); robustness == "healthy" || robustness == "unknown" {
							fmt.Printf("Volume %s is ready (state=%s, robustness=%s)\n", volumeName, state, robustness)
							return nil
						}
					}
				}
			}
		}

		if i%12 == 0 { // Log every minute
			fmt.Printf("Waiting for volume %s to be ready... (%d/%d)\n", volumeName, i, maxRetries)
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("timeout waiting for volume to be ready")
}

// Helper functions

func (l *LonghornNativeStrategy) getPVCs(kubeconfigPath, namespace string) ([]string, error) {
	cmd := exec.Command("kubectl", "get", "pvc", "-n", namespace, "-o", "jsonpath={.items[*].metadata.name}")
	tools.WithKubeconfig(cmd, kubeconfigPath)

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	pvcs := strings.Fields(string(output))
	return pvcs, nil
}

func (l *LonghornNativeStrategy) getPVCSize(kubeconfigPath, namespace, pvcName string) string {
	cmd := exec.Command("kubectl", "get", "pvc", "-n", namespace, pvcName,
		"-o", "jsonpath={.spec.resources.requests.storage}")
	tools.WithKubeconfig(cmd, kubeconfigPath)

	if output, err := cmd.Output(); err == nil && len(output) > 0 {
		return string(output)
	}
	return "10Gi" // Default fallback
}

func (l *LonghornNativeStrategy) getVolumeNameFromPVC(kubeconfigPath, namespace, pvcName string) (string, error) {
	// Get the PV name bound to this PVC
	cmd := exec.Command("kubectl", "get", "pvc", "-n", namespace, pvcName,
		"-o", "jsonpath={.spec.volumeName}")
	tools.WithKubeconfig(cmd, kubeconfigPath)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get volume name: %w", err)
	}

	volumeName := string(output)
	if volumeName == "" {
		return "", fmt.Errorf("no volume bound to PVC %s", pvcName)
	}

	return volumeName, nil
}

func (l *LonghornNativeStrategy) getLonghornAPIEndpoint(kubeconfigPath string) (string, error) {
	// Check if port-forward is already running
	checkCmd := exec.Command("curl", "-s", "-o", "/dev/null", "-w", "%{http_code}", "http://localhost:8080/v1/volumes")
	if err := checkCmd.Run(); err == nil {
		// Port forward is already running
		return "http://localhost:8080", nil
	}

	// Start port-forward in the background
	cmd := exec.Command("kubectl", "port-forward", "-n", "longhorn-system", "service/longhorn-frontend", "8080:80")
	tools.WithKubeconfig(cmd, kubeconfigPath)

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start port-forward: %w", err)
	}

	// Give it a moment to establish
	time.Sleep(3 * time.Second)

	// Verify it's working
	verifyCmd := exec.Command("curl", "-s", "-o", "/dev/null", "-w", "%{http_code}", "http://localhost:8080/v1/volumes")
	if err := verifyCmd.Run(); err != nil {
		return "", fmt.Errorf("port-forward not responding after setup: %w", err)
	}

	return "http://localhost:8080", nil
}

func (l *LonghornNativeStrategy) createSnapshot(kubeconfigPath, volumeName, snapshotName string) error {
	// Get Longhorn API endpoint
	apiURL, err := l.getLonghornAPIEndpoint(kubeconfigPath)
	if err != nil {
		return err
	}

	// Create snapshot via Longhorn API
	url := fmt.Sprintf("%s/v1/volumes/%s?action=snapshotCreate", apiURL, volumeName)
	payload := fmt.Sprintf(`{"name":"%s"}`, snapshotName)

	cmd := exec.Command("curl", "-X", "POST", url,
		"-H", "Content-Type: application/json",
		"-d", payload, "-s")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
	}

	// Wait a moment for snapshot to be created
	time.Sleep(2 * time.Second)
	return nil
}

func (l *LonghornNativeStrategy) createBackup(kubeconfigPath, volumeName, snapshotName string) (string, error) {
	// Get Longhorn API endpoint
	apiURL, err := l.getLonghornAPIEndpoint(kubeconfigPath)
	if err != nil {
		return "", err
	}

	// Create backup via Longhorn API
	url := fmt.Sprintf("%s/v1/volumes/%s?action=snapshotBackup", apiURL, volumeName)
	payload := fmt.Sprintf(`{"name":"%s"}`, snapshotName)

	// Use curl directly from host since we're using port-forward
	cmd := exec.Command("curl", "-X", "POST", url,
		"-H", "Content-Type: application/json",
		"-d", payload, "-s")

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	// Parse response to get backup ID
	var response map[string]interface{}
	if err := json.Unmarshal(output, &response); err != nil {
		return "", fmt.Errorf("failed to parse backup response: %w", err)
	}

	// Extract backup ID from backupStatus
	if backupStatus, ok := response["backupStatus"].([]interface{}); ok && len(backupStatus) > 0 {
		if status, ok := backupStatus[0].(map[string]interface{}); ok {
			if id, ok := status["id"].(string); ok {
				return id, nil
			}
		}
	}

	return "", fmt.Errorf("backup ID not found in response")
}

func (l *LonghornNativeStrategy) waitForBackupComplete(kubeconfigPath, volumeName, backupID string) (string, error) {
	// Get Longhorn API endpoint
	apiURL, err := l.getLonghornAPIEndpoint(kubeconfigPath)
	if err != nil {
		return "", err
	}

	maxRetries := 120 // 10 minutes with 5-second intervals
	for i := 0; i < maxRetries; i++ {
		// Get volume status to check backup progress
		url := fmt.Sprintf("%s/v1/volumes/%s", apiURL, volumeName)

		// Use curl directly from host since we're using port-forward
		cmd := exec.Command("curl", "-s", url)

		output, err := cmd.Output()
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		var volume map[string]interface{}
		if err := json.Unmarshal(output, &volume); err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		// Check backup status
		if backupStatus, ok := volume["backupStatus"].([]interface{}); ok {
			for _, status := range backupStatus {
				if s, ok := status.(map[string]interface{}); ok {
					if id, _ := s["id"].(string); id == backupID {
						if state, _ := s["state"].(string); state == "Completed" {
							// Get the backup URL
							if backupURL, ok := s["backupURL"].(string); ok && backupURL != "" {
								return backupURL, nil
							}
							// If no URL yet, try to get it from backup volume
							return l.getBackupURL(kubeconfigPath, volumeName, backupID)
						}
						if errorMsg, _ := s["error"].(string); errorMsg != "" {
							return "", fmt.Errorf("backup failed: %s", errorMsg)
						}
					}
				}
			}
		}

		time.Sleep(5 * time.Second)
	}
	return "", fmt.Errorf("timeout waiting for backup to complete")
}

func (l *LonghornNativeStrategy) getBackupURL(kubeconfigPath, volumeName, backupID string) (string, error) {
	// Construct backup URL from volume name and backup ID
	// Format: nfs://server:/path/backupstore/volumes/{volumeID}/backups/backup-{id}
	return fmt.Sprintf("backup://%s/%s", volumeName, backupID), nil
}

func (l *LonghornNativeStrategy) waitForPVC(kubeconfigPath, namespace, pvcName string) error {
	maxRetries := 60 // 5 minutes with 5-second intervals
	for i := 0; i < maxRetries; i++ {
		cmd := exec.Command("kubectl", "get", "pvc", "-n", namespace, pvcName,
			"-o", "jsonpath={.status.phase}")
		tools.WithKubeconfig(cmd, kubeconfigPath)

		output, err := cmd.Output()
		if err == nil && string(output) == "Bound" {
			return nil
		}

		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("timeout waiting for PVC to be bound")
}

func (l *LonghornNativeStrategy) cleanupOldBackups(kubeconfigPath, volumeName, keepBackupID string) error {
	// For now, skip automatic cleanup of old backups
	// This can be implemented later using Longhorn's backup volume API
	return nil
}

// Verify checks if Longhorn backups exist and are valid
func (l *LonghornNativeStrategy) Verify(component *btypes.ComponentBackup, dest btypes.BackupDestination) error {
	parts := strings.Split(component.Location, "/")
	if len(parts) < 3 {
		return fmt.Errorf("invalid backup location format")
	}
	instanceName := parts[1]

	kubeconfigPath := tools.GetKubeconfigPath(l.dataDir, instanceName)

	backups, ok := component.Metadata["backups"].([]interface{})
	if !ok {
		return fmt.Errorf("no backups found in metadata")
	}

	// Get Longhorn API endpoint
	apiURL, err := l.getLonghornAPIEndpoint(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to get Longhorn API endpoint: %w", err)
	}

	// Verify each backup exists
	for _, b := range backups {
		backup, ok := b.(map[string]interface{})
		if !ok {
			continue
		}

		volumeName, _ := backup["volume"].(string)
		backupID, _ := backup["backupID"].(string)
		if volumeName == "" || backupID == "" {
			continue
		}

		// Check if backup exists via API
		url := fmt.Sprintf("%s/v1/volumes/%s", apiURL, volumeName)

		cmd := exec.Command("kubectl", "exec", "-n", "longhorn-system",
			"deployment/longhorn-ui", "--",
			"curl", "-s", url)
		tools.WithKubeconfig(cmd, kubeconfigPath)

		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to check backup %s: %w", backupID, err)
		}

		var volume map[string]interface{}
		if err := json.Unmarshal(output, &volume); err != nil {
			return fmt.Errorf("failed to parse volume status: %w", err)
		}

		// Check if our backup ID exists and is completed
		found := false
		if backupStatus, ok := volume["backupStatus"].([]interface{}); ok {
			for _, status := range backupStatus {
				if s, ok := status.(map[string]interface{}); ok {
					if id, _ := s["id"].(string); id == backupID {
						if state, _ := s["state"].(string); state == "Completed" {
							found = true
							break
						}
					}
				}
			}
		}

		if !found {
			return fmt.Errorf("backup %s not found or not completed", backupID)
		}
	}

	return nil
}

// Supports checks if this strategy can handle the app based on its manifest
func (l *LonghornNativeStrategy) Supports(manifest *apps.AppManifest) bool {
	// This strategy supports any app with PVCs
	return true
}