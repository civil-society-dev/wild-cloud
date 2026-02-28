package backup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/wild-cloud/wild-central/daemon/internal/apps"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
)

// LonghornStrategy implements backup strategy for PVCs using CSI snapshots
type LonghornStrategy struct {
	dataDir string
}

// NewLonghornStrategy creates a new Longhorn/CSI backup strategy
func NewLonghornStrategy(dataDir string) *LonghornStrategy {
	return &LonghornStrategy{
		dataDir: dataDir,
	}
}

// Name returns the strategy identifier
func (l *LonghornStrategy) Name() string {
	return "pvc"
}

// VolumeSnapshot represents a Kubernetes VolumeSnapshot
type VolumeSnapshot struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
	Spec struct {
		Source struct {
			PersistentVolumeClaimName string `yaml:"persistentVolumeClaimName,omitempty"`
			VolumeSnapshotContentName string `yaml:"volumeSnapshotContentName,omitempty"`
		} `yaml:"source"`
	} `yaml:"spec"`
	Status struct {
		ReadyToUse                 bool   `yaml:"readyToUse"`
		BoundVolumeSnapshotContentName string `yaml:"boundVolumeSnapshotContentName"`
	} `yaml:"status"`
}

// Backup creates CSI snapshots of all PVCs for an app
func (l *LonghornStrategy) Backup(instanceName, appName string, manifest *apps.AppManifest, dest BackupDestination) (*ComponentBackup, error) {
	kubeconfigPath := tools.GetKubeconfigPath(l.dataDir, instanceName)

	// Get all PVCs in the app namespace
	pvcs, err := l.getPVCs(kubeconfigPath, appName)
	if err != nil {
		return nil, fmt.Errorf("failed to get PVCs: %w", err)
	}

	if len(pvcs) == 0 {
		// No PVCs to backup
		return nil, nil
	}

	timestamp := time.Now().Format("20060102-150405")
	snapshots := []map[string]string{}

	// Create a snapshot for each PVC
	for _, pvc := range pvcs {
		// Skip cache or temp volumes
		if strings.Contains(pvc, "-cache") || strings.Contains(pvc, "-tmp") {
			continue
		}

		snapshotName := fmt.Sprintf("%s-%s-backup-%s", appName, pvc, timestamp)

		// Create VolumeSnapshot resource
		snapshot := fmt.Sprintf(`apiVersion: snapshot.storage.k8s.io/v1
kind: VolumeSnapshot
metadata:
  name: %s
  namespace: %s
spec:
  volumeSnapshotClassName: longhorn-snapshot-class
  source:
    persistentVolumeClaimName: %s`, snapshotName, appName, pvc)

		// Apply the snapshot
		cmd := exec.Command("kubectl", "apply", "-f", "-")
		tools.WithKubeconfig(cmd, kubeconfigPath)
		cmd.Stdin = strings.NewReader(snapshot)

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to create snapshot for PVC %s: %w, stderr: %s", pvc, err, stderr.String())
		}

		// Wait for snapshot to be ready
		if err := l.waitForSnapshot(kubeconfigPath, appName, snapshotName); err != nil {
			// Clean up failed snapshot
			l.deleteSnapshot(kubeconfigPath, appName, snapshotName)
			return nil, fmt.Errorf("snapshot not ready for PVC %s: %w", pvc, err)
		}

		// Get snapshot details including the content name
		contentName, err := l.getSnapshotContentName(kubeconfigPath, appName, snapshotName)
		if err != nil {
			return nil, fmt.Errorf("failed to get snapshot content name: %w", err)
		}

		snapshots = append(snapshots, map[string]string{
			"pvc":         pvc,
			"snapshot":    snapshotName,
			"contentName": contentName,
		})

		// For Longhorn, the snapshot is stored in the cluster itself
		// We save metadata about it to the backup destination
		metadataKey := fmt.Sprintf("snapshots/%s/%s/%s.json", instanceName, appName, snapshotName)
		metadata, _ := json.Marshal(map[string]interface{}{
			"snapshot":    snapshotName,
			"pvc":         pvc,
			"contentName": contentName,
			"namespace":   appName,
			"timestamp":   timestamp,
			"type":        "csi",
		})

		if _, err := dest.Put(metadataKey, bytes.NewReader(metadata)); err != nil {
			// Clean up snapshot on failure
			l.deleteSnapshot(kubeconfigPath, appName, snapshotName)
			return nil, fmt.Errorf("failed to save snapshot metadata: %w", err)
		}
	}

	if len(snapshots) == 0 {
		// No PVCs were backed up
		return nil, nil
	}

	// Return component backup info
	return &ComponentBackup{
		Type:     "pvc",
		Name:     fmt.Sprintf("pvcs.%s", appName),
		Size:     0, // CSI snapshots don't have a size in the traditional sense
		Location: fmt.Sprintf("snapshots/%s/%s/%s", instanceName, appName, timestamp),
		Metadata: map[string]interface{}{
			"snapshots": snapshots,
			"count":     len(snapshots),
			"type":      "csi",
		},
	}, nil
}

