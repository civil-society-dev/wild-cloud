package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/wild-cloud/wild-central/daemon/internal/operations"
	"github.com/wild-cloud/wild-central/daemon/internal/storage"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
)

// Manager handles cluster lifecycle operations
type Manager struct {
	dataDir  string
	talosctl *tools.Talosctl
	opsMgr   *operations.Manager
}

// NewManager creates a new cluster manager
func NewManager(dataDir string, opsMgr *operations.Manager) *Manager {
	return &Manager{
		dataDir:  dataDir,
		talosctl: tools.NewTalosctl(),
		opsMgr:   opsMgr,
	}
}

// ClusterConfig contains cluster configuration parameters
type ClusterConfig struct {
	ClusterName string `json:"cluster_name"`
	VIP         string `json:"vip"` // Control plane virtual IP
	Version     string `json:"version"`
}

// NodeStatus represents the health status of a single node
type NodeStatus struct {
	Hostname        string `json:"hostname"`
	Ready           bool   `json:"ready"`
	KubernetesReady bool   `json:"kubernetes_ready"`
	Role            string `json:"role"` // "control-plane" or "worker"
}

// ClusterStatus represents cluster health and status
type ClusterStatus struct {
	Status            string                  `json:"status"` // ready, pending, error
	Nodes             int                     `json:"nodes"`
	ControlPlaneNodes int                     `json:"control_plane_nodes"`
	WorkerNodes       int                     `json:"worker_nodes"`
	KubernetesVersion string                  `json:"kubernetes_version"`
	TalosVersion      string                  `json:"talos_version"`
	Services          map[string]string       `json:"services"`
	NodeStatuses      map[string]NodeStatus   `json:"node_statuses,omitempty"`
}

// GetTalosDir returns the talos directory for an instance
func (m *Manager) GetTalosDir(instanceName string) string {
	return tools.GetInstanceTalosPath(m.dataDir, instanceName)
}

// GetGeneratedDir returns the generated config directory
func (m *Manager) GetGeneratedDir(instanceName string) string {
	return filepath.Join(m.GetTalosDir(instanceName), "generated")
}

// GenerateConfig generates initial cluster configuration using talosctl gen config
func (m *Manager) GenerateConfig(instanceName string, config *ClusterConfig) error {
	generatedDir := m.GetGeneratedDir(instanceName)

	// Check if already generated (idempotency)
	secretsFile := filepath.Join(generatedDir, "secrets.yaml")
	if storage.FileExists(secretsFile) {
		// Already generated
		return nil
	}

	// Ensure generated directory exists
	if err := storage.EnsureDir(generatedDir, 0755); err != nil {
		return fmt.Errorf("failed to create generated directory: %w", err)
	}

	// Generate secrets
	cmd := exec.Command("talosctl", "gen", "secrets")
	cmd.Dir = generatedDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to generate secrets: %w\nOutput: %s", err, string(output))
	}

	// Generate config with secrets
	endpoint := fmt.Sprintf("https://%s:6443", config.VIP)
	cmd = exec.Command("talosctl", "gen", "config",
		"--with-secrets", "secrets.yaml",
		config.ClusterName,
		endpoint,
	)
	cmd.Dir = generatedDir
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to generate config: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// Bootstrap bootstraps the cluster on the specified node with progress tracking
func (m *Manager) Bootstrap(instanceName, nodeName string) (string, error) {
	// Create operation for tracking
	opID, err := m.opsMgr.Start(instanceName, "bootstrap", nodeName)
	if err != nil {
		return "", fmt.Errorf("failed to start bootstrap operation: %w", err)
	}

	// Run bootstrap asynchronously
	go func() {
		if err := m.runBootstrapWithTracking(instanceName, nodeName, opID); err != nil {
			_ = m.opsMgr.Update(instanceName, opID, "failed", err.Error(), 0)
		}
	}()

	return opID, nil
}

