package grpc

import (
	"fmt"

	"github.com/AbhiramiRajeev/pulse-chat-platform/api-gateway/internal/config"
	authpb "github.com/AbhiramiRajeev/pulse-chat-platform/proto/authpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Clients struct {
	AuthClient authpb.AuthServiceClient
	conn       *grpc.ClientConn
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

	return &Clients{
		AuthClient: authClient,
		conn:       conn,
	}, nil
}

func (c *Clients) Close() error {
	return c.conn.Close()
}