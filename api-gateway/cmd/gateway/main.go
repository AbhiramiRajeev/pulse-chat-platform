package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/AbhiramiRajeev/pulse-chat-platform/api-gateway/internal/config"
	gatewaygrpc "github.com/AbhiramiRajeev/pulse-chat-platform/api-gateway/internal/grpc"
	"github.com/AbhiramiRajeev/pulse-chat-platform/api-gateway/internal/handler"
	"github.com/AbhiramiRajeev/pulse-chat-platform/api-gateway/internal/middleware"
	chatws "github.com/AbhiramiRajeev/pulse-chat-platform/api-gateway/internal/websocket"

	gatewayredis "github.com/AbhiramiRajeev/pulse-chat-platform/api-gateway/internal/redis"
)

func main() {
	fmt.Println("Starting gateway service")

	cfg := config.Load()

	ctx := context.Background()

	redisClient, err := gatewayredis.NewClient(
		ctx,
		cfg,
	)
	if err != nil {
		log.Fatalf(
			"failed to connect to Redis: %v",
			err,
		)
	}
	defer redisClient.Close()

	fmt.Println("Connected to Redis successfully")

	// Create the gRPC connection and clients.
	clients, err := gatewaygrpc.NewClients(cfg)
	if err != nil {
		log.Fatalf("failed to create gRPC clients: %v", err)
	}
	defer clients.Close()

	fmt.Println("Connected to Chat Service successfully")

	// Create HTTP handlers.
	authHandler := handler.NewAuthHandler(clients)

	roomHandler := handler.NewRoomHandler(clients)

	messageHandler := handler.NewMessageHandler(clients)

	// Create WebSocket Hub and handler.
	hub := chatws.NewHub()
	wsHandler := chatws.NewHandler(hub)

	// Subscribe to Redis Pub/Sub for messages and broadcast them to WebSocket clients.
	go gatewayredis.SubscribeToMessages(
		ctx,
		redisClient,
		hub,
	)

	// Create HTTP routes.
	mux := http.NewServeMux()

	// Protected route.
	authMiddleware := middleware.Auth(cfg.JWTSecret)

	mux.Handle(
		"POST /rooms",
		authMiddleware(http.HandlerFunc(roomHandler.CreateRoom)),
	)

	mux.Handle(
		"GET /rooms",
		authMiddleware(http.HandlerFunc(roomHandler.GetRooms)),
	)

	mux.Handle(
		"POST /rooms/{roomID}/messages",
		authMiddleware(http.HandlerFunc(messageHandler.CreateMessage)),
	)

	mux.Handle(
		"GET /rooms/{roomID}/messages",
		authMiddleware(http.HandlerFunc(messageHandler.GetMessages)),
	)

	// Public routes.
	mux.HandleFunc(
		"POST /register",
		authHandler.Register,
	)

	mux.HandleFunc(
		"POST /login",
		authHandler.Login,
	)

	// WebSocket route.
	mux.HandleFunc(
		"GET /ws/rooms/{roomID}",
		wsHandler.Connect,
	)
	addr := ":" + cfg.HTTPPort

	fmt.Println("Gateway listening on HTTP port:", cfg.HTTPPort)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}
