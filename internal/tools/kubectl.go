package tools

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Kubectl provides a thin wrapper around the kubectl command-line tool
type Kubectl struct {
	kubeconfigPath string
}

// NewKubectl creates a new Kubectl wrapper
func NewKubectl(kubeconfigPath string) *Kubectl {
	return &Kubectl{
		kubeconfigPath: kubeconfigPath,
	}
}

// DeploymentExists checks if a deployment exists in the specified namespace
func (k *Kubectl) DeploymentExists(name, namespace string) bool {
	args := []string{
		"get", "deployment", name,
		"-n", namespace,
	}

	if k.kubeconfigPath != "" {
		args = append([]string{"--kubeconfig", k.kubeconfigPath}, args...)
	}

	cmd := exec.Command("kubectl", args...)
	err := cmd.Run()
	return err == nil
}

// PodInfo represents pod information from kubectl
type PodInfo struct {
	Name      string
	Status    string
	Ready     string
	Restarts  int
	Age       string
	Node      string
	IP        string
}

// DeploymentInfo represents deployment information
type DeploymentInfo struct {
	Desired   int32
	Current   int32
	Ready     int32
	Available int32
}

// GetPods retrieves pod information for a namespace
func (k *Kubectl) GetPods(namespace string) ([]PodInfo, error) {
	args := []string{
		"get", "pods",
		"-n", namespace,
		"-o", "json",
	}

	if k.kubeconfigPath != "" {
		args = append([]string{"--kubeconfig", k.kubeconfigPath}, args...)
	}

	cmd := exec.Command("kubectl", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get pods: %w", err)
	}

	var podList struct {
		Items []struct {
			Metadata struct {
				Name              string    `json:"name"`
				CreationTimestamp time.Time `json:"creationTimestamp"`
			} `json:"metadata"`
			Spec struct {
				NodeName string `json:"nodeName"`
			} `json:"spec"`
			Status struct {
				Phase             string `json:"phase"`
				PodIP             string `json:"podIP"`
				ContainerStatuses []struct {
					Ready        bool `json:"ready"`
					RestartCount int  `json:"restartCount"`
				} `json:"containerStatuses"`
			} `json:"status"`
		} `json:"items"`
	}

	if err := json.Unmarshal(output, &podList); err != nil {
		return nil, fmt.Errorf("failed to parse pod list: %w", err)
	}

	pods := make([]PodInfo, 0, len(podList.Items))
	for _, pod := range podList.Items {
		// Calculate ready containers
		readyCount := 0
		totalCount := len(pod.Status.ContainerStatuses)
		restarts := 0
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Ready {
				readyCount++
			}
			restarts += cs.RestartCount
		}

		// Calculate age
		age := formatAge(time.Since(pod.Metadata.CreationTimestamp))

		pods = append(pods, PodInfo{
			Name:     pod.Metadata.Name,
			Status:   pod.Status.Phase,
			Ready:    fmt.Sprintf("%d/%d", readyCount, totalCount),
			Restarts: restarts,
			Age:      age,
			Node:     pod.Spec.NodeName,
			IP:       pod.Status.PodIP,
		})
	}

	return pods, nil
}

// GetDeployment retrieves deployment information
func (k *Kubectl) GetDeployment(name, namespace string) (*DeploymentInfo, error) {
	args := []string{
		"get", "deployment", name,
		"-n", namespace,
		"-o", "json",
	}

	if k.kubeconfigPath != "" {
		args = append([]string{"--kubeconfig", k.kubeconfigPath}, args...)
	}

	cmd := exec.Command("kubectl", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	var deployment struct {
		Status struct {
			Replicas          int32 `json:"replicas"`
			UpdatedReplicas   int32 `json:"updatedReplicas"`
			ReadyReplicas     int32 `json:"readyReplicas"`
			AvailableReplicas int32 `json:"availableReplicas"`
		} `json:"status"`
		Spec struct {
			Replicas int32 `json:"replicas"`
		} `json:"spec"`
	}

	if err := json.Unmarshal(output, &deployment); err != nil {
		return nil, fmt.Errorf("failed to parse deployment: %w", err)
	}

	return &DeploymentInfo{
		Desired:   deployment.Spec.Replicas,
		Current:   deployment.Status.Replicas,
		Ready:     deployment.Status.ReadyReplicas,
		Available: deployment.Status.AvailableReplicas,
	}, nil
}

// GetLogs retrieves logs from a pod
func (k *Kubectl) GetLogs(namespace, podName string, opts LogOptions) (string, error) {
	args := []string{
		"logs", podName,
		"-n", namespace,
	}

	if opts.Container != "" {
		args = append(args, "-c", opts.Container)
	}
	if opts.Tail > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", opts.Tail))
	}
	if opts.Previous {
		args = append(args, "--previous")
	}
	if opts.Since != "" {
		args = append(args, "--since", opts.Since)
	}

	if k.kubeconfigPath != "" {
		args = append([]string{"--kubeconfig", k.kubeconfigPath}, args...)
	}

	cmd := exec.Command("kubectl", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get logs: %w", err)
	}

	return string(output), nil
}

// LogOptions configures log retrieval
type LogOptions struct {
	Container string
	Tail      int
	Previous  bool
	Since     string
}

// StreamLogs streams logs from a pod
func (k *Kubectl) StreamLogs(namespace, podName string, opts LogOptions) (*exec.Cmd, error) {
	args := []string{
		"logs", podName,
		"-n", namespace,
		"-f", // follow
	}

	if opts.Container != "" {
		args = append(args, "-c", opts.Container)
	}
	if opts.Tail > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", opts.Tail))
	}
	if opts.Since != "" {
		args = append(args, "--since", opts.Since)
	}

	if k.kubeconfigPath != "" {
		args = append([]string{"--kubeconfig", k.kubeconfigPath}, args...)
	}

	cmd := exec.Command("kubectl", args...)
	return cmd, nil
}

// formatAge converts a duration to a human-readable age string
func formatAge(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

// GetFirstPodName returns the name of the first pod in a namespace
func (k *Kubectl) GetFirstPodName(namespace string) (string, error) {
	pods, err := k.GetPods(namespace)
	if err != nil {
		return "", err
	}
	if len(pods) == 0 {
		return "", fmt.Errorf("no pods found in namespace %s", namespace)
	}
	return pods[0].Name, nil
}

// GetPodContainers returns container names for a pod
func (k *Kubectl) GetPodContainers(namespace, podName string) ([]string, error) {
	args := []string{
		"get", "pod", podName,
		"-n", namespace,
		"-o", "jsonpath={.spec.containers[*].name}",
	}

	if k.kubeconfigPath != "" {
		args = append([]string{"--kubeconfig", k.kubeconfigPath}, args...)
	}

	cmd := exec.Command("kubectl", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get pod containers: %w", err)
	}

	containerNames := strings.Fields(string(output))
	return containerNames, nil
}
