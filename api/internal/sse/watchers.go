package sse

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// WatcherManager manages kubectl and talos watchers per instance
type WatcherManager struct {
	watchers   map[string]*InstanceWatchers
	mu         sync.RWMutex
	sseManager *Manager
}

// InstanceWatchers holds all watchers for a single instance
type InstanceWatchers struct {
	kubectl *KubectlWatcher
	talos   *TalosWatcher
}

// NewWatcherManager creates a new watcher manager
func NewWatcherManager(sseManager *Manager) *WatcherManager {
	return &WatcherManager{
		watchers:   make(map[string]*InstanceWatchers),
		sseManager: sseManager,
	}
}

// StartWatchers starts watchers for an instance
func (wm *WatcherManager) StartWatchers(instanceName, kubeconfig, talosconfig, nodeIP string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// Stop existing watchers if any
	if existing, ok := wm.watchers[instanceName]; ok {
		existing.kubectl.Stop()
		if existing.talos != nil {
			existing.talos.Stop()
		}
	}

	// Start kubectl watcher
	kubectlWatcher := NewKubectlWatcher(instanceName, kubeconfig, wm.sseManager)
	if err := kubectlWatcher.Start(); err != nil {
		return fmt.Errorf("failed to start kubectl watcher: %w", err)
	}

	// Start talos watcher (optional, can be nil if not configured)
	var talosWatcher *TalosWatcher
	if talosconfig != "" && nodeIP != "" {
		talosWatcher = NewTalosWatcher(instanceName, talosconfig, nodeIP, wm.sseManager)
		if err := talosWatcher.Start(); err != nil {
			kubectlWatcher.Stop()
			return fmt.Errorf("failed to start talos watcher: %w", err)
		}
	}

	wm.watchers[instanceName] = &InstanceWatchers{
		kubectl: kubectlWatcher,
		talos:   talosWatcher,
	}

	return nil
}

// StopWatchers stops watchers for an instance
func (wm *WatcherManager) StopWatchers(instanceName string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if watchers, ok := wm.watchers[instanceName]; ok {
		watchers.kubectl.Stop()
		if watchers.talos != nil {
			watchers.talos.Stop()
		}
		delete(wm.watchers, instanceName)
	}
}

// KubectlWatcher watches Kubernetes resources using kubectl --watch
type KubectlWatcher struct {
	instanceName string
	kubeconfig   string
	sseManager   *Manager
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// NewKubectlWatcher creates a new kubectl watcher
func NewKubectlWatcher(instanceName string, kubeconfig string, manager *Manager) *KubectlWatcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &KubectlWatcher{
		instanceName: instanceName,
		kubeconfig:   kubeconfig,
		sseManager:   manager,
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start begins watching Kubernetes resources
func (w *KubectlWatcher) Start() error {
	// Watch pods
	w.wg.Add(1)
	go w.watchResource("pods", w.parsePodEvent)

	// Watch deployments
	w.wg.Add(1)
	go w.watchResource("deployments", w.parseDeploymentEvent)

	// Watch services
	w.wg.Add(1)
	go w.watchResource("services", w.parseServiceEvent)

	log.Printf("SSE: Started kubectl watchers for instance %s", w.instanceName)
	return nil
}

// watchResource watches a specific Kubernetes resource type
func (w *KubectlWatcher) watchResource(resourceType string, parser func([]byte, string)) {
	defer w.wg.Done()

	for {
		select {
		case <-w.ctx.Done():
			return
		default:
		}

		// Start kubectl watch command
		cmd := exec.CommandContext(
			w.ctx,
			"kubectl",
			"--kubeconfig", w.kubeconfig,
			"get", resourceType,
			"--all-namespaces",
			"--watch",
			"-o", "json",
		)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Printf("SSE: Failed to create stdout pipe for %s watch: %v", resourceType, err)
			w.handleWatchError(resourceType)
			continue
		}

		if err := cmd.Start(); err != nil {
			log.Printf("SSE: Failed to start %s watch: %v", resourceType, err)
			w.handleWatchError(resourceType)
			continue
		}

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			parser([]byte(line), resourceType)
		}

		if err := scanner.Err(); err != nil {
			log.Printf("SSE: %s watch scanner error: %v", resourceType, err)
		}

		cmd.Wait()

		// If context not cancelled, restart after a delay
		if w.ctx.Err() == nil {
			log.Printf("SSE: Restarting %s watcher for instance %s", resourceType, w.instanceName)
			time.Sleep(5 * time.Second)
		}
	}
}