// Restore restores PVCs from CSI snapshots
func (l *LonghornStrategy) Restore(component *ComponentBackup, dest BackupDestination) error {
	// Get instance name from component location
	parts := strings.Split(component.Location, "/")
	if len(parts) < 3 {
		return fmt.Errorf("invalid backup location format")
	}
	instanceName := parts[1]
	appName := parts[2]

	kubeconfigPath := tools.GetKubeconfigPath(l.dataDir, instanceName)

	snapshots, ok := component.Metadata["snapshots"].([]interface{})
	if !ok {
		return fmt.Errorf("no snapshots found in backup metadata")
	}

	// Scale down the app first to avoid conflicts
	if err := l.scaleApp(kubeconfigPath, appName, 0); err != nil {
		return fmt.Errorf("failed to scale down app: %w", err)
	}

	// Restore each PVC from its snapshot
	for _, s := range snapshots {
		snapshot, ok := s.(map[string]interface{})
		if !ok {
			continue
		}

		pvcName, _ := snapshot["pvc"].(string)
		snapshotName, _ := snapshot["snapshot"].(string)

		if pvcName == "" || snapshotName == "" {
			continue
		}

		// Delete existing PVC if it exists
		deleteCmd := exec.Command("kubectl", "delete", "pvc", "-n", appName, pvcName, "--ignore-not-found")
		tools.WithKubeconfig(deleteCmd, kubeconfigPath)
		deleteCmd.Run()

		// Create new PVC from snapshot
		pvcFromSnapshot := fmt.Sprintf(`apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: %s
  namespace: %s
spec:
  dataSource:
    name: %s
    kind: VolumeSnapshot
    apiGroup: snapshot.storage.k8s.io
  accessModes:
    - ReadWriteOnce
  storageClassName: longhorn
  resources:
    requests:
      storage: 10Gi`, pvcName, appName, snapshotName)

		cmd := exec.Command("kubectl", "apply", "-f", "-")
		tools.WithKubeconfig(cmd, kubeconfigPath)
		cmd.Stdin = strings.NewReader(pvcFromSnapshot)

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to restore PVC %s from snapshot: %w, stderr: %s", pvcName, err, stderr.String())
		}

		// Wait for PVC to be bound
		if err := l.waitForPVC(kubeconfigPath, appName, pvcName); err != nil {
			return fmt.Errorf("restored PVC %s not ready: %w", pvcName, err)
		}
	}

	// Scale app back up
	if err := l.scaleApp(kubeconfigPath, appName, 1); err != nil {
		return fmt.Errorf("failed to scale up app: %w", err)
	}

	return nil
}

