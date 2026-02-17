package sse

import (
	"fmt"
	"testing"
	"time"
)

func TestKubectlWatcherJSONParsing(t *testing.T) {
	manager := NewManager()
	watcher := &KubectlWatcher{
		instanceName: "test-instance",
		kubeconfig:   "/tmp/test-kubeconfig",
		sseManager:   manager,
	}

	// Test valid pod JSON parsing
	validPodJSON := `{
		"type": "ADDED",
		"object": {
			"apiVersion": "v1",
			"kind": "Pod",
			"metadata": {
				"name": "test-pod",
				"namespace": "default",
				"labels": {
					"app": "test-app"
				}
			},
			"status": {
				"phase": "Running",
				"conditions": []
			}
		}
	}`

	// Register a client to receive events
	client := manager.RegisterClient("test-instance", EventFilters{})
	defer manager.UnregisterClient(client)

	// Parse the JSON
	watcher.parsePodEvent([]byte(validPodJSON), "test-instance")

	// Should receive parsed event
	select {
	case event := <-client.Channel:
		if event.Type != "pod.added" {
			t.Errorf("Expected event type 'pod.added', got '%s'", event.Type)
		}
		if podData, ok := event.Data.(map[string]interface{}); ok {
			if podData["name"] != "test-pod" {
				t.Errorf("Expected pod name 'test-pod', got '%v'", podData["name"])
			}
			if podData["namespace"] != "default" {
				t.Errorf("Expected namespace 'default', got '%v'", podData["namespace"])
			}
		} else {
			t.Error("Event data is not a map")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Did not receive pod event")
	}
}

func TestKubectlWatcherEventTypeMapping(t *testing.T) {
	manager := NewManager()
	watcher := &KubectlWatcher{
		instanceName: "test-instance",
		kubeconfig:   "/tmp/test-kubeconfig",
		sseManager:   manager,
	}

	testCases := []struct {
		watchType    string
		expectedType string
	}{
		{"ADDED", "pod.added"},
		{"MODIFIED", "pod.updated"},
		{"DELETED", "pod.deleted"},
	}

	for _, tc := range testCases {
		t.Run(tc.watchType, func(t *testing.T) {
			// Register a client
			client := manager.RegisterClient("test-instance", EventFilters{})
			defer manager.UnregisterClient(client)

			podJSON := fmt.Sprintf(`{
				"type": "%s",
				"object": {
					"kind": "Pod",
					"metadata": {
						"name": "test-pod",
						"namespace": "default"
					}
				}
			}`, tc.watchType)

			// Parse the event
			watcher.parsePodEvent([]byte(podJSON), "test-instance")

			// Check received event type
			select {
			case event := <-client.Channel:
				if event.Type != tc.expectedType {
					t.Errorf("Expected event type '%s', got '%s'", tc.expectedType, event.Type)
				}
			case <-time.After(100 * time.Millisecond):
				t.Error("Did not receive event")
			}
		})
	}
}

func TestDeploymentEventParsing(t *testing.T) {
	manager := NewManager()
	watcher := &KubectlWatcher{
		instanceName: "test-instance",
		kubeconfig:   "/tmp/test-kubeconfig",
		sseManager:   manager,
	}

	deploymentJSON := `{
		"type": "MODIFIED",
		"object": {
			"apiVersion": "apps/v1",
			"kind": "Deployment",
			"metadata": {
				"name": "test-deployment",
				"namespace": "default"
			},
			"spec": {
				"replicas": 3
			},
			"status": {
				"replicas": 3,
				"readyReplicas": 2,
				"availableReplicas": 2
			}
		}
	}`

	// Register a client
	client := manager.RegisterClient("test-instance", EventFilters{})
	defer manager.UnregisterClient(client)

	// Parse deployment event
	watcher.parseDeploymentEvent([]byte(deploymentJSON), "test-instance")

	// Check received event
	select {
	case event := <-client.Channel:
		if event.Type != "deployment.updated" {
			t.Errorf("Expected event type 'deployment.updated', got '%s'", event.Type)
		}
		if deployData, ok := event.Data.(map[string]interface{}); ok {
			if deployData["name"] != "test-deployment" {
				t.Errorf("Expected deployment name 'test-deployment', got '%v'", deployData["name"])
			}
			if deployData["readyReplicas"] != float64(2) {
				t.Errorf("Expected 2 ready replicas, got '%v'", deployData["readyReplicas"])
			}
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Did not receive deployment event")
	}
}

func TestServiceEventParsing(t *testing.T) {
	manager := NewManager()
	watcher := &KubectlWatcher{
		instanceName: "test-instance",
		kubeconfig:   "/tmp/test-kubeconfig",
		sseManager:   manager,
	}

	serviceJSON := `{
		"type": "ADDED",
		"object": {
			"apiVersion": "v1",
			"kind": "Service",
			"metadata": {
				"name": "test-service",
				"namespace": "default"
			},
			"spec": {
				"type": "LoadBalancer",
				"clusterIP": "10.96.0.1",
				"ports": [
					{
						"port": 80,
						"targetPort": 8080,
						"protocol": "TCP"
					}
				]
			},
			"status": {
				"loadBalancer": {
					"ingress": [
						{
							"ip": "192.168.1.100"
						}
					]
				}
			}
		}
	}`

	// Register a client
	client := manager.RegisterClient("test-instance", EventFilters{})
	defer manager.UnregisterClient(client)

	// Parse service event
	watcher.parseServiceEvent([]byte(serviceJSON), "test-instance")

	// Check received event
	select {
	case event := <-client.Channel:
		if event.Type != "service.added" {
			t.Errorf("Expected event type 'service.added', got '%s'", event.Type)
		}
		if serviceData, ok := event.Data.(map[string]interface{}); ok {
			if serviceData["name"] != "test-service" {
				t.Errorf("Expected service name 'test-service', got '%v'", serviceData["name"])
			}
			if serviceData["type"] != "LoadBalancer" {
				t.Errorf("Expected service type 'LoadBalancer', got '%v'", serviceData["type"])
			}
			if serviceData["externalIP"] != "192.168.1.100" {
				t.Errorf("Expected external IP '192.168.1.100', got '%v'", serviceData["externalIP"])
			}
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Did not receive service event")
	}
}

// TestTalosEventParsing is commented out as TalosWatcher doesn't expose parsing methods
// The actual parsing happens internally in the watchEvents method
/*
func TestTalosEventParsing(t *testing.T) {
	manager := NewManager()
	watcher := &TalosWatcher{
		instanceName: "test-instance",
		talosconfig:  "/tmp/test-talosconfig",
		nodeIP:       "192.168.1.10",
		sseManager:   manager,
	}

	// Test machine config event
	machineConfigEvent := `seq:task	message:update successful	node:192.168.1.10	type:MachineConfig`

	// Register a client
	client := manager.RegisterClient("test-instance", EventFilters{})
	defer manager.UnregisterClient(client)

	// Parse talos event
	watcher.parseTalosEventLine(machineConfigEvent)

	// Check received event
	select {
	case event := <-client.Channel:
		if event.Type != "talos.MachineConfig" {
			t.Errorf("Expected event type 'talos.MachineConfig', got '%s'", event.Type)
		}
		if talosData, ok := event.Data.(map[string]interface{}); ok {
			if talosData["message"] != "update successful" {
				t.Errorf("Expected message 'update successful', got '%v'", talosData["message"])
			}
			if talosData["node"] != "192.168.1.10" {
				t.Errorf("Expected node '192.168.1.10', got '%v'", talosData["node"])
			}
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Did not receive talos event")
	}
}
*/

func TestWatcherManagerLifecycle(t *testing.T) {
	sseManager := NewManager()
	watcherManager := NewWatcherManager(sseManager)

	// Test starting watchers for an instance
	err := watcherManager.StartWatchers("test-instance", "/tmp/kubeconfig", "/tmp/talosconfig", "192.168.1.10")
	if err != nil {
		// This will likely fail because the configs don't exist, but we can test the manager state
		t.Logf("Expected error starting watchers with non-existent configs: %v", err)
	}

	// Check if instance is registered (even if watchers failed to start)
	watcherManager.mu.RLock()
	_, exists := watcherManager.watchers["test-instance"]
	watcherManager.mu.RUnlock()

	if !exists {
		t.Error("Instance should be registered even if watchers fail to start")
	}

	// Test stopping watchers
	watcherManager.StopWatchers("test-instance")

	// Check instance is removed
	watcherManager.mu.RLock()
	_, exists = watcherManager.watchers["test-instance"]
	watcherManager.mu.RUnlock()

	if exists {
		t.Error("Instance should be removed after stopping watchers")
	}
}

func TestWatcherManagerMultipleInstances(t *testing.T) {
	sseManager := NewManager()
	watcherManager := NewWatcherManager(sseManager)

	// Start watchers for multiple instances
	_ = watcherManager.StartWatchers("instance-1", "/tmp/kubeconfig1", "/tmp/talosconfig1", "192.168.1.11")
	_ = watcherManager.StartWatchers("instance-2", "/tmp/kubeconfig2", "/tmp/talosconfig2", "192.168.1.12")

	// Check both instances are registered
	watcherManager.mu.RLock()
	count := len(watcherManager.watchers)
	watcherManager.mu.RUnlock()

	if count != 2 {
		t.Errorf("Expected 2 instances registered, got %d", count)
	}

	// Stop watchers for each instance
	watcherManager.StopWatchers("instance-1")
	watcherManager.StopWatchers("instance-2")

	// Check all instances are removed
	watcherManager.mu.RLock()
	count = len(watcherManager.watchers)
	watcherManager.mu.RUnlock()

	if count != 0 {
		t.Errorf("Expected 0 instances after stopping all, got %d", count)
	}
}

func TestInvalidJSONHandling(t *testing.T) {
	manager := NewManager()
	watcher := &KubectlWatcher{
		instanceName: "test-instance",
		kubeconfig:   "/tmp/test-kubeconfig",
		sseManager:   manager,
	}

	// Register a client
	client := manager.RegisterClient("test-instance", EventFilters{})
	defer manager.UnregisterClient(client)

	// Try parsing invalid JSON
	invalidJSON := `{invalid json}`
	watcher.parsePodEvent([]byte(invalidJSON), "test-instance")

	// Should not receive any event
	select {
	case <-client.Channel:
		t.Error("Should not receive event for invalid JSON")
	case <-time.After(50 * time.Millisecond):
		// Expected - no event for invalid JSON
	}
}

func TestEmptyEventHandling(t *testing.T) {
	manager := NewManager()
	watcher := &KubectlWatcher{
		instanceName: "test-instance",
		kubeconfig:   "/tmp/test-kubeconfig",
		sseManager:   manager,
	}

	// Register a client
	client := manager.RegisterClient("test-instance", EventFilters{})
	defer manager.UnregisterClient(client)

	// Try parsing empty JSON
	emptyJSON := `{}`
	watcher.parsePodEvent([]byte(emptyJSON), "test-instance")

	// Should not crash, might not produce event
	select {
	case <-client.Channel:
		// If we get an event, make sure it doesn't crash
		t.Log("Received event for empty JSON (acceptable)")
	case <-time.After(50 * time.Millisecond):
		// Also acceptable - no event for empty JSON
	}
}

func BenchmarkJSONParsing(b *testing.B) {
	manager := NewManager()
	watcher := &KubectlWatcher{
		instanceName: "test-instance",
		kubeconfig:   "/tmp/test-kubeconfig",
		sseManager:   manager,
	}

	podJSON := `{
		"type": "ADDED",
		"object": {
			"apiVersion": "v1",
			"kind": "Pod",
			"metadata": {
				"name": "test-pod",
				"namespace": "default",
				"labels": {
					"app": "test-app"
				}
			},
			"status": {
				"phase": "Running"
			}
		}
	}`

	// Register a client to consume events
	client := manager.RegisterClient("test-instance", EventFilters{})
	defer manager.UnregisterClient(client)

	// Consume events in background
	go func() {
		for range client.Channel {
			// Consume events
		}
	}()

	// Benchmark JSON parsing and event creation
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		watcher.parsePodEvent([]byte(podJSON), "test-instance")
	}
}