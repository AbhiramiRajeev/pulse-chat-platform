package websocket

import (
	"sync"

	"github.com/gorilla/websocket"
)

// conn wraps a WebSocket connection with a write mutex.
// Gorilla WebSocket allows only one concurrent writer per connection;
// the mutex ensures that broadcasts from multiple goroutines are serialised.
type conn struct {
	ws  *websocket.Conn
	mu  sync.Mutex
}

func (c *conn) writeMessage(messageType int, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ws.WriteMessage(messageType, data)
}

func (c *conn) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ws.Close()
}

type Hub struct {
	mu sync.RWMutex

	// rooms maps roomID → set of active connections.
	rooms map[string]map[*conn]bool
}

/*
	rooms
 │
 ├── "room-1"
 │      │
 │      ├── conn A → true
 │      └── conn B → true
 │
 └── "room-2"
        │
        └── conn C → true

The outer map is:

Room ID → clients in that room
*/

func NewHub() *Hub {
	return &Hub{
		rooms: make(map[string]map[*conn]bool),
	}
}

// JoinRoom adds a WebSocket connection to a room.
// Returns the wrapped conn so the caller can use it to call LeaveRoom and close.
func (h *Hub) JoinRoom(roomID string, ws *websocket.Conn) *conn {
	c := &conn{ws: ws}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.rooms[roomID] == nil {
		h.rooms[roomID] = make(map[*conn]bool)
	}

	h.rooms[roomID][c] = true
	return c
}

// LeaveRoom removes a connection from a room and cleans up the room map if empty.
func (h *Hub) LeaveRoom(roomID string, c *conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients, exists := h.rooms[roomID]
	if !exists {
		return
	}

	delete(clients, c)

	if len(clients) == 0 {
		delete(h.rooms, roomID)
	}
}

// Broadcast sends a message to all connections in a room.
// Failed connections (dead/disconnected) are removed after the broadcast pass.
func (h *Hub) Broadcast(roomID string, message []byte) {
	h.mu.RLock()

	clients, exists := h.rooms[roomID]
	if !exists {
		h.mu.RUnlock()
		return
	}

	// Snapshot the current connection set so we don't hold the lock while writing.
	conns := make([]*conn, 0, len(clients))
	for c := range clients {
		conns = append(conns, c)
	}

	h.mu.RUnlock()

	// Write to each connection outside the hub lock.
	// Each conn has its own write mutex, so concurrent Broadcast calls are safe.
	var failedConns []*conn
	for _, c := range conns {
		if err := c.writeMessage(websocket.TextMessage, message); err != nil {
			failedConns = append(failedConns, c)
		}
	}

	// Remove failed connections under a write lock.
	if len(failedConns) > 0 {
		h.mu.Lock()
		clients = h.rooms[roomID] // re-fetch; room may have been deleted
		if clients != nil {
			for _, c := range failedConns {
				delete(clients, c)
			}
			if len(clients) == 0 {
				delete(h.rooms, roomID)
			}
		}
		h.mu.Unlock()
	}
}