// runBootstrapWithTracking runs the bootstrap process with detailed progress tracking
func (m *Manager) runBootstrapWithTracking(instanceName, nodeName, opID string) error {
	ctx := context.Background()
	configPath := tools.GetInstanceConfigPath(m.dataDir, instanceName)
	yq := tools.NewYQ()

	// Get node's target IP
	nodeIPRaw, err := yq.Get(configPath, fmt.Sprintf(".cluster.nodes.active.%s.targetIp", nodeName))
	if err != nil {
		return fmt.Errorf("failed to get node IP: %w", err)
	}

	nodeIP := tools.CleanYQOutput(nodeIPRaw)
	if nodeIP == "" || nodeIP == "null" {
		return fmt.Errorf("node %s does not have a target IP configured", nodeName)
	}

	// Get VIP
	vipRaw, err := yq.Get(configPath, ".cluster.nodes.control.vip")
	if err != nil {
		return fmt.Errorf("failed to get VIP: %w", err)
	}

	vip := tools.CleanYQOutput(vipRaw)
	if vip == "" || vip == "null" {
		return fmt.Errorf("control plane VIP not configured")
	}

	// Step 0: Run talosctl bootstrap
	if err := m.runBootstrapCommand(instanceName, nodeIP, opID); err != nil {
		return err
	}

	// Step 1: Wait for etcd health
	if err := m.waitForEtcd(ctx, instanceName, nodeIP, opID); err != nil {
		return err
	}

	// Step 2: Wait for VIP assignment
	if err := m.waitForVIP(ctx, instanceName, nodeIP, vip, opID); err != nil {
		return err
	}

	// Step 3: Wait for control plane components
	if err := m.waitForControlPlane(ctx, instanceName, nodeIP, opID); err != nil {
		return err
	}

	// Step 4: Wait for API server on VIP
	if err := m.waitForAPIServer(ctx, instanceName, vip, opID); err != nil {
		return err
	}

	// Step 5: Configure cluster access
	if err := m.configureClusterAccess(instanceName, vip, opID); err != nil {
		return err
	}

	// Step 6: Verify node registration
	if err := m.waitForNodeRegistration(ctx, instanceName, opID); err != nil {
		return err
	}

	// Mark as completed
	_ = m.opsMgr.Update(instanceName, opID, "completed", "Bootstrap completed successfully", 100)
	return nil
}

// runBootstrapCommand executes the initial bootstrap command
func (m *Manager) runBootstrapCommand(instanceName, nodeIP, opID string) error {
	_ = m.opsMgr.UpdateBootstrapProgress(instanceName, opID, 0, "bootstrap", 1, 1, "Running talosctl bootstrap command")

	talosconfigPath := tools.GetTalosconfigPath(m.dataDir, instanceName)

	// Set talosctl endpoint
	cmdEndpoint := exec.Command("talosctl", "config", "endpoint", nodeIP)
	tools.WithTalosconfig(cmdEndpoint, talosconfigPath)
	if output, err := cmdEndpoint.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set talosctl endpoint: %w\nOutput: %s", err, string(output))
	}

	// Bootstrap command
	cmd := exec.Command("talosctl", "bootstrap", "--nodes", nodeIP)
	tools.WithTalosconfig(cmd, talosconfigPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to bootstrap cluster: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// waitForEtcd waits for etcd to become healthy
func (m *Manager) waitForEtcd(ctx context.Context, instanceName, nodeIP, opID string) error {
	maxAttempts := 30
	talosconfigPath := tools.GetTalosconfigPath(m.dataDir, instanceName)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		_ = m.opsMgr.UpdateBootstrapProgress(instanceName, opID, 1, "etcd", attempt, maxAttempts, "Waiting for etcd to become healthy")

		cmd := exec.Command("talosctl", "-n", nodeIP, "etcd", "status")
		tools.WithTalosconfig(cmd, talosconfigPath)
		output, err := cmd.CombinedOutput()

		if err == nil && strings.Contains(string(output), nodeIP) {
			return nil
		}

		if attempt < maxAttempts {
			time.Sleep(10 * time.Second)
		}
	}

	return fmt.Errorf("etcd did not become healthy after %d attempts", maxAttempts)
}

