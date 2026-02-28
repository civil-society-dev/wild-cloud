package services

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/wild-cloud/wild-central/daemon/internal/contracts"
	"github.com/wild-cloud/wild-central/daemon/internal/storage"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
)

// GetLogs retrieves buffered logs from a service
func (m *Manager) GetLogs(instanceName, serviceName string, opts contracts.ServiceLogsRequest) (*contracts.ServiceLogsResponse, error) {
	// 1. Get service namespace
	manifest, err := m.GetManifest(serviceName)
	if err != nil {
		return nil, fmt.Errorf("service not found: %w", err)
	}

	namespace := manifest.Namespace

	// 2. Get kubeconfig path
	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)
	if !storage.FileExists(kubeconfigPath) {
		return nil, fmt.Errorf("kubeconfig not found - cluster may not be bootstrapped")
	}

	kubectl := tools.NewKubectl(kubeconfigPath)

	// 3. Get pod name - first try to get pods from the service's deployment
	podName := ""
	var pods []tools.PodInfo

	// Try to get pods by deployment name (assumes deployment name matches service name)
	deploymentName := serviceName
	pods, err = kubectl.GetPodsByDeployment(namespace, deploymentName)
	if err != nil || len(pods) == 0 {
		// Fallback to getting all pods in namespace
		pods, err = kubectl.GetPods(namespace, false)
		if err != nil {
			return nil, fmt.Errorf("failed to list pods: %w", err)
		}
	}

	// Check if we have any pods
	if len(pods) == 0 {
		return &contracts.ServiceLogsResponse{
			Lines: []string{"No pods found for service. The service may not be deployed yet."},
		}, nil
	}

	// Use first pod or find pod with specified container
	if opts.Container == "" {
		// Use first pod
		podName = pods[0].Name

		// If no container specified, get first container
		containers, err := kubectl.GetPodContainers(namespace, podName)
		if err != nil {
			return nil, fmt.Errorf("failed to get pod containers: %w", err)
		}
		if len(containers) > 0 {
			opts.Container = containers[0]
		}
	} else {
		// Find pod with specified container
		podName = pods[0].Name
	}

	// 4. Set default tail if not specified
	if opts.Tail == 0 {
		opts.Tail = 100
	}
	// Enforce maximum tail
	if opts.Tail > 5000 {
		opts.Tail = 5000
	}

	// 5. Get logs
	logOpts := tools.LogOptions{
		Container:    opts.Container,
		Tail:         opts.Tail,
		Previous:     opts.Previous,
		Since:        opts.Since,
		SinceSeconds: 0,
	}

	logEntries, err := kubectl.GetLogs(namespace, podName, logOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}

	// 6. Convert structured logs to string lines
	lines := make([]string, 0, len(logEntries))
	for _, entry := range logEntries {
		lines = append(lines, entry.Message)
	}

	truncated := false
	if len(lines) > opts.Tail {
		lines = lines[len(lines)-opts.Tail:]
		truncated = true
	}

	return &contracts.ServiceLogsResponse{
		Service:   serviceName,
		Namespace: namespace,
		Container: opts.Container,
		Lines:     lines,
		Truncated: truncated,
		Timestamp: time.Now(),
	}, nil
}

// StreamLogs streams logs from a service using SSE
func (m *Manager) StreamLogs(instanceName, serviceName string, opts contracts.ServiceLogsRequest, writer io.Writer) error {
	// 1. Get service namespace
	manifest, err := m.GetManifest(serviceName)
	if err != nil {
		return fmt.Errorf("service not found: %w", err)
	}

	namespace := manifest.Namespace

	// 2. Get kubeconfig path
	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)
	if !storage.FileExists(kubeconfigPath) {
		return fmt.Errorf("kubeconfig not found - cluster may not be bootstrapped")
	}

	kubectl := tools.NewKubectl(kubeconfigPath)

	// 3. Get pod name - first try to get pods from the service's deployment
	podName := ""
	var pods []tools.PodInfo

	// Try to get pods by deployment name (assumes deployment name matches service name)
	deploymentName := serviceName
	pods, err = kubectl.GetPodsByDeployment(namespace, deploymentName)
	if err != nil || len(pods) == 0 {
		// Fallback to getting all pods in namespace
		pods, err = kubectl.GetPods(namespace, false)
		if err != nil {
			return fmt.Errorf("failed to list pods: %w", err)
		}
	}

	// Check if we have any pods
	if len(pods) == 0 {
		// Send a message event indicating no pods
		fmt.Fprintf(writer, "data: No pods found for service. The service may not be deployed yet.\n\n")
		return nil
	}

	// Use first pod or find pod with specified container
	if opts.Container == "" {
		// Use first pod
		podName = pods[0].Name

		// Get first container
		containers, err := kubectl.GetPodContainers(namespace, podName)
		if err != nil {
			return fmt.Errorf("failed to get pod containers: %w", err)
		}
		if len(containers) > 0 {
			opts.Container = containers[0]
		}
	} else {
		// Use first pod with specified container
		podName = pods[0].Name
	}

	// 4. Set default tail for streaming
	if opts.Tail == 0 {
		opts.Tail = 50
	}

	// 5. Stream logs
	logOpts := tools.LogOptions{
		Container: opts.Container,
		Tail:      opts.Tail,
		Since:     opts.Since,
	}

	cmd, err := kubectl.StreamLogs(namespace, podName, logOpts)
	if err != nil {
		return fmt.Errorf("failed to start log stream: %w", err)
	}

	// Get stdout pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start kubectl logs: %w", err)
	}

	// Stream logs line by line as SSE events
	scanner := bufio.NewScanner(stdout)
	errScanner := bufio.NewScanner(stderr)

	// Channel to signal completion
	done := make(chan error, 1)

	// Read stderr in background
	go func() {
		for errScanner.Scan() {
			event := contracts.ServiceLogsSSEEvent{
				Type:      "error",
				Error:     errScanner.Text(),
				Container: opts.Container,
				Timestamp: time.Now(),
			}
			_ = writeSSEEvent(writer, event)
		}
	}()

	// Read stdout
	go func() {
		for scanner.Scan() {
			event := contracts.ServiceLogsSSEEvent{
				Type:      "log",
				Line:      scanner.Text(),
				Container: opts.Container,
				Timestamp: time.Now(),
			}
			if err := writeSSEEvent(writer, event); err != nil {
				done <- err
				return
			}
		}

		if err := scanner.Err(); err != nil {
			done <- err
			return
		}

		done <- nil
	}()

	// Wait for completion or error
	err = <-done
	_ = cmd.Process.Kill()

	// Send end event
	endEvent := contracts.ServiceLogsSSEEvent{
		Type:      "end",
		Timestamp: time.Now(),
	}
	_ = writeSSEEvent(writer, endEvent)

	return err
}

// writeSSEEvent writes an SSE event to the writer
func writeSSEEvent(w io.Writer, event contracts.ServiceLogsSSEEvent) error {
	// Marshal the event to JSON safely
	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal SSE event: %w", err)
	}

	// Write SSE format: "data: <json>\n\n"
	data := fmt.Sprintf("data: %s\n\n", jsonData)

	_, err = w.Write([]byte(data))
	if err != nil {
		return err
	}

	// Flush if writer supports it
	if flusher, ok := w.(interface{ Flush() }); ok {
		flusher.Flush()
	}

	return nil
}