// Verify checks if CSI snapshots exist and are valid
func (l *LonghornStrategy) Verify(component *ComponentBackup, dest BackupDestination) error {
	// Get instance name from component location
	parts := strings.Split(component.Location, "/")
	if len(parts) < 3 {
		return fmt.Errorf("invalid backup location format")
	}
	instanceName := parts[1]
	appName := parts[2]

	kubeconfigPath := tools.GetKubeconfigPath(l.dataDir, instanceName)

	snapshots, ok := component.Metadata["snapshots"].([]interface{})
	if !ok {
		return fmt.Errorf("no snapshots found in backup metadata")
	}

	// Verify each snapshot exists and is ready
	for _, s := range snapshots {
		snapshot, ok := s.(map[string]interface{})
		if !ok {
			continue
		}

		snapshotName, _ := snapshot["snapshot"].(string)
		if snapshotName == "" {
			continue
		}

		// Check if snapshot exists
		cmd := exec.Command("kubectl", "get", "volumesnapshot", "-n", appName, snapshotName, "-o", "json")
		tools.WithKubeconfig(cmd, kubeconfigPath)

		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("snapshot %s not found: %w", snapshotName, err)
		}

		// Parse snapshot status
		var snap VolumeSnapshot
		if err := json.Unmarshal(output, &snap); err != nil {
			return fmt.Errorf("failed to parse snapshot %s: %w", snapshotName, err)
		}

		if !snap.Status.ReadyToUse {
			return fmt.Errorf("snapshot %s is not ready", snapshotName)
		}
	}

	return nil
}

// Supports checks if this strategy can handle the app based on its manifest
func (l *LonghornStrategy) Supports(manifest *apps.AppManifest) bool {
	// Longhorn strategy supports any app that might have PVCs
	// We'll check for actual PVCs at backup time
	// Return true for all apps for now
	return true
}

// getPVCs returns all PVC names in the app namespace
func (l *LonghornStrategy) getPVCs(kubeconfigPath, namespace string) ([]string, error) {
	cmd := exec.Command("kubectl", "get", "pvc", "-n", namespace, "-o", "jsonpath={.items[*].metadata.name}")
	tools.WithKubeconfig(cmd, kubeconfigPath)

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	pvcs := strings.Fields(string(output))
	return pvcs, nil
}

// waitForSnapshot waits for a snapshot to be ready
func (l *LonghornStrategy) waitForSnapshot(kubeconfigPath, namespace, snapshotName string) error {
	maxRetries := 60 // 5 minutes with 5-second intervals
	for i := 0; i < maxRetries; i++ {
		cmd := exec.Command("kubectl", "get", "volumesnapshot", "-n", namespace, snapshotName,
			"-o", "jsonpath={.status.readyToUse}")
		tools.WithKubeconfig(cmd, kubeconfigPath)

		output, err := cmd.Output()
		if err == nil && string(output) == "true" {
			return nil
		}

		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("timeout waiting for snapshot to be ready")
}

// getSnapshotContentName gets the VolumeSnapshotContent name for a snapshot
func (l *LonghornStrategy) getSnapshotContentName(kubeconfigPath, namespace, snapshotName string) (string, error) {
	cmd := exec.Command("kubectl", "get", "volumesnapshot", "-n", namespace, snapshotName,
		"-o", "jsonpath={.status.boundVolumeSnapshotContentName}")
	tools.WithKubeconfig(cmd, kubeconfigPath)

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// deleteSnapshot deletes a VolumeSnapshot
func (l *LonghornStrategy) deleteSnapshot(kubeconfigPath, namespace, snapshotName string) error {
	cmd := exec.Command("kubectl", "delete", "volumesnapshot", "-n", namespace, snapshotName, "--ignore-not-found")
	tools.WithKubeconfig(cmd, kubeconfigPath)
	return cmd.Run()
}

// waitForPVC waits for a PVC to be bound
func (l *LonghornStrategy) waitForPVC(kubeconfigPath, namespace, pvcName string) error {
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

// scaleApp scales the app deployment
func (l *LonghornStrategy) scaleApp(kubeconfigPath, namespace string, replicas int) error {
	cmd := exec.Command("kubectl", "scale", "deployment", "-n", namespace,
		"-l", fmt.Sprintf("app=%s", namespace), "--replicas", fmt.Sprintf("%d", replicas))
	tools.WithKubeconfig(cmd, kubeconfigPath)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Some apps might use StatefulSet instead of Deployment
		cmd = exec.Command("kubectl", "scale", "statefulset", "-n", namespace,
			"-l", fmt.Sprintf("app=%s", namespace), "--replicas", fmt.Sprintf("%d", replicas))
		tools.WithKubeconfig(cmd, kubeconfigPath)

		if err := cmd.Run(); err != nil {
			// Not all apps can be scaled, so this is not fatal
			return nil
		}
	}

	// Wait a bit for pods to terminate or start
	time.Sleep(5 * time.Second)
	return nil
}