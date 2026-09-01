package websocket

import (
	"net/http"

	"github.com/gorilla/websocket"
)

type Handler struct {
	hub      *Hub
	upgrader websocket.Upgrader
}

func NewHandler(hub *Hub) *Handler {
	return &Handler{
		hub: hub,
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

	conn, err := h.upgrader.Upgrade(
		w,
		r,
		nil,
	)
	if err != nil {
		return
	}

	defer conn.Close()

	h.hub.JoinRoom(
		roomID,
		conn,
	)

	defer h.hub.LeaveRoom(
		roomID,
		conn,
	)

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}