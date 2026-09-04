package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	wsHandler := chatws.NewHandler(clients, hub)

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
		"POST /rooms/{roomID}/members",
		authMiddleware(
			middleware.RoomMembership(clients)(
				http.HandlerFunc(roomHandler.AddMember),
			),
		),
	)

	mux.Handle(
		"GET /rooms/{roomID}/members",
		authMiddleware(
			middleware.RoomMembership(clients)(
				http.HandlerFunc(roomHandler.ListMembers),
			),
		),
	)

	mux.Handle(
		"POST /rooms/{roomID}/messages",
		authMiddleware(
			middleware.RoomMembership(clients)(
				http.HandlerFunc(messageHandler.CreateMessage),
			),
		),
	)

	mux.Handle(
		"GET /rooms/{roomID}/messages",
		authMiddleware(
			middleware.RoomMembership(clients)(
				http.HandlerFunc(messageHandler.GetMessages),
			),
		),
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
	mux.Handle(
		"GET /ws/rooms/{roomID}",
		middleware.WebSocketAuth(cfg.JWTSecret)(
			middleware.RoomMembership(clients)(
				http.HandlerFunc(wsHandler.Connect),
			),
		),
	)

	// Create HTTP server
	addr := ":" + cfg.HTTPPort
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	fmt.Println("Gateway listening on HTTP port:", cfg.HTTPPort)

	// Start server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for signal
	sig := <-sigChan
	log.Printf("Received signal: %v", sig)

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Println("Shutting down HTTP server gracefully...")
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("Gateway shutdown complete")
}