// parsePodEvent parses pod watch events
func (w *KubectlWatcher) parsePodEvent(data []byte, resourceType string) {
	var event struct {
		Type   string `json:"type"`   // ADDED, MODIFIED, DELETED
		Object struct {
			Metadata struct {
				Name      string            `json:"name"`
				Namespace string            `json:"namespace"`
				Labels    map[string]string `json:"labels"`
			} `json:"metadata"`
			Status struct {
				Phase      string `json:"phase"`
				Conditions []struct {
					Type   string `json:"type"`
					Status string `json:"status"`
				} `json:"conditions"`
				ContainerStatuses []struct {
					Name  string `json:"name"`
					Ready bool   `json:"ready"`
					State struct {
						Running    interface{} `json:"running"`
						Waiting    interface{} `json:"waiting"`
						Terminated interface{} `json:"terminated"`
					} `json:"state"`
				} `json:"containerStatuses"`
			} `json:"status"`
		} `json:"object"`
	}

	// Try parsing as watch event first
	if err := json.Unmarshal(data, &event); err == nil && event.Type != "" {
		eventType := ""
		switch event.Type {
		case "ADDED":
			eventType = "pod:added"
		case "MODIFIED":
			eventType = "pod:modified"
		case "DELETED":
			eventType = "pod:deleted"
		default:
			return
		}

		w.sseManager.Broadcast(&Event{
			Type:         eventType,
			InstanceName: w.instanceName,
			Data: map[string]interface{}{
				"name":      event.Object.Metadata.Name,
				"namespace": event.Object.Metadata.Namespace,
				"phase":     event.Object.Status.Phase,
				"ready":     w.isPodReady(event.Object.Status.ContainerStatuses),
			},
			Metadata: map[string]interface{}{
				"namespace": event.Object.Metadata.Namespace,
				"app":       event.Object.Metadata.Labels["app"],
			},
		})
		return
	}

	// Try parsing as initial pod object (from initial list)
	var pod struct {
		Metadata struct {
			Name      string            `json:"name"`
			Namespace string            `json:"namespace"`
			Labels    map[string]string `json:"labels"`
		} `json:"metadata"`
		Status struct {
			Phase             string `json:"phase"`
			ContainerStatuses []struct {
				Ready bool `json:"ready"`
			} `json:"containerStatuses"`
		} `json:"status"`
	}

	if err := json.Unmarshal(data, &pod); err == nil && pod.Metadata.Name != "" {
		w.sseManager.Broadcast(&Event{
			Type:         "pod:added",
			InstanceName: w.instanceName,
			Data: map[string]interface{}{
				"name":      pod.Metadata.Name,
				"namespace": pod.Metadata.Namespace,
				"phase":     pod.Status.Phase,
				"ready":     w.isPodReady(pod.Status.ContainerStatuses),
			},
			Metadata: map[string]interface{}{
				"namespace": pod.Metadata.Namespace,
				"app":       pod.Metadata.Labels["app"],
			},
		})
	}
}

