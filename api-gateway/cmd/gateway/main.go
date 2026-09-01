package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/AbhiramiRajeev/pulse-chat-platform/api-gateway/internal/config"
	gatewaygrpc "github.com/AbhiramiRajeev/pulse-chat-platform/api-gateway/internal/grpc"
	"github.com/AbhiramiRajeev/pulse-chat-platform/api-gateway/internal/handler"
	"github.com/AbhiramiRajeev/pulse-chat-platform/api-gateway/internal/middleware"
)

func main() {
	fmt.Println("Starting gateway service")

	cfg := config.Load()

	// Create the gRPC connection and clients.
	clients, err := gatewaygrpc.NewClients(cfg)
	if err != nil {
		log.Fatalf("failed to create gRPC clients: %v", err)
	}
	defer clients.Close()

	fmt.Println("Connected to Chat Service successfully")

	// Create HTTP handlers.
	authHandler := handler.NewAuthHandler(clients)

	// Create HTTP routes.
	mux := http.NewServeMux()

	// Protected route.
	mux.Handle(
		"GET /health",
		middleware.Auth(cfg.JWTSecret)(
			http.HandlerFunc(handler.Health),
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

	addr := ":" + cfg.HTTPPort

	fmt.Println("Gateway listening on HTTP port:", cfg.HTTPPort)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}
