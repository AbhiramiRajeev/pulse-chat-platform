package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	gatewaygrpc "github.com/AbhiramiRajeev/pulse-chat-platform/api-gateway/internal/grpc"
	"github.com/AbhiramiRajeev/pulse-chat-platform/api-gateway/internal/middleware"
	messagepb "github.com/AbhiramiRajeev/pulse-chat-platform/proto/messagepb"
	"github.com/google/uuid"
)

type MessageHandler struct {
	clients *gatewaygrpc.Clients
}

func NewMessageHandler(
	clients *gatewaygrpc.Clients,
) *MessageHandler {
	return &MessageHandler{
		clients: clients,
	}
}

type CreateMessageRequest struct {
	Content         string `json:"content"`
	ClientMessageID string `json:"client_message_id"`
}

type MessageResponse struct {
	ID              string `json:"id"`
	RoomID          string `json:"room_id"`
	UserID          string `json:"user_id"`
	Content         string `json:"content"`
	ClientMessageID string `json:"client_message_id"`
	CreatedAt       string `json:"created_at"`
}

func (h *MessageHandler) CreateMessage(
	w http.ResponseWriter,
	r *http.Request,
) {
	roomID := r.PathValue("roomID")

	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(
			w,
			"unauthorized",
			http.StatusUnauthorized,
		)
		return
	}

	var req CreateMessageRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		r.Context(),
		5*time.Second,
	)
	defer cancel()

	response, err := h.clients.MessageClient.CreateMessage(
		ctx,
		&messagepb.CreateMessageRequest{
			RoomId:          roomID,
			UserId:          userID.String(),
			Content:         req.Content,
			ClientMessageId: req.ClientMessageID,
		},
	)

	if err != nil {
		handleGRPCError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(
		MessageResponse{
			ID:              response.GetId(),
			RoomID:          response.GetRoomId(),
			UserID:          response.GetUserId(),
			Content:         response.GetContent(),
			ClientMessageID: response.GetClientMessageId(),
			CreatedAt:       response.GetCreatedAt(),
		},
	)
}

func (h *MessageHandler) GetMessages(
	w http.ResponseWriter,
	r *http.Request,
) {
	roomID := r.PathValue("roomID")

	ctx, cancel := context.WithTimeout(
		r.Context(),
		5*time.Second,
	)
	defer cancel()

	response, err := h.clients.MessageClient.GetMessages(
		ctx,
		&messagepb.GetMessagesRequest{
			RoomId: roomID,
		},
	)

	if err != nil {
		handleGRPCError(w, err)
		return
	}

	var messages []MessageResponse

	for _, message := range response.GetMessages() {
		messages = append(
			messages,
			MessageResponse{
				ID:              message.GetId(),
				RoomID:          message.GetRoomId(),
				UserID:          message.GetUserId(),
				Content:         message.GetContent(),
				ClientMessageID: message.GetClientMessageId(),
				CreatedAt:       message.GetCreatedAt(),
			},
		)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}