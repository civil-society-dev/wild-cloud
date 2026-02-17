package v1

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/wild-cloud/wild-central/daemon/internal/sse"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
)

// InstanceEventStream handles SSE connections for instance events
func (api *API) InstanceEventStream(w http.ResponseWriter, r *http.Request) {
	// 1. Extract instance name from URL
	vars := mux.Vars(r)
	instanceName := vars["name"]

	// 2. Validate instance exists
	if err := api.instance.ValidateInstance(instanceName); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
		return
	}

	// 3. Parse event filters from query parameters
	filters := sse.EventFilters{
		EventTypes: parseQueryList(r.URL.Query().Get("types")),
		Namespaces: parseQueryList(r.URL.Query().Get("namespaces")),
		Apps:       parseQueryList(r.URL.Query().Get("apps")),
	}

	// 4. Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// 5. Register client with SSE manager
	client := api.sseManager.RegisterClient(instanceName, filters)
	defer api.sseManager.UnregisterClient(client)

	// 6. Start watchers if needed (only once per instance)
	kubeconfigPath := tools.GetKubeconfigPath(api.dataDir, instanceName)
	talosconfigPath := tools.GetTalosconfigPath(api.dataDir, instanceName)
	configPath := tools.GetInstanceConfigPath(api.dataDir, instanceName)

	// Get control plane VIP for talos events
	nodeIP, err := api.config.GetConfigValue(configPath, "cluster.nodes.control.vip")
	if err != nil {
		// Default to empty string if not found - talos events will be skipped
		nodeIP = ""
		log.Printf("Control plane VIP not found for instance %s, Talos events will be disabled", instanceName)
	}

	// Start watchers for this instance if not already running
	if err := api.watcherManager.StartWatchers(instanceName, kubeconfigPath, talosconfigPath, nodeIP); err != nil {
		log.Printf("Failed to start watchers for instance %s: %v", instanceName, err)
		// Continue anyway - client might still receive events from other sources
	}

	// 7. Send initial connected event
	connectedEvent := &sse.Event{
		Type:         "connected",
		InstanceName: instanceName,
		Data: map[string]interface{}{
			"message": "Successfully connected to event stream",
			"filters": filters,
		},
	}
	if err := sendSSEEvent(w, connectedEvent); err != nil {
		log.Printf("Failed to send connected event: %v", err)
		return
	}

	// 8. Flush immediately to establish connection
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// 9. Send heartbeat and handle events
	heartbeatInterval := 30 // seconds
	heartbeatTicker := time.NewTicker(time.Duration(heartbeatInterval) * time.Second)
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-client.Context.Done():
			// Client disconnected
			return

		case <-r.Context().Done():
			// Request cancelled
			return

		case event := <-client.Channel:
			// Send event to client
			if err := sendSSEEvent(w, event); err != nil {
				log.Printf("Failed to send event: %v", err)
				return
			}

			// Flush after each event
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}

		case <-heartbeatTicker.C:
			// Send heartbeat to keep connection alive
			heartbeatEvent := &sse.Event{
				Type:         "heartbeat",
				InstanceName: instanceName,
				Data: map[string]interface{}{
					"timestamp": time.Now().Unix(),
				},
			}
			if err := sendSSEEvent(w, heartbeatEvent); err != nil {
				log.Printf("Failed to send heartbeat: %v", err)
				return
			}

			// Flush after heartbeat
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}
}

// sendSSEEvent writes an SSE event to the response writer
func sendSSEEvent(w http.ResponseWriter, event *sse.Event) error {
	// Set event ID
	if _, err := fmt.Fprintf(w, "id: %s\n", event.ID); err != nil {
		return err
	}

	// Set event type
	if _, err := fmt.Fprintf(w, "event: %s\n", event.Type); err != nil {
		return err
	}

	// Set retry interval (in milliseconds)
	if _, err := fmt.Fprintf(w, "retry: 5000\n"); err != nil {
		return err
	}

	// Marshal event data to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Write data field
	if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
		return err
	}

	return nil
}

// GlobalEventStream handles SSE connections for ALL events (global and instance-specific)
func (api *API) GlobalEventStream(w http.ResponseWriter, r *http.Request) {
	// 1. Parse event filters from query parameters
	filters := sse.EventFilters{
		EventTypes: parseQueryList(r.URL.Query().Get("types")),
		Namespaces: parseQueryList(r.URL.Query().Get("namespaces")),
		Apps:       parseQueryList(r.URL.Query().Get("apps")),
	}

	// 2. Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// 3. Register client with SSE manager
	// Use "*" as a special instance name to receive ALL events
	client := api.sseManager.RegisterClient("*", filters)
	defer api.sseManager.UnregisterClient(client)

	// 4. Send initial connected event
	connectedEvent := &sse.Event{
		Type:         "connected",
		InstanceName: "global",
		Data: map[string]interface{}{
			"message": "Successfully connected to global event stream",
			"filters": filters,
		},
	}
	if err := sendSSEEvent(w, connectedEvent); err != nil {
		log.Printf("Failed to send connected event: %v", err)
		return
	}

	// 5. Flush immediately to establish connection
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// 6. Send heartbeat and handle events
	heartbeatInterval := 30 // seconds
	heartbeatTicker := time.NewTicker(time.Duration(heartbeatInterval) * time.Second)
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-client.Context.Done():
			// Client disconnected
			return

		case <-r.Context().Done():
			// Request cancelled
			return

		case event := <-client.Channel:
			// Send event to client
			if err := sendSSEEvent(w, event); err != nil {
				log.Printf("Failed to send event: %v", err)
				return
			}

			// Flush after each event
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}

		case <-heartbeatTicker.C:
			// Send heartbeat to keep connection alive
			heartbeatEvent := &sse.Event{
				Type:         "heartbeat",
				InstanceName: "global",
				Data: map[string]interface{}{
					"timestamp": time.Now().Unix(),
				},
			}
			if err := sendSSEEvent(w, heartbeatEvent); err != nil {
				log.Printf("Failed to send heartbeat: %v", err)
				return
			}

			// Flush after heartbeat
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}
}

// parseQueryList parses comma-separated query parameter into slice
func parseQueryList(param string) []string {
	if param == "" {
		return nil
	}

	parts := strings.Split(param, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}