package tools

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Kubectl provides a comprehensive wrapper around the kubectl command-line tool
type Kubectl struct {
	kubeconfigPath string
}

// NewKubectl creates a new Kubectl wrapper
func NewKubectl(kubeconfigPath string) *Kubectl {
	return &Kubectl{
		kubeconfigPath: kubeconfigPath,
	}
}

// Pod Information Structures

// PodInfo represents pod information from kubectl
type PodInfo struct {
	Name       string          `json:"name"`
	Status     string          `json:"status"`
	Ready      string          `json:"ready"`
	Restarts   int             `json:"restarts"`
	Age        string          `json:"age"`
	Node       string          `json:"node,omitempty"`
	IP         string          `json:"ip,omitempty"`
	Containers []ContainerInfo `json:"containers,omitempty"`
	Conditions []PodCondition  `json:"conditions,omitempty"`
}

// ContainerInfo represents detailed container information
type ContainerInfo struct {
	Name         string         `json:"name"`
	Image        string         `json:"image"`
	Ready        bool           `json:"ready"`
	RestartCount int            `json:"restartCount"`
	State        ContainerState `json:"state"`
}

// ContainerState represents the state of a container
type ContainerState struct {
	Status  string    `json:"status"`
	Reason  string    `json:"reason,omitempty"`
	Message string    `json:"message,omitempty"`
	Since   time.Time `json:"since,omitempty"`
}

// PodCondition represents a pod condition
type PodCondition struct {
	Type    string    `json:"type"`
	Status  string    `json:"status"`
	Reason  string    `json:"reason,omitempty"`
	Message string    `json:"message,omitempty"`
	Since   time.Time `json:"since,omitempty"`
}

// Deployment Information Structures

// DeploymentInfo represents deployment information
type DeploymentInfo struct {
	Desired   int32 `json:"desired"`
	Current   int32 `json:"current"`
	Ready     int32 `json:"ready"`
	Available int32 `json:"available"`
}

// ReplicaInfo represents aggregated replica information
type ReplicaInfo struct {
	Desired   int `json:"desired"`
	Current   int `json:"current"`
	Ready     int `json:"ready"`
	Available int `json:"available"`
}

// Resource Information Structures

// ResourceMetric represents resource usage for a specific resource type
type ResourceMetric struct {
	Used       string  `json:"used"`
	Requested  string  `json:"requested"`
	Limit      string  `json:"limit"`
	Percentage float64 `json:"percentage"`
}

// ResourceUsage represents aggregated resource usage
type ResourceUsage struct {
	CPU     *ResourceMetric `json:"cpu,omitempty"`
	Memory  *ResourceMetric `json:"memory,omitempty"`
	Storage *ResourceMetric `json:"storage,omitempty"`
}

// Event Information Structures

// KubernetesEvent represents a Kubernetes event
type KubernetesEvent struct {
	Type      string    `json:"type"`
	Reason    string    `json:"reason"`
	Message   string    `json:"message"`
	Count     int       `json:"count"`
	FirstSeen time.Time `json:"firstSeen"`
	LastSeen  time.Time `json:"lastSeen"`
	Object    string    `json:"object"`
}

// Logging Structures

// LogOptions configures log retrieval
type LogOptions struct {
	Container    string
	Tail         int
	Previous     bool
	Since        string
	SinceSeconds int
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Pod       string    `json:"pod"`
}

// Pod Operations

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

