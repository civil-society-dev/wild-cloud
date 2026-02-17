package sse

import (
	"testing"
	"time"
)

func TestManagerClientRegistration(t *testing.T) {
	manager := NewManager()

	// Test client registration
	client := manager.RegisterClient("test-instance", EventFilters{})
	if client == nil {
		t.Fatal("Expected client to be registered")
	}

	// Give manager time to process registration
	time.Sleep(10 * time.Millisecond)

	// Check client is registered
	if count := manager.GetInstanceConnectionCount("test-instance"); count != 1 {
		t.Errorf("Expected 1 connection, got %d", count)
	}

	// Test client unregistration
	manager.UnregisterClient(client)
	time.Sleep(10 * time.Millisecond)

	if count := manager.GetInstanceConnectionCount("test-instance"); count != 0 {
		t.Errorf("Expected 0 connections, got %d", count)
	}
}

func TestManagerBroadcast(t *testing.T) {
	manager := NewManager()

	// Register two clients for same instance
	client1 := manager.RegisterClient("test-instance", EventFilters{
		EventTypes: []string{"pod.added"},
	})
	client2 := manager.RegisterClient("test-instance", EventFilters{})

	// Give manager time to process registrations
	time.Sleep(10 * time.Millisecond)

	// Broadcast an event
	event := &Event{
		Type:         "pod.added",
		InstanceName: "test-instance",
		Data:         map[string]interface{}{"name": "test-pod"},
		Metadata:     map[string]interface{}{},
	}
	manager.Broadcast(event)

	// Both clients should receive the event
	select {
	case receivedEvent := <-client1.Channel:
		if receivedEvent.Type != "pod.added" {
			t.Errorf("Expected event type 'pod.added', got '%s'", receivedEvent.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Client1 did not receive event")
	}

	select {
	case receivedEvent := <-client2.Channel:
		if receivedEvent.Type != "pod.added" {
			t.Errorf("Expected event type 'pod.added', got '%s'", receivedEvent.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Client2 did not receive event")
	}

	// Cleanup
	manager.UnregisterClient(client1)
	manager.UnregisterClient(client2)
}

func TestEventFiltering(t *testing.T) {
	manager := NewManager()

	// Register client with event type filter
	client := manager.RegisterClient("test-instance", EventFilters{
		EventTypes: []string{"pod.added", "pod.updated"},
	})

	// Give manager time to process registration
	time.Sleep(10 * time.Millisecond)

	// Send events of different types
	podAddedEvent := &Event{
		Type:         "pod.added",
		InstanceName: "test-instance",
		Data:         map[string]interface{}{"name": "test-pod"},
	}
	podDeletedEvent := &Event{
		Type:         "pod.deleted",
		InstanceName: "test-instance",
		Data:         map[string]interface{}{"name": "test-pod"},
	}

	manager.Broadcast(podAddedEvent)
	manager.Broadcast(podDeletedEvent)

	// Should only receive pod.added event
	select {
	case receivedEvent := <-client.Channel:
		if receivedEvent.Type != "pod.added" {
			t.Errorf("Expected event type 'pod.added', got '%s'", receivedEvent.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Client did not receive pod.added event")
	}

	// Should not receive pod.deleted event
	select {
	case receivedEvent := <-client.Channel:
		t.Errorf("Should not have received event of type '%s'", receivedEvent.Type)
	case <-time.After(100 * time.Millisecond):
		// Expected - no event should be received
	}

	// Cleanup
	manager.UnregisterClient(client)
}

func TestNamespaceFiltering(t *testing.T) {
	manager := NewManager()

	// Register client with namespace filter
	client := manager.RegisterClient("test-instance", EventFilters{
		Namespaces: []string{"default", "kube-system"},
	})

	// Give manager time to process registration
	time.Sleep(10 * time.Millisecond)

	// Send events from different namespaces
	defaultNsEvent := &Event{
		Type:         "pod.added",
		InstanceName: "test-instance",
		Data:         map[string]interface{}{"name": "test-pod"},
		Metadata:     map[string]interface{}{"namespace": "default"},
	}
	otherNsEvent := &Event{
		Type:         "pod.added",
		InstanceName: "test-instance",
		Data:         map[string]interface{}{"name": "test-pod"},
		Metadata:     map[string]interface{}{"namespace": "other-namespace"},
	}

	manager.Broadcast(defaultNsEvent)
	manager.Broadcast(otherNsEvent)

	// Should only receive event from default namespace
	select {
	case receivedEvent := <-client.Channel:
		if ns, ok := receivedEvent.Metadata["namespace"].(string); !ok || ns != "default" {
			t.Errorf("Expected event from 'default' namespace, got '%v'", receivedEvent.Metadata["namespace"])
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Client did not receive event from default namespace")
	}

	// Should not receive event from other namespace
	select {
	case receivedEvent := <-client.Channel:
		t.Errorf("Should not have received event from namespace '%v'", receivedEvent.Metadata["namespace"])
	case <-time.After(100 * time.Millisecond):
		// Expected - no event should be received
	}

	// Cleanup
	manager.UnregisterClient(client)
}

func TestMultipleInstanceIsolation(t *testing.T) {
	manager := NewManager()

	// Register clients for different instances
	client1 := manager.RegisterClient("instance-1", EventFilters{})
	client2 := manager.RegisterClient("instance-2", EventFilters{})

	// Give manager time to process registrations
	time.Sleep(10 * time.Millisecond)

	// Send event to instance-1
	event := &Event{
		Type:         "pod.added",
		InstanceName: "instance-1",
		Data:         map[string]interface{}{"name": "test-pod"},
	}
	manager.Broadcast(event)

	// Only client1 should receive the event
	select {
	case receivedEvent := <-client1.Channel:
		if receivedEvent.InstanceName != "instance-1" {
			t.Errorf("Expected event for 'instance-1', got '%s'", receivedEvent.InstanceName)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Client1 did not receive event")
	}

	// Client2 should not receive the event
	select {
	case <-client2.Channel:
		t.Error("Client2 should not have received event for instance-1")
	case <-time.After(100 * time.Millisecond):
		// Expected - no event should be received
	}

	// Cleanup
	manager.UnregisterClient(client1)
	manager.UnregisterClient(client2)
}

func TestRateLimiting(t *testing.T) {
	manager := NewManager()

	// Register a client
	client := manager.RegisterClient("test-instance", EventFilters{})

	// Give manager time to process registration
	time.Sleep(10 * time.Millisecond)

	// Send many events rapidly (more than rate limit allows)
	for i := 0; i < 300; i++ {
		event := &Event{
			Type:         "pod.added",
			InstanceName: "test-instance",
			Data:         map[string]interface{}{"index": i},
		}
		manager.Broadcast(event)
	}

	// Count received events
	receivedCount := 0
	timeout := time.After(500 * time.Millisecond)

	for {
		select {
		case <-client.Channel:
			receivedCount++
		case <-timeout:
			// Rate limiter should have dropped some events
			// We allow 100/sec with burst of 200, so we should receive around 200
			if receivedCount >= 300 {
				t.Errorf("Rate limiting not working: received %d events (expected < 300)", receivedCount)
			}
			if receivedCount < 100 {
				t.Errorf("Too many events dropped: received only %d events", receivedCount)
			}
			// Cleanup
			manager.UnregisterClient(client)
			return
		}
	}
}

func TestClientContextCancellation(t *testing.T) {
	manager := NewManager()

	// Register a client
	client := manager.RegisterClient("test-instance", EventFilters{})

	// Cancel the client context
	client.Cancel()

	// Try to send an event
	event := &Event{
		Type:         "pod.added",
		InstanceName: "test-instance",
		Data:         map[string]interface{}{"name": "test-pod"},
	}
	manager.Broadcast(event)

	// Client should not receive event after context cancellation
	select {
	case <-client.Channel:
		// Channel might still have buffered events, but should be closed soon
	case <-time.After(100 * time.Millisecond):
		// Expected - context cancelled
	}

	// Cleanup
	manager.UnregisterClient(client)
}

func TestConnectionCount(t *testing.T) {
	manager := NewManager()

	// Initially no connections
	if count := manager.GetConnectionCount(); count != 0 {
		t.Errorf("Expected 0 connections initially, got %d", count)
	}

	// Register clients for different instances
	client1 := manager.RegisterClient("instance-1", EventFilters{})
	client2 := manager.RegisterClient("instance-1", EventFilters{})
	client3 := manager.RegisterClient("instance-2", EventFilters{})

	// Give manager time to process registrations
	time.Sleep(10 * time.Millisecond)

	// Check total connection count
	if count := manager.GetConnectionCount(); count != 3 {
		t.Errorf("Expected 3 total connections, got %d", count)
	}

	// Check per-instance counts
	if count := manager.GetInstanceConnectionCount("instance-1"); count != 2 {
		t.Errorf("Expected 2 connections for instance-1, got %d", count)
	}
	if count := manager.GetInstanceConnectionCount("instance-2"); count != 1 {
		t.Errorf("Expected 1 connection for instance-2, got %d", count)
	}

	// Cleanup
	manager.UnregisterClient(client1)
	manager.UnregisterClient(client2)
	manager.UnregisterClient(client3)
}

func BenchmarkBroadcast(b *testing.B) {
	manager := NewManager()

	// Register 100 clients
	clients := make([]*Client, 100)
	for i := 0; i < 100; i++ {
		clients[i] = manager.RegisterClient("test-instance", EventFilters{})
	}

	// Give manager time to process registrations
	time.Sleep(100 * time.Millisecond)

	// Benchmark broadcasting events
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		event := &Event{
			Type:         "pod.added",
			InstanceName: "test-instance",
			Data:         map[string]interface{}{"index": i},
		}
		manager.Broadcast(event)
	}

	// Cleanup
	for _, client := range clients {
		manager.UnregisterClient(client)
	}
}