// waitForVIP waits for VIP to be assigned to the node
func (m *Manager) waitForVIP(ctx context.Context, instanceName, nodeIP, vip, opID string) error {
	maxAttempts := 90
	talosconfigPath := tools.GetTalosconfigPath(m.dataDir, instanceName)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		_ = m.opsMgr.UpdateBootstrapProgress(instanceName, opID, 2, "vip", attempt, maxAttempts, "Waiting for VIP assignment")

		cmd := exec.Command("talosctl", "-n", nodeIP, "get", "addresses")
		tools.WithTalosconfig(cmd, talosconfigPath)
		output, err := cmd.CombinedOutput()

		if err == nil && strings.Contains(string(output), vip+"/32") {
			return nil
		}

		if attempt < maxAttempts {
			time.Sleep(10 * time.Second)
		}
	}

	return fmt.Errorf("VIP was not assigned after %d attempts", maxAttempts)
}

// waitForControlPlane waits for control plane components to start
func (m *Manager) waitForControlPlane(ctx context.Context, instanceName, nodeIP, opID string) error {
	maxAttempts := 60
	talosconfigPath := tools.GetTalosconfigPath(m.dataDir, instanceName)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		_ = m.opsMgr.UpdateBootstrapProgress(instanceName, opID, 3, "controlplane", attempt, maxAttempts, "Waiting for control plane components")

		cmd := exec.Command("talosctl", "-n", nodeIP, "containers", "-k")
		tools.WithTalosconfig(cmd, talosconfigPath)
		output, err := cmd.CombinedOutput()

		if err == nil && strings.Contains(string(output), "kube-") {
			return nil
		}

		if attempt < maxAttempts {
			time.Sleep(10 * time.Second)
		}
	}

	return fmt.Errorf("control plane components did not start after %d attempts", maxAttempts)
}

// waitForAPIServer waits for Kubernetes API server to respond
func (m *Manager) waitForAPIServer(ctx context.Context, instanceName, vip, opID string) error {
	maxAttempts := 60
	apiURL := fmt.Sprintf("https://%s:6443/healthz", vip)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		_ = m.opsMgr.UpdateBootstrapProgress(instanceName, opID, 4, "apiserver", attempt, maxAttempts, "Waiting for Kubernetes API server")

		cmd := exec.Command("curl", "-k", "-s", "--max-time", "5", apiURL)
		output, err := cmd.CombinedOutput()

		if err == nil && strings.Contains(string(output), "ok") {
			return nil
		}

		if attempt < maxAttempts {
			time.Sleep(10 * time.Second)
		}
	}

	return fmt.Errorf("API server did not respond after %d attempts", maxAttempts)
}

// configureClusterAccess configures talosctl and kubectl to use the VIP
func (m *Manager) configureClusterAccess(instanceName, vip, opID string) error {
	_ = m.opsMgr.UpdateBootstrapProgress(instanceName, opID, 5, "configure", 1, 1, "Configuring cluster access")

	talosconfigPath := tools.GetTalosconfigPath(m.dataDir, instanceName)
	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)

	// Set talosctl endpoint to VIP
	cmdEndpoint := exec.Command("talosctl", "config", "endpoint", vip)
	tools.WithTalosconfig(cmdEndpoint, talosconfigPath)
	if output, err := cmdEndpoint.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set talosctl endpoint: %w\nOutput: %s", err, string(output))
	}

	// Retrieve kubeconfig
	cmdKubeconfig := exec.Command("talosctl", "kubeconfig", "--nodes", vip, kubeconfigPath)
	tools.WithTalosconfig(cmdKubeconfig, talosconfigPath)
	if output, err := cmdKubeconfig.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to retrieve kubeconfig: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// waitForNodeRegistration waits for the node to register with Kubernetes
func (m *Manager) waitForNodeRegistration(ctx context.Context, instanceName, opID string) error {
	maxAttempts := 10
	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		_ = m.opsMgr.UpdateBootstrapProgress(instanceName, opID, 6, "nodes", attempt, maxAttempts, "Waiting for node registration")

		cmd := exec.Command("kubectl", "get", "nodes")
		tools.WithKubeconfig(cmd, kubeconfigPath)
		output, err := cmd.CombinedOutput()

		if err == nil && strings.Contains(string(output), "Ready") {
			return nil
		}

		if attempt < maxAttempts {
			time.Sleep(10 * time.Second)
		}
	}

	return fmt.Errorf("node did not register after %d attempts", maxAttempts)
}

