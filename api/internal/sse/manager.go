package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

// Event represents an SSE event
type Event struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	InstanceName string                 `json:"instanceName"`
	Timestamp    time.Time              `json:"timestamp"`
	Data         interface{}            `json:"data"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Client represents an SSE client connection
type Client struct {
	ID           string
	InstanceName string
	Channel      chan *Event
	Filters      EventFilters
	Context      context.Context
	Cancel       context.CancelFunc
}

// EventFilters for client-specific filtering
type EventFilters struct {
	EventTypes []string `json:"eventTypes,omitempty"`
	Namespaces []string `json:"namespaces,omitempty"`
	Apps       []string `json:"apps,omitempty"`
}

// Manager manages all SSE connections
type Manager struct {
	clients      map[string]map[string]*Client // instanceName -> clientID -> Client
	register     chan *Client
	unregister   chan *Client
	broadcast    chan *Event
	mu           sync.RWMutex
	rateLimiters map[string]*rate.Limiter
}

// NewManager creates a new SSE manager
func NewManager() *Manager {
	m := &Manager{
		clients:      make(map[string]map[string]*Client),
		register:     make(chan *Client, 100),
		unregister:   make(chan *Client, 100),
		broadcast:    make(chan *Event, 1000),
		rateLimiters: make(map[string]*rate.Limiter),
	}
	go m.run()
	return m
}

// run processes client registration and event broadcasting
func (m *Manager) run() {
	for {
		select {
		case client := <-m.register:
			m.mu.Lock()
			if m.clients[client.InstanceName] == nil {
				m.clients[client.InstanceName] = make(map[string]*Client)
			}
			m.clients[client.InstanceName][client.ID] = client
			m.mu.Unlock()
			log.Printf("SSE: Client %s registered for instance %s", client.ID, client.InstanceName)

		case client := <-m.unregister:
			m.mu.Lock()
			if clients, ok := m.clients[client.InstanceName]; ok {
				delete(clients, client.ID)
				if len(clients) == 0 {
					delete(m.clients, client.InstanceName)
				}
			}
			close(client.Channel)
			m.mu.Unlock()
			log.Printf("SSE: Client %s unregistered", client.ID)

		case event := <-m.broadcast:
			m.mu.RLock()
			// Send to clients registered for this specific instance
			instanceClients := m.clients[event.InstanceName]
			// Also send to clients registered with wildcard "*" to receive all events
			globalClients := m.clients["*"]
			m.mu.RUnlock()

			// Send to instance-specific clients
			for _, client := range instanceClients {
				if m.shouldSendToClient(event, client) {
					select {
					case client.Channel <- event:
					default:
						// Client channel full, skip
						log.Printf("SSE: Client %s channel full, skipping event", client.ID)
					}
				}
			}

			// Send to global clients (registered with "*")
			for _, client := range globalClients {
				if m.shouldSendToClient(event, client) {
					select {
					case client.Channel <- event:
					default:
						// Client channel full, skip
						log.Printf("SSE: Client %s channel full, skipping event", client.ID)
					}
				}
			}
		}
	}
}

// shouldSendToClient checks if event matches client filters
func (m *Manager) shouldSendToClient(event *Event, client *Client) bool {
	// Rate limiting per client
	limiter := m.getRateLimiter(client.ID)
	if !limiter.Allow() {
		return false
	}

	// Check event type filter
	if len(client.Filters.EventTypes) > 0 {
		matched := false
		for _, eventType := range client.Filters.EventTypes {
			if event.Type == eventType {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check namespace filter for k8s events
	if len(client.Filters.Namespaces) > 0 {
		if namespace, ok := event.Metadata["namespace"].(string); ok {
			matched := false
			for _, ns := range client.Filters.Namespaces {
				if namespace == ns {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		}
	}

	// Check app filter
	if len(client.Filters.Apps) > 0 {
		if appName, ok := event.Metadata["app"].(string); ok {
			matched := false
			for _, app := range client.Filters.Apps {
				if appName == app {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		}
	}

	return true
}

// getRateLimiter gets or creates a rate limiter for a client
func (m *Manager) getRateLimiter(clientID string) *rate.Limiter {
	m.mu.Lock()
	defer m.mu.Unlock()

	if limiter, exists := m.rateLimiters[clientID]; exists {
		return limiter
	}

	// 100 events per second with burst of 200
	limiter := rate.NewLimiter(100, 200)
	m.rateLimiters[clientID] = limiter
	return limiter
}

// RegisterClient registers a new SSE client
func (m *Manager) RegisterClient(instanceName string, filters EventFilters) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		ID:           generateClientID(),
		InstanceName: instanceName,
		Channel:      make(chan *Event, 100),
		Filters:      filters,
		Context:      ctx,
		Cancel:       cancel,
	}

	m.register <- client
	return client
}

// UnregisterClient removes a client
func (m *Manager) UnregisterClient(client *Client) {
	client.Cancel()
	m.unregister <- client

	// Clean up rate limiter
	m.mu.Lock()
	delete(m.rateLimiters, client.ID)
	m.mu.Unlock()
}

// Broadcast sends an event to all matching clients
func (m *Manager) Broadcast(event *Event) {
	event.ID = generateEventID()
	event.Timestamp = time.Now()

	select {
	case m.broadcast <- event:
	default:
		log.Printf("SSE: Broadcast channel full, dropping event %s", event.ID)
	}
}

// GetConnectionCount returns the number of active connections
func (m *Manager) GetConnectionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, clients := range m.clients {
		count += len(clients)
	}
	return count
}

// GetInstanceConnectionCount returns the number of connections for a specific instance
func (m *Manager) GetInstanceConnectionCount(instanceName string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if clients, ok := m.clients[instanceName]; ok {
		return len(clients)
	}
	return 0
}

// Helper functions
func generateClientID() string {
	return fmt.Sprintf("client-%s", uuid.New().String()[:8])
}

func generateEventID() string {
	return fmt.Sprintf("event-%s", uuid.New().String()[:8])
}

// JSON marshals the event to JSON
func (e *Event) JSON() ([]byte, error) {
	return json.Marshal(e)
}