// GetPods retrieves pod information for a namespace
// If detailed is true, includes containers and conditions
func (k *Kubectl) GetPods(namespace string, detailed bool) ([]PodInfo, error) {
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
				NodeName   string `json:"nodeName"`
				Containers []struct {
					Name  string `json:"name"`
					Image string `json:"image"`
				} `json:"containers"`
			} `json:"spec"`
			Status struct {
				Phase      string `json:"phase"`
				PodIP      string `json:"podIP"`
				Conditions []struct {
					Type               string    `json:"type"`
					Status             string    `json:"status"`
					LastTransitionTime time.Time `json:"lastTransitionTime"`
					Reason             string    `json:"reason"`
					Message            string    `json:"message"`
				} `json:"conditions"`
				ContainerStatuses []struct {
					Name         string `json:"name"`
					Image        string `json:"image"`
					Ready        bool   `json:"ready"`
					RestartCount int    `json:"restartCount"`
					State        struct {
						Running    *struct{ StartedAt time.Time }    `json:"running,omitempty"`
						Waiting    *struct{ Reason, Message string } `json:"waiting,omitempty"`
						Terminated *struct {
							Reason     string
							Message    string
							FinishedAt time.Time
						} `json:"terminated,omitempty"`
					} `json:"state"`
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
		totalRestarts := 0

		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Ready {
				readyCount++
			}
			totalRestarts += cs.RestartCount
		}

		// Ensure status is never empty
		status := pod.Status.Phase
		if status == "" {
			status = "Unknown"
		}

		podInfo := PodInfo{
			Name:     pod.Metadata.Name,
			Status:   status,
			Ready:    fmt.Sprintf("%d/%d", readyCount, totalCount),
			Restarts: totalRestarts,
			Age:      formatAge(time.Since(pod.Metadata.CreationTimestamp)),
			Node:     pod.Spec.NodeName,
			IP:       pod.Status.PodIP,
		}

		// Include detailed information if requested
		if detailed {
			// Add container details
			containers := make([]ContainerInfo, 0, len(pod.Status.ContainerStatuses))
			for _, cs := range pod.Status.ContainerStatuses {
				containerState := ContainerState{Status: "unknown"}
				if cs.State.Running != nil {
					containerState.Status = "running"
					containerState.Since = cs.State.Running.StartedAt
				} else if cs.State.Waiting != nil {
					containerState.Status = "waiting"
					containerState.Reason = cs.State.Waiting.Reason
					containerState.Message = cs.State.Waiting.Message
				} else if cs.State.Terminated != nil {
					containerState.Status = "terminated"
					containerState.Reason = cs.State.Terminated.Reason
					containerState.Message = cs.State.Terminated.Message
					containerState.Since = cs.State.Terminated.FinishedAt
				}

				containers = append(containers, ContainerInfo{
					Name:         cs.Name,
					Image:        cs.Image,
					Ready:        cs.Ready,
					RestartCount: cs.RestartCount,
					State:        containerState,
				})
			}
			podInfo.Containers = containers

			// Add condition details
			conditions := make([]PodCondition, 0, len(pod.Status.Conditions))
			for _, cond := range pod.Status.Conditions {
				conditions = append(conditions, PodCondition{
					Type:    cond.Type,
					Status:  cond.Status,
					Reason:  cond.Reason,
					Message: cond.Message,
					Since:   cond.LastTransitionTime,
				})
			}
			podInfo.Conditions = conditions
		}

		pods = append(pods, podInfo)
	}

	return pods, nil
}