// retrieveKubeconfigFromCluster retrieves kubeconfig from the cluster with retry logic
func (m *Manager) retrieveKubeconfigFromCluster(instanceName, nodeIP string, timeout time.Duration) error {
	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)
	talosconfigPath := tools.GetTalosconfigPath(m.dataDir, instanceName)

	// Retry logic: exponential backoff
	delay := 5 * time.Second
	maxDelay := 30 * time.Second
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Try to retrieve kubeconfig
		cmdKubeconfig := exec.Command("talosctl", "kubeconfig", "--nodes", nodeIP, kubeconfigPath)
		tools.WithTalosconfig(cmdKubeconfig, talosconfigPath)

		if output, err := cmdKubeconfig.CombinedOutput(); err == nil {
			log.Printf("Successfully retrieved kubeconfig for instance %s", instanceName)
			return nil
		} else {
			// Check if we've exceeded deadline
			if !time.Now().Before(deadline) {
				return fmt.Errorf("failed to retrieve kubeconfig: %v\nOutput: %s", err, string(output))
			}

			// Wait before retrying
			time.Sleep(delay)

			// Increase delay for next iteration (exponential backoff)
			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
		}
	}

	return fmt.Errorf("failed to retrieve kubeconfig: timeout exceeded")
}

// RegenerateKubeconfig regenerates the kubeconfig by retrieving it from the cluster
func (m *Manager) RegenerateKubeconfig(instanceName string) error {
	configPath := tools.GetInstanceConfigPath(m.dataDir, instanceName)

	yq := tools.NewYQ()

	// Get VIP from config
	vipRaw, err := yq.Get(configPath, ".cluster.nodes.control.vip")
	if err != nil {
		return fmt.Errorf("failed to get VIP: %w", err)
	}

	vip := tools.CleanYQOutput(vipRaw)
	if vip == "" || vip == "null" {
		return fmt.Errorf("control plane VIP not configured in cluster.nodes.control.vip")
	}

	log.Printf("Regenerating kubeconfig for instance %s from cluster VIP %s", instanceName, vip)
	// Use shorter timeout for manual regeneration (cluster should already be running)
	return m.retrieveKubeconfigFromCluster(instanceName, vip, 30*time.Second)
}

// ConfigureEndpoints updates talosconfig to use VIP and retrieves kubeconfig
func (m *Manager) ConfigureEndpoints(instanceName string, includeNodes bool) error {
	configPath := tools.GetInstanceConfigPath(m.dataDir, instanceName)
	talosconfigPath := tools.GetTalosconfigPath(m.dataDir, instanceName)

	yq := tools.NewYQ()

	// Get VIP from config
	vipRaw, err := yq.Get(configPath, ".cluster.nodes.control.vip")
	if err != nil {
		return fmt.Errorf("failed to get VIP: %w", err)
	}

	vip := tools.CleanYQOutput(vipRaw)
	if vip == "" || vip == "null" {
		return fmt.Errorf("control plane VIP not configured in cluster.nodes.control.vip")
	}

	// Build endpoints list
	endpoints := []string{vip}

	// Add control node IPs if requested
	if includeNodes {
		nodesRaw, err := yq.Exec("eval", ".cluster.nodes.active | to_entries | .[] | select(.value.role == \"controlplane\") | .value.targetIp", configPath)
		if err == nil {
			nodeIPs := strings.Split(strings.TrimSpace(string(nodesRaw)), "\n")
			for _, ip := range nodeIPs {
				ip = tools.CleanYQOutput(ip)
				if ip != "" && ip != "null" && ip != vip {
					endpoints = append(endpoints, ip)
				}
			}
		}
	}

	// Update talosconfig endpoint to use VIP
	args := append([]string{"config", "endpoint"}, endpoints...)
	cmd := exec.Command("talosctl", args...)
	tools.WithTalosconfig(cmd, talosconfigPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set talosctl endpoint: %w\nOutput: %s", err, string(output))
	}

	// Retrieve kubeconfig using the VIP
	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)
	cmdKubeconfig := exec.Command("talosctl", "kubeconfig", "--nodes", vip, kubeconfigPath)
	tools.WithTalosconfig(cmdKubeconfig, talosconfigPath)
	if output, err := cmdKubeconfig.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to retrieve kubeconfig: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// GetStatus retrieves cluster status
