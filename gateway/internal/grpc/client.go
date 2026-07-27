package grpc

import (
	"fmt"

	chatpb "github.com/AbhiramiRajeev/pulse-chat-platform/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewChatClient() (chatpb.ChatServiceClient, error) {
	conn, err := grpc.NewClient(
		"localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to chat service: %w", err)
	}

	client := chatpb.NewChatServiceClient(conn)

	return client, nil
}
