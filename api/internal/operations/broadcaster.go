package operations

import (
	"sync"
)

// Broadcaster manages SSE clients subscribed to operation output
type Broadcaster struct {
	clients map[string]map[chan []byte]bool // opID -> set of client channels
	mu      sync.RWMutex
}

// NewBroadcaster creates a new broadcaster
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		clients: make(map[string]map[chan []byte]bool),
	}
}

// Subscribe creates a new channel for receiving operation output
func (b *Broadcaster) Subscribe(opID string) chan []byte {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan []byte, 100) // Buffered to prevent slow clients from blocking
	if b.clients[opID] == nil {
		b.clients[opID] = make(map[chan []byte]bool)
	}
	b.clients[opID][ch] = true
	return ch
}

// Unsubscribe removes a client channel and closes it
func (b *Broadcaster) Unsubscribe(opID string, ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if clients, ok := b.clients[opID]; ok {
		delete(clients, ch)
		close(ch)
		if len(clients) == 0 {
			delete(b.clients, opID)
		}
	}
}

// Publish sends data to all subscribed clients for an operation
func (b *Broadcaster) Publish(opID string, data []byte) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if clients, ok := b.clients[opID]; ok {
		for ch := range clients {
			select {
			case ch <- data:
				// Sent successfully
			default:
				// Channel buffer full, skip this message for this client
			}
		}
	}
}

// Close closes all client channels for an operation
func (b *Broadcaster) Close(opID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if clients, ok := b.clients[opID]; ok {
		for ch := range clients {
			close(ch)
		}
		delete(b.clients, opID)
	}
}