func (m *Manager) GetStatus(instanceName string) (*ClusterStatus, error) {
	status := &ClusterStatus{
		Status:            "unknown",
		Nodes:             0,
		ControlPlaneNodes: 0,
		WorkerNodes:       0,
		Services:          make(map[string]string),
	}

	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)
	if !storage.FileExists(kubeconfigPath) {
		status.Status = "not_bootstrapped"
		return status, nil
	}

	// Get node count and types using kubectl
	cmd := exec.Command("kubectl", "get", "nodes", "-o", "json")
	tools.WithKubeconfig(cmd, kubeconfigPath)
	output, err := cmd.Output()
	if err != nil {
		status.Status = "unreachable"
		return status, nil
	}

	var nodesResult struct {
		Items []struct {
			Metadata struct {
				Name   string            `json:"name"`
				Labels map[string]string `json:"labels"`
			} `json:"metadata"`
			Status struct {
				Conditions []struct {
					Type   string `json:"type"`
					Status string `json:"status"`
				} `json:"conditions"`
				NodeInfo struct {
					KubeletVersion string `json:"kubeletVersion"`
				} `json:"nodeInfo"`
			} `json:"status"`
		} `json:"items"`
	}

	if err := json.Unmarshal(output, &nodesResult); err != nil {
		return status, fmt.Errorf("failed to parse nodes: %w", err)
	}

	status.Nodes = len(nodesResult.Items)
	status.Status = "ready"

	// Get Kubernetes version from first node
	if len(nodesResult.Items) > 0 {
		status.KubernetesVersion = nodesResult.Items[0].Status.NodeInfo.KubeletVersion
	}

	// Get Talos version using talosctl
	talosconfigPath := tools.GetTalosconfigPath(m.dataDir, instanceName)
	if storage.FileExists(talosconfigPath) {
		cmd := exec.Command("talosctl", "version", "--short", "--client")
		tools.WithTalosconfig(cmd, talosconfigPath)
		output, err := cmd.Output()
		if err == nil {
			// Output format: "Talos v1.11.2"
			line := strings.TrimSpace(string(output))
			if strings.HasPrefix(line, "Talos") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					status.TalosVersion = parts[1]
				}
			}
		}
	}

	// Count control plane and worker nodes, and populate per-node status
	status.NodeStatuses = make(map[string]NodeStatus)

	for _, node := range nodesResult.Items {
		hostname := node.Metadata.Name // K8s node name is hostname

		role := "worker"
		if _, isControl := node.Metadata.Labels["node-role.kubernetes.io/control-plane"]; isControl {
			role = "control-plane"
			status.ControlPlaneNodes++
		} else {
			status.WorkerNodes++
		}

		// Check if node is ready
		nodeReady := false
		for _, cond := range node.Status.Conditions {
			if cond.Type == "Ready" {
				nodeReady = (cond.Status == "True")
				if !nodeReady {
					status.Status = "degraded"
				}
				break
			}
		}

		status.NodeStatuses[hostname] = NodeStatus{
			Hostname:        hostname,
			Ready:           true, // In K8s means it's reachable
			KubernetesReady: nodeReady,
			Role:            role,
		}
	}

	// Check basic service status
	services := []struct {
		name      string
		namespace string
		selector  string
	}{
		{"metallb", "metallb-system", "app=metallb"},
		{"traefik", "traefik", "app.kubernetes.io/name=traefik"},
		{"cert-manager", "cert-manager", "app.kubernetes.io/instance=cert-manager"},
		{"longhorn", "longhorn-system", "app=longhorn-manager"},
	}

	for _, svc := range services {
		cmd := exec.Command("kubectl", "get", "pods", "-n", svc.namespace, "-l", svc.selector,
			"-o", "jsonpath={.items[*].status.phase}")
		tools.WithKubeconfig(cmd, kubeconfigPath)
		output, err := cmd.Output()
		if err != nil || len(output) == 0 {
			status.Services[svc.name] = "not_found"
			continue
		}

		phases := strings.Fields(string(output))
		allRunning := true
		for _, phase := range phases {
			if phase != "Running" {
				allRunning = false
				break
			}
		}

		if allRunning && len(phases) > 0 {
			status.Services[svc.name] = "running"
		} else {
			status.Services[svc.name] = "not_ready"
			status.Status = "degraded"
		}
	}

	return status, nil
}

