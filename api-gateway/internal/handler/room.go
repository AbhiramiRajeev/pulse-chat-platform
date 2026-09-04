package handler

import (
	"context"
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

type roomService interface {
	IsUserInRoom(context.Context, string, string) (bool, error)
}

type RoomGRPCServer struct {
	roompb.UnimplementedRoomServiceServer
	roomService roomService
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

func (s *RoomGRPCServer) IsUserInRoom(
	ctx context.Context,
	req *roompb.IsUserInRoomRequest,
) (*roompb.IsUserInRoomResponse, error) {

	isMember, err := s.roomService.IsUserInRoom(
		ctx,
		req.GetRoomId(),
		req.GetUserId(),
	)
	if err != nil {
		return nil, err
	}

	return &roompb.IsUserInRoomResponse{
		IsMember: isMember,
	}, nil
}

type AddMemberRequest struct {
	UserID string `json:"user_id"`
}

func (h *RoomHandler) AddMember(
	w http.ResponseWriter,
	r *http.Request,
) {
	roomID := r.PathValue("roomID")

	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	response, err := h.clients.RoomClient.AddMember(
		r.Context(),
		&roompb.AddMemberRequest{
			RoomId:           roomID,
			UserId:           req.UserID,
			RequestingUserId: userID.String(),
		},
	)
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	statusCode := http.StatusOK
	if !response.GetSuccess() {
		statusCode = http.StatusBadRequest
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": response.GetSuccess(),
		"message": response.GetMessage(),
	})
}

func (h *RoomHandler) ListMembers(
	w http.ResponseWriter,
	r *http.Request,
) {
	roomID := r.PathValue("roomID")

	response, err := h.clients.RoomClient.ListMembers(
		r.Context(),
		&roompb.ListMembersRequest{
			RoomId: roomID,
		},
	)
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	members := make([]map[string]string, 0, len(response.GetMembers()))
	for _, member := range response.GetMembers() {
		members = append(members, map[string]string{
			"user_id":   member.GetUserId(),
			"username":  member.GetUsername(),
			"email":     member.GetEmail(),
			"joined_at": member.GetJoinedAt(),
		})
	}

	json.NewEncoder(w).Encode(members)
}
