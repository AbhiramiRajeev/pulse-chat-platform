package websocket

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	mu sync.RWMutex

	rooms map[string]map[*websocket.Conn]bool
}
/*
	rooms
 │
 ├── "room-1"
 │      │
 │      ├── Connection A → true
 │      └── Connection B → true
 │
 └── "room-2"
        │
        └── Connection C → true
The outer map is:

Room ID → clients in that room
*/

func NewHub() *Hub {
	return &Hub{
		rooms: make(map[string]map[*websocket.Conn]bool),
	}
}

//JoinRoom adds this WebSocket connection to this room.
// A connection represents an active user's connection (for example, their browser or device). We store the connection so we can later
// send messages directly to it using conn.WriteMessage().
// If the room does not exist in the Hub, we first create its
// connections map, then add the connection to that room.
func (h *Hub) JoinRoom(
	roomID string,
	conn *websocket.Conn,
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.rooms[roomID] == nil {
		h.rooms[roomID] = make(
			map[*websocket.Conn]bool,
		)
	}

	h.rooms[roomID][conn] = true
}


func (h *Hub) LeaveRoom(
	roomID string,
	conn *websocket.Conn,
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients, exists := h.rooms[roomID]
	if !exists {
		return
	}

	delete(clients, conn)

	if len(clients) == 0 {
		delete(h.rooms, roomID)
	}
}

func (h *Hub) Broadcast(
	roomID string,
	message []byte,
) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, exists := h.rooms[roomID]
	if !exists {
		return
	}

	for conn := range clients {
		err := conn.WriteMessage(
			websocket.TextMessage,
			message,
		)

		if err != nil {
			continue
		}
	}
}