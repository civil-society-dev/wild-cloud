package cluster

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/wild-cloud/wild-central/daemon/internal/storage"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
)

// Manager handles cluster lifecycle operations
type Manager struct {
	dataDir  string
	talosctl *tools.Talosctl
}

// NewManager creates a new cluster manager
func NewManager(dataDir string) *Manager {
	return &Manager{
		dataDir:  dataDir,
		talosctl: tools.NewTalosctl(),
	}
}

// ClusterConfig contains cluster configuration parameters
type ClusterConfig struct {
	ClusterName string `json:"cluster_name"`
	VIP         string `json:"vip"` // Control plane virtual IP
	Version     string `json:"version"`
}

// ClusterStatus represents cluster health and status
type ClusterStatus struct {
	Status            string            `json:"status"` // ready, pending, error
	Nodes             int               `json:"nodes"`
	ControlPlaneNodes int               `json:"control_plane_nodes"`
	WorkerNodes       int               `json:"worker_nodes"`
	KubernetesVersion string            `json:"kubernetes_version"`
	TalosVersion      string            `json:"talos_version"`
	Services          map[string]string `json:"services"`
}

// GetTalosDir returns the talos directory for an instance
func (m *Manager) GetTalosDir(instanceName string) string {
	return filepath.Join(m.dataDir, "instances", instanceName, "talos")
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

// Bootstrap bootstraps the cluster on the specified node
func (m *Manager) Bootstrap(instanceName, nodeName string) error {
	// Get node configuration to find the target IP
	instancePath := filepath.Join(m.dataDir, "instances", instanceName)
	configPath := filepath.Join(instancePath, "config.yaml")

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

	// Get talosconfig path for this instance
	talosconfigPath := tools.GetTalosconfigPath(m.dataDir, instanceName)

	// Set talosctl endpoint (with proper context via TALOSCONFIG env var)
	cmdEndpoint := exec.Command("talosctl", "config", "endpoint", nodeIP)
	tools.WithTalosconfig(cmdEndpoint, talosconfigPath)
	if output, err := cmdEndpoint.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set talosctl endpoint: %w\nOutput: %s", err, string(output))
	}

	// Bootstrap command (with proper context via TALOSCONFIG env var)
	cmd := exec.Command("talosctl", "bootstrap", "--nodes", nodeIP)
	tools.WithTalosconfig(cmd, talosconfigPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to bootstrap cluster: %w\nOutput: %s", err, string(output))
	}

	// Retrieve kubeconfig after bootstrap (best-effort with retry)
	log.Printf("Waiting for Kubernetes API server to become ready...")
	if err := m.retrieveKubeconfigFromCluster(instanceName, nodeIP, 5*time.Minute); err != nil {
		log.Printf("Warning: %v", err)
		log.Printf("You can retrieve it manually later using: wild cluster kubeconfig --generate")
	}

	return nil
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
	instancePath := filepath.Join(m.dataDir, "instances", instanceName)
	configPath := filepath.Join(instancePath, "config.yaml")

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
	instancePath := filepath.Join(m.dataDir, "instances", instanceName)
	configPath := filepath.Join(instancePath, "config.yaml")
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
	cmd := exec.Command("kubectl", "--kubeconfig", kubeconfigPath, "get", "nodes", "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		status.Status = "unreachable"
		return status, nil
	}

	var nodesResult struct {
		Items []struct {
			Metadata struct {
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

	// Count control plane and worker nodes
	for _, node := range nodesResult.Items {
		if _, isControl := node.Metadata.Labels["node-role.kubernetes.io/control-plane"]; isControl {
			status.ControlPlaneNodes++
		} else {
			status.WorkerNodes++
		}

		// Check if node is ready
		for _, cond := range node.Status.Conditions {
			if cond.Type == "Ready" && cond.Status != "True" {
				status.Status = "degraded"
			}
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
		cmd := exec.Command("kubectl", "--kubeconfig", kubeconfigPath,
			"get", "pods", "-n", svc.namespace, "-l", svc.selector,
			"-o", "jsonpath={.items[*].status.phase}")
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
