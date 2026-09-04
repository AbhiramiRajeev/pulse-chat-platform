package websocket

import (
	"net/http"

	gatewaygrpc "github.com/AbhiramiRajeev/pulse-chat-platform/api-gateway/internal/grpc"
	"github.com/gorilla/websocket"
)

type Handler struct {
	clients  *gatewaygrpc.Clients
	hub      *Hub
	upgrader websocket.Upgrader
}

func NewHandler(clients *gatewaygrpc.Clients, hub *Hub) *Handler {
	return &Handler{
		clients: clients,
		hub:     hub,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (h *Handler) Connect(
	w http.ResponseWriter,
	r *http.Request,
) {
	roomID := r.PathValue("roomID")
	if roomID == "" {
		http.Error(
			w,
			"room ID is required",
			http.StatusBadRequest,
		)
		return
	}

	ws, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	// JoinRoom returns the wrapped conn (with its write mutex).
	c := h.hub.JoinRoom(roomID, ws)
	defer func() {
		h.hub.LeaveRoom(roomID, c)
		c.close()
	}()

	// Read loop — keeps the connection alive and detects disconnects.
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}
}
