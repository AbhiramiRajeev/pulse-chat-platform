package handler

import (
	"encoding/json"
	"net/http"
	"github.com/google/uuid"

	gatewaygrpc "github.com/AbhiramiRajeev/pulse-chat-platform/api-gateway/internal/grpc"
	"github.com/AbhiramiRajeev/pulse-chat-platform/api-gateway/internal/middleware"
	roompb "github.com/AbhiramiRajeev/pulse-chat-platform/proto/roompb"
)

type RoomHandler struct {
	clients *gatewaygrpc.Clients
}

func NewRoomHandler(clients *gatewaygrpc.Clients) *RoomHandler {
	return &RoomHandler{
		clients: clients,
	}
}

type CreateRoomRequest struct {
	Name string `json:"name"`
}

func (h *RoomHandler) CreateRoom(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req CreateRoomRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "user ID not found", http.StatusUnauthorized)
		return
	}

	response, err := h.clients.RoomClient.CreateRoom(
		r.Context(),
		&roompb.CreateRoomRequest{
			Name:   req.Name,
			UserId: userID.String(),
		},
	)
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(map[string]string{
		"id":         response.GetId(),
		"name":       response.GetName(),
		"created_by": response.GetCreatedBy(),
		"created_at": response.GetCreatedAt(),
	})
}

func (h *RoomHandler) GetRooms(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "user ID not found", http.StatusUnauthorized)
		return
	}

	response, err := h.clients.RoomClient.GetRooms(
		r.Context(),
		&roompb.GetRoomsRequest{
			UserId: userID.String(),
		},
	)
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(response.GetRooms())
}