// GetFirstPodName returns the name of the first pod in a namespace
func (k *Kubectl) GetFirstPodName(namespace string) (string, error) {
	pods, err := k.GetPods(namespace, false)
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

// Deployment Operations

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

// GetDaemonSet retrieves daemonset information
func (k *Kubectl) GetDaemonSet(name, namespace string) (*DeploymentInfo, error) {
	args := []string{
		"get", "daemonset", name,
		"-n", namespace,
		"-o", "json",
	}

	if k.kubeconfigPath != "" {
		args = append([]string{"--kubeconfig", k.kubeconfigPath}, args...)
	}

	cmd := exec.Command("kubectl", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get daemonset: %w", err)
	}

	var daemonset struct {
		Status struct {
			CurrentNumberScheduled int32 `json:"currentNumberScheduled"`
			NumberReady            int32 `json:"numberReady"`
			DesiredNumberScheduled int32 `json:"desiredNumberScheduled"`
			NumberAvailable        int32 `json:"numberAvailable"`
		} `json:"status"`
	}

	if err := json.Unmarshal(output, &daemonset); err != nil {
		return nil, fmt.Errorf("failed to parse daemonset: %w", err)
	}

	return &DeploymentInfo{
		Desired:   daemonset.Status.DesiredNumberScheduled,
		Current:   daemonset.Status.CurrentNumberScheduled,
		Ready:     daemonset.Status.NumberReady,
		Available: daemonset.Status.NumberAvailable,
	}, nil
}

// GetReplicas retrieves aggregated replica information for a namespace
func (k *Kubectl) GetReplicas(namespace string) (*ReplicaInfo, error) {
	info := &ReplicaInfo{}

	// Get deployments
	deployCmd := exec.Command("kubectl", "get", "deployments", "-n", namespace, "-o", "json")
	WithKubeconfig(deployCmd, k.kubeconfigPath)

	deployOutput, err := deployCmd.Output()
	if err == nil {
		var deployList struct {
			Items []struct {
				Spec struct {
					Replicas int `json:"replicas"`
				} `json:"spec"`
				Status struct {
					Replicas          int `json:"replicas"`
					ReadyReplicas     int `json:"readyReplicas"`
					AvailableReplicas int `json:"availableReplicas"`
				} `json:"status"`
			} `json:"items"`
		}

		if json.Unmarshal(deployOutput, &deployList) == nil {
			for _, deploy := range deployList.Items {
				info.Desired += deploy.Spec.Replicas
				info.Current += deploy.Status.Replicas
				info.Ready += deploy.Status.ReadyReplicas
				info.Available += deploy.Status.AvailableReplicas
			}
		}
	}

	// Get statefulsets
	stsCmd := exec.Command("kubectl", "get", "statefulsets", "-n", namespace, "-o", "json")
	WithKubeconfig(stsCmd, k.kubeconfigPath)

	stsOutput, err := stsCmd.Output()
	if err == nil {
		var stsList struct {
			Items []struct {
				Spec struct {
					Replicas int `json:"replicas"`
				} `json:"spec"`
				Status struct {
					Replicas      int `json:"replicas"`
					ReadyReplicas int `json:"readyReplicas"`
				} `json:"status"`
			} `json:"items"`
		}

		if json.Unmarshal(stsOutput, &stsList) == nil {
			for _, sts := range stsList.Items {
				info.Desired += sts.Spec.Replicas
				info.Current += sts.Status.Replicas
				info.Ready += sts.Status.ReadyReplicas
				// StatefulSets don't have availableReplicas, use ready as proxy
				info.Available += sts.Status.ReadyReplicas
			}
		}
	}

	return info, nil
}

// Resource Monitoring

// GetResources retrieves aggregated resource usage for a namespace
func (k *Kubectl) GetResources(namespace string) (*ResourceUsage, error) {
	cmd := exec.Command("kubectl", "get", "pods", "-n", namespace, "-o", "json")
	WithKubeconfig(cmd, k.kubeconfigPath)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get pods: %w", err)
	}

	var podList struct {
		Items []struct {
			Spec struct {
				Containers []struct {
					Resources struct {
						Requests map[string]string `json:"requests,omitempty"`
						Limits   map[string]string `json:"limits,omitempty"`
					} `json:"resources"`
				} `json:"containers"`
			} `json:"spec"`
		} `json:"items"`
	}

	if err := json.Unmarshal(output, &podList); err != nil {
		return nil, fmt.Errorf("failed to parse pod list: %w", err)
	}

	// Aggregate resources
	cpuRequests := int64(0)
	cpuLimits := int64(0)
	memRequests := int64(0)
	memLimits := int64(0)

	for _, pod := range podList.Items {
		for _, container := range pod.Spec.Containers {
			if req, ok := container.Resources.Requests["cpu"]; ok {
				cpuRequests += parseResourceQuantity(req)
			}
			if lim, ok := container.Resources.Limits["cpu"]; ok {
				cpuLimits += parseResourceQuantity(lim)
			}
			if req, ok := container.Resources.Requests["memory"]; ok {
				memRequests += parseResourceQuantity(req)
			}
			if lim, ok := container.Resources.Limits["memory"]; ok {
				memLimits += parseResourceQuantity(lim)
			}
		}
	}

	// Build resource usage with metrics
	usage := &ResourceUsage{}

	// CPU metrics (if any resources defined)
	if cpuRequests > 0 || cpuLimits > 0 {
		cpuUsed := cpuRequests // Approximate "used" as requests for now
		cpuPercentage := 0.0
		if cpuLimits > 0 {
			cpuPercentage = float64(cpuUsed) / float64(cpuLimits) * 100
		}
		usage.CPU = &ResourceMetric{
			Used:       formatCPU(cpuUsed),
			Requested:  formatCPU(cpuRequests),
			Limit:      formatCPU(cpuLimits),
			Percentage: cpuPercentage,
		}
	}

	// Memory metrics (if any resources defined)
	if memRequests > 0 || memLimits > 0 {
		memUsed := memRequests // Approximate "used" as requests for now
		memPercentage := 0.0
		if memLimits > 0 {
			memPercentage = float64(memUsed) / float64(memLimits) * 100
		}
		usage.Memory = &ResourceMetric{
			Used:       formatMemory(memUsed),
			Requested:  formatMemory(memRequests),
			Limit:      formatMemory(memLimits),
			Percentage: memPercentage,
		}
	}

	return usage, nil
}

