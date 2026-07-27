package router

import (
	"net/http"

	"github.com/AbhiramiRajeev/pulse-chat-platform/gateway/internal/api"
	"github.com/AbhiramiRajeev/pulse-chat-platform/gateway/internal/websocket"
)

func NewRouter() *http.ServeMux {
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /health", api.HealthHandler)

	// Authentication
	mux.HandleFunc("POST /api/v1/auth/register", api.RegisterHandler)
	mux.HandleFunc("POST /api/v1/auth/login", api.LoginHandler)
	mux.HandleFunc("POST /api/v1/auth/refresh", api.RefreshTokenHandler)

	// Rooms
	mux.HandleFunc("POST /api/v1/rooms", api.CreateRoomHandler)
	mux.HandleFunc("GET /api/v1/rooms", api.ListRoomsHandler)
	mux.HandleFunc("GET /api/v1/rooms/{roomId}", api.GetRoomHandler)

	// Messages
	mux.HandleFunc("GET /api/v1/rooms/{roomId}/messages", api.GetMessageHistoryHandler)

	// WebSocket
	mux.HandleFunc("GET /ws", websocket.Handler)

	return mux
}