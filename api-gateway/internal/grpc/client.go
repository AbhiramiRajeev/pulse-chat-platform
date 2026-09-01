package grpc

import (
	"fmt"

	"github.com/AbhiramiRajeev/pulse-chat-platform/api-gateway/internal/config"
	authpb "github.com/AbhiramiRajeev/pulse-chat-platform/proto/authpb"
	messagepb "github.com/AbhiramiRajeev/pulse-chat-platform/proto/messagepb"
	roompb "github.com/AbhiramiRajeev/pulse-chat-platform/proto/roompb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Clients struct {
	AuthClient    authpb.AuthServiceClient
	RoomClient    roompb.RoomServiceClient
	MessageClient messagepb.MessageServiceClient

	conn *grpc.ClientConn
}

func NewClients(cfg config.Config) (*Clients, error) {
	conn, err := grpc.NewClient(
		cfg.ChatServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to chat service: %w", err)
	}

	authClient := authpb.NewAuthServiceClient(conn)
	roomClient := roompb.NewRoomServiceClient(conn)
	messageClient := messagepb.NewMessageServiceClient(conn)

	return &Clients{
		AuthClient:    authClient,
		RoomClient:    roomClient,
		MessageClient: messageClient,
		conn:          conn,
	}, nil
}

func (c *Clients) Close() error {
	return c.conn.Close()
}