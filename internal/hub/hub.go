package hub

import (
	"encoding/json"
	"log"
	"sync"
)

// Hub maintains the set of active WebSocket clients and broadcasts events.
type Hub struct {
	// clients maps userID -> set of connected clients.
	clients map[uint]map[*Client]bool

	// broadcast sends an event to all connected clients.
	broadcast chan Event

	// register adds a client to the hub.
	register chan *Client

	// unregister removes a client from the hub.
	unregister chan *Client

	// mu protects clients map for concurrent reads (e.g. BroadcastToUser).
	mu sync.RWMutex
}

// NewHub creates a new Hub instance.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uint]map[*Client]bool),
		broadcast:  make(chan Event, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's event loop. Must be called in its own goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.userID] == nil {
				h.clients[client.userID] = make(map[*Client]bool)
			}
			h.clients[client.userID][client] = true
			h.mu.Unlock()
			log.Printf("websocket client registered for user %d (total: %d)", client.userID, len(h.clients[client.userID]))

		case client := <-h.unregister:
			h.mu.Lock()
			if conns, ok := h.clients[client.userID]; ok {
				if _, exists := conns[client]; exists {
					delete(conns, client)
					close(client.send)
					if len(conns) == 0 {
						delete(h.clients, client.userID)
					}
				}
			}
			h.mu.Unlock()

		case event := <-h.broadcast:
			data, err := json.Marshal(event)
			if err != nil {
				log.Printf("failed to marshal broadcast event: %v", err)
				continue
			}
			h.mu.RLock()
			for _, conns := range h.clients {
				for client := range conns {
					select {
					case client.send <- data:
					default:
						// Buffer full — will be cleaned up on next write failure.
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Register adds a client to the hub via the register channel.
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub via the unregister channel.
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// BroadcastToUser sends an event to all WebSocket clients for a specific user.
func (h *Hub) BroadcastToUser(userID uint, event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("failed to marshal event for user %d: %v", userID, err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	conns, ok := h.clients[userID]
	if !ok {
		return
	}

	for client := range conns {
		select {
		case client.send <- data:
		default:
			// Buffer full — drop silently.
		}
	}
}

// Broadcast sends an event to all connected clients.
func (h *Hub) Broadcast(event Event) {
	h.broadcast <- event
}
