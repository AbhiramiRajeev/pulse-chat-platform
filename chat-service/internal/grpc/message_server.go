package grpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/service"
	messagepb "github.com/AbhiramiRajeev/pulse-chat-platform/proto/messagepb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MessageGRPCServer struct {
	messagepb.UnimplementedMessageServiceServer
	messageService *service.MessageService
}

func NewMessageGRPCServer(
	messageService *service.MessageService,
) *MessageGRPCServer {
	return &MessageGRPCServer{
		messageService: messageService,
	}
}

func (s *MessageGRPCServer) CreateMessage(
	ctx context.Context,
	req *messagepb.CreateMessageRequest,
) (*messagepb.CreateMessageResponse, error) {

	message, err := s.messageService.CreateMessage(
		ctx,
		req.GetRoomId(),
		req.GetUserId(),
		req.GetContent(),
		req.GetClientMessageId(),
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidMessageInput):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case errors.Is(err, service.ErrNotRoomMember):
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
		return nil, fmt.Errorf("create message: %w", err)
	}

	return &messagepb.CreateMessageResponse{
		Id:              message.ID.String(),
		RoomId:          message.RoomID.String(),
		UserId:          message.UserID.String(),
		Content:         message.Content,
		ClientMessageId: message.ClientMessageID.String(),
		CreatedAt:       message.CreatedAt.Format(time.RFC3339Nano),
	}, nil
}

func (s *MessageGRPCServer) GetMessages(
	ctx context.Context,
	req *messagepb.GetMessagesRequest,
) (*messagepb.GetMessagesResponse, error) {

	messages, err := s.messageService.GetMessages(
		ctx,
		req.GetRoomId(),
	)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}

	response := &messagepb.GetMessagesResponse{}

	for _, message := range messages {
		response.Messages = append(
			response.Messages,
			&messagepb.Message{
				Id:              message.ID.String(),
				RoomId:          message.RoomID.String(),
				UserId:          message.UserID.String(),
				Content:         message.Content,
				ClientMessageId: message.ClientMessageID.String(),
				CreatedAt:       message.CreatedAt.Format(time.RFC3339Nano),
			},
		)
	}

	return response, nil
}