// parseDeploymentEvent parses deployment watch events
func (w *KubectlWatcher) parseDeploymentEvent(data []byte, resourceType string) {
	var deployment struct {
		Type   string `json:"type"`
		Object struct {
			Metadata struct {
				Name      string            `json:"name"`
				Namespace string            `json:"namespace"`
				Labels    map[string]string `json:"labels"`
			} `json:"metadata"`
			Status struct {
				Replicas          int32 `json:"replicas"`
				UpdatedReplicas   int32 `json:"updatedReplicas"`
				ReadyReplicas     int32 `json:"readyReplicas"`
				AvailableReplicas int32 `json:"availableReplicas"`
			} `json:"status"`
		} `json:"object"`
		// Also handle direct deployment objects
		Metadata struct {
			Name      string            `json:"name"`
			Namespace string            `json:"namespace"`
			Labels    map[string]string `json:"labels"`
		} `json:"metadata"`
		Status struct {
			Replicas          int32 `json:"replicas"`
			UpdatedReplicas   int32 `json:"updatedReplicas"`
			ReadyReplicas     int32 `json:"readyReplicas"`
			AvailableReplicas int32 `json:"availableReplicas"`
		} `json:"status"`
	}

	if err := json.Unmarshal(data, &deployment); err != nil {
		return
	}

	// Determine if it's a watch event or direct object
	var name, namespace, eventType string
	var labels map[string]string
	var status struct {
		Replicas      int32
		ReadyReplicas int32
	}

	if deployment.Type != "" {
		// Watch event
		switch deployment.Type {
		case "ADDED":
			eventType = "deployment:added"
		case "MODIFIED":
			eventType = "deployment:modified"
		case "DELETED":
			eventType = "deployment:deleted"
		default:
			return
		}
		name = deployment.Object.Metadata.Name
		namespace = deployment.Object.Metadata.Namespace
		labels = deployment.Object.Metadata.Labels
		status.Replicas = deployment.Object.Status.Replicas
		status.ReadyReplicas = deployment.Object.Status.ReadyReplicas
	} else if deployment.Metadata.Name != "" {
		// Direct object (initial list)
		eventType = "deployment:added"
		name = deployment.Metadata.Name
		namespace = deployment.Metadata.Namespace
		labels = deployment.Metadata.Labels
		status.Replicas = deployment.Status.Replicas
		status.ReadyReplicas = deployment.Status.ReadyReplicas
	} else {
		return
	}

	w.sseManager.Broadcast(&Event{
		Type:         eventType,
		InstanceName: w.instanceName,
		Data: map[string]interface{}{
			"name":          name,
			"namespace":     namespace,
			"replicas":      status.Replicas,
			"readyReplicas": status.ReadyReplicas,
			"ready":         status.Replicas == status.ReadyReplicas && status.Replicas > 0,
		},
		Metadata: map[string]interface{}{
			"namespace": namespace,
			"app":       labels["app"],
		},
	})
}

// parseServiceEvent parses service watch events
func (w *KubectlWatcher) parseServiceEvent(data []byte, resourceType string) {
	var service struct {
		Type   string `json:"type"`
		Object struct {
			Metadata struct {
				Name      string            `json:"name"`
				Namespace string            `json:"namespace"`
				Labels    map[string]string `json:"labels"`
			} `json:"metadata"`
			Spec struct {
				Type      string `json:"type"`
				ClusterIP string `json:"clusterIP"`
				Ports     []struct {
					Port int32 `json:"port"`
				} `json:"ports"`
			} `json:"spec"`
		} `json:"object"`
		// Also handle direct service objects
		Metadata struct {
			Name      string            `json:"name"`
			Namespace string            `json:"namespace"`
			Labels    map[string]string `json:"labels"`
		} `json:"metadata"`
		Spec struct {
			Type      string `json:"type"`
			ClusterIP string `json:"clusterIP"`
			Ports     []struct {
				Port int32 `json:"port"`
			} `json:"ports"`
		} `json:"spec"`
	}

	if err := json.Unmarshal(data, &service); err != nil {
		return
	}

	// Determine if it's a watch event or direct object
	var name, namespace, serviceType, eventType string
	var labels map[string]string

	if service.Type != "" {
		// Watch event
		switch service.Type {
		case "ADDED":
			eventType = "service:added"
		case "MODIFIED":
			eventType = "service:modified"
		case "DELETED":
			eventType = "service:deleted"
		default:
			return
		}
		name = service.Object.Metadata.Name
		namespace = service.Object.Metadata.Namespace
		labels = service.Object.Metadata.Labels
		serviceType = service.Object.Spec.Type
	} else if service.Metadata.Name != "" {
		// Direct object (initial list)
		eventType = "service:added"
		name = service.Metadata.Name
		namespace = service.Metadata.Namespace
		labels = service.Metadata.Labels
		serviceType = service.Spec.Type
	} else {
		return
	}

	w.sseManager.Broadcast(&Event{
		Type:         eventType,
		InstanceName: w.instanceName,
		Data: map[string]interface{}{
			"name":      name,
			"namespace": namespace,
			"type":      serviceType,
		},
		Metadata: map[string]interface{}{
			"namespace": namespace,
			"app":       labels["app"],
		},
	})
}

