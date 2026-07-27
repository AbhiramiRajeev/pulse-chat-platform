package grpc

import (
	"context"

	chatpb "github.com/AbhiramiRajeev/pulse-chat-platform/proto"
)

type ChatServer struct {
	chatpb.UnimplementedChatServiceServer
}

func NewChatServer() *ChatServer {
	return &ChatServer{}
}


func (s *ChatServer) ListRooms(ctx context.Context,req *chatpb.ListRoomsRequest,) (*chatpb.ListRoomsResponse, error) {
	return &chatpb.ListRoomsResponse{
		Rooms: []*chatpb.Room{},
	}, nil
}