// GetRecentEvents retrieves recent events for a namespace
func (k *Kubectl) GetRecentEvents(namespace string, limit int) ([]KubernetesEvent, error) {
	cmd := exec.Command("kubectl", "get", "events", "-n", namespace,
		"--sort-by=.lastTimestamp", "-o", "json")
	WithKubeconfig(cmd, k.kubeconfigPath)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	var eventList struct {
		Items []struct {
			Type           string    `json:"type"`
			Reason         string    `json:"reason"`
			Message        string    `json:"message"`
			Count          int       `json:"count"`
			FirstTimestamp time.Time `json:"firstTimestamp"`
			LastTimestamp  time.Time `json:"lastTimestamp"`
			InvolvedObject struct {
				Kind string `json:"kind"`
				Name string `json:"name"`
			} `json:"involvedObject"`
		} `json:"items"`
	}

	if err := json.Unmarshal(output, &eventList); err != nil {
		return nil, fmt.Errorf("failed to parse events: %w", err)
	}

	// Sort by last timestamp (most recent first)
	sort.Slice(eventList.Items, func(i, j int) bool {
		return eventList.Items[i].LastTimestamp.After(eventList.Items[j].LastTimestamp)
	})

	// Limit results
	if limit > 0 && len(eventList.Items) > limit {
		eventList.Items = eventList.Items[:limit]
	}

	events := make([]KubernetesEvent, 0, len(eventList.Items))
	for _, event := range eventList.Items {
		events = append(events, KubernetesEvent{
			Type:      event.Type,
			Reason:    event.Reason,
			Message:   event.Message,
			Count:     event.Count,
			FirstSeen: event.FirstTimestamp,
			LastSeen:  event.LastTimestamp,
			Object:    fmt.Sprintf("%s/%s", event.InvolvedObject.Kind, event.InvolvedObject.Name),
		})
	}

	return events, nil
}

// Logging Operations

// GetLogs retrieves logs from a pod
func (k *Kubectl) GetLogs(namespace, podName string, opts LogOptions) ([]LogEntry, error) {
	args := []string{"logs", podName, "-n", namespace}

	if opts.Container != "" {
		args = append(args, "-c", opts.Container)
	}
	if opts.Tail > 0 {
		args = append(args, "--tail", strconv.Itoa(opts.Tail))
	}
	if opts.SinceSeconds > 0 {
		args = append(args, "--since", fmt.Sprintf("%ds", opts.SinceSeconds))
	} else if opts.Since != "" {
		args = append(args, "--since", opts.Since)
	}
	if opts.Previous {
		args = append(args, "--previous")
	}

	if k.kubeconfigPath != "" {
		args = append([]string{"--kubeconfig", k.kubeconfigPath}, args...)
	}

	cmd := exec.Command("kubectl", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	entries := make([]LogEntry, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}
		entries = append(entries, LogEntry{
			Timestamp: time.Now(), // Best effort - kubectl doesn't provide structured timestamps
			Message:   line,
			Pod:       podName,
		})
	}

	return entries, nil
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

// Helper Functions

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

// parseResourceQuantity converts kubernetes resource quantities to millicores/bytes
func parseResourceQuantity(quantity string) int64 {
	quantity = strings.TrimSpace(quantity)
	if quantity == "" {
		return 0
	}

	// Handle CPU (cores)
	if strings.HasSuffix(quantity, "m") {
		val, _ := strconv.ParseInt(strings.TrimSuffix(quantity, "m"), 10, 64)
		return val
	}

	// Handle memory (bytes)
	multipliers := map[string]int64{
		"Ki": 1024,
		"Mi": 1024 * 1024,
		"Gi": 1024 * 1024 * 1024,
		"Ti": 1024 * 1024 * 1024 * 1024,
		"K":  1000,
		"M":  1000 * 1000,
		"G":  1000 * 1000 * 1000,
		"T":  1000 * 1000 * 1000 * 1000,
	}

	for suffix, mult := range multipliers {
		if strings.HasSuffix(quantity, suffix) {
			val, _ := strconv.ParseInt(strings.TrimSuffix(quantity, suffix), 10, 64)
			return val * mult
		}
	}

	// Plain number
	val, _ := strconv.ParseInt(quantity, 10, 64)
	return val
}

// formatCPU formats millicores to human-readable format
func formatCPU(millicores int64) string {
	if millicores == 0 {
		return "0"
	}
	if millicores < 1000 {
		return fmt.Sprintf("%dm", millicores)
	}
	return fmt.Sprintf("%.1f", float64(millicores)/1000.0)
}

// formatMemory formats bytes to human-readable format
func formatMemory(bytes int64) string {
	if bytes == 0 {
		return "0"
	}

	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"Ki", "Mi", "Gi", "Ti"}
	return fmt.Sprintf("%.1f%s", float64(bytes)/float64(div), units[exp])
}
