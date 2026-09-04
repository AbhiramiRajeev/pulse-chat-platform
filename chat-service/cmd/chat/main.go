package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/auth"
	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/config"
	chatgrpc "github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/grpc"
	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/postgres"
	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/redis"
	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/repository"
	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/service"

	authpb "github.com/AbhiramiRajeev/pulse-chat-platform/proto/authpb"
	messagepb "github.com/AbhiramiRajeev/pulse-chat-platform/proto/messagepb"
	roompb "github.com/AbhiramiRajeev/pulse-chat-platform/proto/roompb"

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
	roomRepo := repository.NewRoomRepository(db)
	messageRepo := repository.NewMessageRepository(db)

	// JWT
	jwtManager := auth.NewJWTManager(
		cfg.JWTSecret,
		cfg.JWTExpiry,
	)

	// Services
	authService := service.NewAuthService(userRepo, jwtManager)
	roomService := service.NewRoomService(roomRepo)

	messageService := service.NewMessageService(
		messageRepo,
		redisClient,
	)

	// Our gRPC server implementations
	authGRPCServer := chatgrpc.NewAuthGRPCServer(authService)
	roomGRPCServer := chatgrpc.NewRoomServer(roomService)
	messageGRPCServer := chatgrpc.NewMessageGRPCServer(messageService)

	// Create actual gRPC server
	grpcServer := grpc.NewServer()

	// Register our implementations
	authpb.RegisterAuthServiceServer(
		grpcServer,
		authGRPCServer,
	)

	roompb.RegisterRoomServiceServer(
		grpcServer,
		roomGRPCServer,
	)

	messagepb.RegisterMessageServiceServer(
		grpcServer,
		messageGRPCServer,
	)

	// Listen
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Println("Chat Service starting on gRPC port 50051")

	// Start gRPC server in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := grpcServer.Serve(listener); err != nil {
			log.Printf("gRPC server error: %v", err)
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

	log.Println("Shutting down gRPC server gracefully...")
	grpcServer.GracefulStop()

	// Wait for server goroutine to finish
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("gRPC server stopped")
	case <-shutdownCtx.Done():
		log.Println("Shutdown timeout exceeded, forcing stop")
		grpcServer.Stop()
	}

	log.Println("Chat service shutdown complete")
}