// isPodReady checks if all containers in a pod are ready
func (w *KubectlWatcher) isPodReady(containerStatuses interface{}) bool {
	switch statuses := containerStatuses.(type) {
	case []struct {
		Name  string `json:"name"`
		Ready bool   `json:"ready"`
		State struct {
			Running    interface{} `json:"running"`
			Waiting    interface{} `json:"waiting"`
			Terminated interface{} `json:"terminated"`
		} `json:"state"`
	}:
		if len(statuses) == 0 {
			return false
		}
		for _, status := range statuses {
			if !status.Ready {
				return false
			}
		}
		return true
	case []struct {
		Ready bool `json:"ready"`
	}:
		if len(statuses) == 0 {
			return false
		}
		for _, status := range statuses {
			if !status.Ready {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// handleWatchError handles errors in watching resources
func (w *KubectlWatcher) handleWatchError(resourceType string) {
	// Send error event
	w.sseManager.Broadcast(&Event{
		Type:         "k8s:watch:error",
		InstanceName: w.instanceName,
		Data: map[string]interface{}{
			"resource": resourceType,
			"error":    "Watch process failed, restarting",
		},
	})
}

// Stop stops the watcher
func (w *KubectlWatcher) Stop() {
	w.cancel()
	w.wg.Wait()
	log.Printf("SSE: Stopped kubectl watchers for instance %s", w.instanceName)
}

// TalosWatcher watches Talos events using talosctl
type TalosWatcher struct {
	instanceName string
	talosconfig  string
	nodeIP       string
	sseManager   *Manager
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewTalosWatcher creates a new Talos watcher
func NewTalosWatcher(instanceName, talosconfig, nodeIP string, manager *Manager) *TalosWatcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &TalosWatcher{
		instanceName: instanceName,
		talosconfig:  talosconfig,
		nodeIP:       nodeIP,
		sseManager:   manager,
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start begins watching Talos events
func (w *TalosWatcher) Start() error {
	go w.watchEvents()
	log.Printf("SSE: Started talos watcher for instance %s", w.instanceName)
	return nil
}

// watchEvents watches Talos events
func (w *TalosWatcher) watchEvents() {
	for {
		select {
		case <-w.ctx.Done():
			return
		default:
		}

		cmd := exec.CommandContext(
			w.ctx,
			"talosctl",
			"--talosconfig", w.talosconfig,
			"--nodes", w.nodeIP,
			"events",
			"--tail", "0", // Don't show historical events
			"--follow",
		)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Printf("SSE: Failed to create stdout pipe for Talos events: %v", err)
			time.Sleep(10 * time.Second)
			continue
		}

		if err := cmd.Start(); err != nil {
			log.Printf("SSE: Failed to start Talos event watch: %v", err)
			time.Sleep(10 * time.Second)
			continue
		}

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			// Parse Talos event output
			// Talos events typically look like:
			// 172.20.0.2: time.syncd: 2024-01-15T10:30:45.123Z: NTP time sync successful
			if strings.Contains(line, "Service") {
				w.sseManager.Broadcast(&Event{
					Type:         "talos:service:status",
					InstanceName: w.instanceName,
					Data: map[string]interface{}{
						"node":    w.nodeIP,
						"message": line,
					},
				})
			} else if strings.Contains(line, "Health") || strings.Contains(line, "health") {
				w.sseManager.Broadcast(&Event{
					Type:         "talos:node:health",
					InstanceName: w.instanceName,
					Data: map[string]interface{}{
						"node":    w.nodeIP,
						"message": line,
					},
				})
			}
		}

		cmd.Wait()

		// If context not cancelled, restart after a delay
		if w.ctx.Err() == nil {
			log.Printf("SSE: Restarting talos watcher for instance %s", w.instanceName)
			time.Sleep(10 * time.Second)
		}
	}
}

// Stop stops the watcher
func (w *TalosWatcher) Stop() {
	w.cancel()
	log.Printf("SSE: Stopped talos watcher for instance %s", w.instanceName)
}