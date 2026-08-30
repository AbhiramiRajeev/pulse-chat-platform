package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/auth"
	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/config"
	chatgrpc "github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/grpc"
	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/postgres"
	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/redis"
	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/repository"
	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/service"
	authpb "github.com/AbhiramiRajeev/pulse-chat-platform/proto/authpb"
	"google.golang.org/grpc"
)

func main() {
	fmt.Println("Starting chat service")

	cfg := config.Load()
	ctx := context.Background()

	// PostgreSQL
	db, err := postgres.NewPool(ctx, cfg)
	if err != nil {
		log.Fatal("Failed to connect to PostgreSQL:", err)
	}
	defer db.Close()

	fmt.Println("Connected to PostgreSQL successfully")

	// Redis
	redisClient, err := redis.NewClient(ctx, cfg)
	if err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}
	defer redisClient.Close()

	fmt.Println("Connected to Redis successfully")

	// Repositories
	userRepo := repository.NewUserRepository(db)

	jwtManager := auth.NewJWTManager(
		cfg.JWTSecret,
		cfg.JWTExpiry,
	)

	// Services
	authService := service.NewAuthService(userRepo, jwtManager)

	authGRPCServer := chatgrpc.NewAuthGRPCServer(authService)

	//Create actual GRPC server
	grpcServer := grpc.NewServer()

	//Register our implementation
	authpb.RegisterAuthServiceServer(
		grpcServer,
		authGRPCServer,
	)

	//Listen and serve
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Println("Chat Service starting on gRPC port 50051")

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

	// result, err := authService.Login(
	// 	ctx,
	// 	"abhirami@example.com",
	// 	"password123",
	// )

	// if err != nil {
	// 	log.Println("Login result:", err)
	// 	return
	// }

	// fmt.Printf(
	// 	"Login successful: ID=%s, Username=%s, Email=%s\n",
	// 	result.User.ID,
	// 	result.User.Username,
	// 	result.User.Email,
	// )

	// fmt.Println("JWT Token:", result.Token)

}