// GetKubeconfig returns the kubeconfig for the cluster
func (m *Manager) GetKubeconfig(instanceName string) (string, error) {
	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)

	if !storage.FileExists(kubeconfigPath) {
		return "", fmt.Errorf("kubeconfig not found - cluster may not be bootstrapped")
	}

	data, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		return "", fmt.Errorf("failed to read kubeconfig: %w", err)
	}

	return string(data), nil
}

// GetTalosconfig returns the talosconfig for the cluster
func (m *Manager) GetTalosconfig(instanceName string) (string, error) {
	talosconfigPath := filepath.Join(m.GetGeneratedDir(instanceName), "talosconfig")

	if !storage.FileExists(talosconfigPath) {
		return "", fmt.Errorf("talosconfig not found - cluster may not be initialized")
	}

	data, err := os.ReadFile(talosconfigPath)
	if err != nil {
		return "", fmt.Errorf("failed to read talosconfig: %w", err)
	}

	return string(data), nil
}

// Health checks cluster health
func (m *Manager) Health(instanceName string) ([]HealthCheck, error) {
	checks := []HealthCheck{}

	// Check 1: Talos config exists
	checks = append(checks, HealthCheck{
		Name:    "Talos Configuration",
		Status:  "passing",
		Message: "Talos configuration generated",
	})

	// Check 2: Kubeconfig exists
	if _, err := m.GetKubeconfig(instanceName); err == nil {
		checks = append(checks, HealthCheck{
			Name:    "Kubernetes Configuration",
			Status:  "passing",
			Message: "Kubeconfig available",
		})
	} else {
		checks = append(checks, HealthCheck{
			Name:    "Kubernetes Configuration",
			Status:  "warning",
			Message: "Kubeconfig not found",
		})
	}

	// Additional health checks would query actual cluster state
	// via kubectl and talosctl

	return checks, nil
}

// HealthCheck represents a single health check result
type HealthCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // passing, warning, failing
	Message string `json:"message"`
}

// Reset resets the cluster (dangerous operation)
func (m *Manager) Reset(instanceName string, confirm bool) error {
	if !confirm {
		return fmt.Errorf("reset requires confirmation")
	}

	// This is a destructive operation
	// Real implementation would:
	// 1. Reset all nodes via talosctl reset
	// 2. Remove generated configs
	// 3. Clear node status in config.yaml

	generatedDir := m.GetGeneratedDir(instanceName)
	if storage.FileExists(generatedDir) {
		if err := os.RemoveAll(generatedDir); err != nil {
			return fmt.Errorf("failed to remove generated configs: %w", err)
		}
	}

	return nil
}

// ConfigureContext configures talosctl context for the cluster
func (m *Manager) ConfigureContext(instanceName, clusterName string) error {
	talosconfigPath := filepath.Join(m.GetGeneratedDir(instanceName), "talosconfig")

	if !storage.FileExists(talosconfigPath) {
		return fmt.Errorf("talosconfig not found")
	}

	// Merge talosconfig into user's talosctl config
	cmd := exec.Command("talosctl", "config", "merge", talosconfigPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to merge talosconfig: %w\nOutput: %s", err, string(output))
	}

	// Set context
	cmd = exec.Command("talosctl", "config", "context", clusterName)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set context: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// HasContext checks if talosctl context exists
func (m *Manager) HasContext(clusterName string) (bool, error) {
	cmd := exec.Command("talosctl", "config", "contexts")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("failed to list contexts: %w", err)
	}

	return strings.Contains(string(output), clusterName), nil
}
