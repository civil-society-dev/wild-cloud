// Package utilities provides helper functions for cluster operations
package utilities

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// HealthStatus represents cluster health information
type HealthStatus struct {
	Overall   string            `json:"overall"`   // healthy, degraded, unhealthy
	Components map[string]string `json:"components"` // component -> status
	Issues    []string          `json:"issues"`
}

// DashboardToken represents a Kubernetes dashboard token
type DashboardToken struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

// NodeIP represents a node's IP address information
type NodeIP struct {
	Hostname   string `json:"hostname"`
	InternalIP string `json:"internal_ip"`
	ExternalIP string `json:"external_ip,omitempty"`
}

// GetClusterHealth checks the health of cluster components
func GetClusterHealth(kubeconfigPath string) (*HealthStatus, error) {
	status := &HealthStatus{
		Overall:    "healthy",
		Components: make(map[string]string),
		Issues:     []string{},
	}

	// Check MetalLB
	if err := checkComponent(kubeconfigPath, "MetalLB", "metallb-system", "app=metallb"); err != nil {
		status.Components["metallb"] = "unhealthy"
		status.Issues = append(status.Issues, fmt.Sprintf("MetalLB: %v", err))
		status.Overall = "degraded"
	} else {
		status.Components["metallb"] = "healthy"
	}

	// Check Traefik
	if err := checkComponent(kubeconfigPath, "Traefik", "traefik", "app.kubernetes.io/name=traefik"); err != nil {
		status.Components["traefik"] = "unhealthy"
		status.Issues = append(status.Issues, fmt.Sprintf("Traefik: %v", err))
		status.Overall = "degraded"
	} else {
		status.Components["traefik"] = "healthy"
	}

	// Check cert-manager
	if err := checkComponent(kubeconfigPath, "cert-manager", "cert-manager", "app.kubernetes.io/instance=cert-manager"); err != nil {
		status.Components["cert-manager"] = "unhealthy"
		status.Issues = append(status.Issues, fmt.Sprintf("cert-manager: %v", err))
		status.Overall = "degraded"
	} else {
		status.Components["cert-manager"] = "healthy"
	}

	// Check Longhorn
	if err := checkComponent(kubeconfigPath, "Longhorn", "longhorn-system", "app=longhorn-manager"); err != nil {
		status.Components["longhorn"] = "unhealthy"
		status.Issues = append(status.Issues, fmt.Sprintf("Longhorn: %v", err))
		status.Overall = "degraded"
	} else {
		status.Components["longhorn"] = "healthy"
	}

	if len(status.Issues) > 3 {
		status.Overall = "unhealthy"
	}

	return status, nil
}

// checkComponent checks if a component is running
func checkComponent(kubeconfigPath, name, namespace, selector string) error {
	args := []string{"get", "pods", "-n", namespace, "-l", selector, "-o", "json"}
	if kubeconfigPath != "" {
		args = append([]string{"--kubeconfig", kubeconfigPath}, args...)
	}

	cmd := exec.Command("kubectl", args...)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get pods: %w", err)
	}

	var result struct {
		Items []struct {
			Status struct {
				Phase string `json:"phase"`
				ContainerStatuses []struct {
					Ready bool `json:"ready"`
				} `json:"containerStatuses"`
			} `json:"status"`
		} `json:"items"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return fmt.Errorf("failed to parse output: %w", err)
	}

	if len(result.Items) == 0 {
		return fmt.Errorf("no pods found")
	}

	for _, pod := range result.Items {
		if pod.Status.Phase != "Running" {
			return fmt.Errorf("pod not running (phase: %s)", pod.Status.Phase)
		}
		for _, container := range pod.Status.ContainerStatuses {
			if !container.Ready {
				return fmt.Errorf("container not ready")
			}
		}
	}

	return nil
}

// GetDashboardToken retrieves or creates a Kubernetes dashboard token
func GetDashboardToken() (*DashboardToken, error) {
	// Check if service account exists
	cmd := exec.Command("kubectl", "get", "serviceaccount", "-n", "kubernetes-dashboard", "dashboard-admin")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("dashboard-admin service account not found")
	}

	// Create token
	cmd = exec.Command("kubectl", "-n", "kubernetes-dashboard", "create", "token", "dashboard-admin")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to create token: %w", err)
	}

	token := strings.TrimSpace(string(output))
	return &DashboardToken{
		Token: token,
	}, nil
}

// GetDashboardTokenFromSecret retrieves dashboard token from secret (fallback method)
func GetDashboardTokenFromSecret() (*DashboardToken, error) {
	cmd := exec.Command("kubectl", "-n", "kubernetes-dashboard", "get", "secret",
		"dashboard-admin-token", "-o", "jsonpath={.data.token}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get token secret: %w", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(string(output))
	if err != nil {
		return nil, fmt.Errorf("failed to decode token: %w", err)
	}

	return &DashboardToken{
		Token: string(decoded),
	}, nil
}

// GetNodeIPs returns IP addresses for all cluster nodes
func GetNodeIPs() ([]*NodeIP, error) {
	cmd := exec.Command("kubectl", "get", "nodes", "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}

	var result struct {
		Items []struct {
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
			Status struct {
				Addresses []struct {
					Type    string `json:"type"`
					Address string `json:"address"`
				} `json:"addresses"`
			} `json:"status"`
		} `json:"items"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse output: %w", err)
	}

	var nodes []*NodeIP
	for _, item := range result.Items {
		node := &NodeIP{
			Hostname: item.Metadata.Name,
		}
		for _, addr := range item.Status.Addresses {
			switch addr.Type {
			case "InternalIP":
				node.InternalIP = addr.Address
			case "ExternalIP":
				node.ExternalIP = addr.Address
			}
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

// GetControlPlaneIP returns the IP of the first control plane node
func GetControlPlaneIP() (string, error) {
	cmd := exec.Command("kubectl", "get", "nodes", "-l", "node-role.kubernetes.io/control-plane",
		"-o", "jsonpath={.items[0].status.addresses[?(@.type==\"InternalIP\")].address}")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get control plane IP: %w", err)
	}

	ip := strings.TrimSpace(string(output))
	if ip == "" {
		return "", fmt.Errorf("no control plane IP found")
	}

	return ip, nil
}

// CopySecretBetweenNamespaces copies a secret from one namespace to another
func CopySecretBetweenNamespaces(secretName, srcNamespace, dstNamespace string) error {
	// Get secret from source namespace
	cmd := exec.Command("kubectl", "get", "secret", "-n", srcNamespace, secretName, "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get secret from %s: %w", srcNamespace, err)
	}

	// Parse and modify secret
	var secret map[string]interface{}
	if err := json.Unmarshal(output, &secret); err != nil {
		return fmt.Errorf("failed to parse secret: %w", err)
	}

	// Remove fields that shouldn't be copied
	if metadata, ok := secret["metadata"].(map[string]interface{}); ok {
		delete(metadata, "resourceVersion")
		delete(metadata, "uid")
		delete(metadata, "creationTimestamp")
		metadata["namespace"] = dstNamespace
	}

	// Convert back to JSON
	secretJSON, err := json.Marshal(secret)
	if err != nil {
		return fmt.Errorf("failed to marshal secret: %w", err)
	}

	// Apply to destination namespace
	cmd = exec.Command("kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(string(secretJSON))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to apply secret to %s: %w\nOutput: %s", dstNamespace, err, string(output))
	}

	return nil
}

// GetClusterVersion returns the Kubernetes cluster version
func GetClusterVersion() (string, error) {
	cmd := exec.Command("kubectl", "version", "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get cluster version: %w", err)
	}

	var result struct {
		ServerVersion struct {
			GitVersion string `json:"gitVersion"`
		} `json:"serverVersion"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return "", fmt.Errorf("failed to parse version: %w", err)
	}

	return result.ServerVersion.GitVersion, nil
}

// GetTalosVersion returns the Talos version for nodes
func GetTalosVersion() (string, error) {
	cmd := exec.Command("talosctl", "version", "--short")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get Talos version: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}
