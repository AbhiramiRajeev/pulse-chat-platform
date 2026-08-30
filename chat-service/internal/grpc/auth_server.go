package grpc

import (
	"context"
	"errors"

	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/service"
	authpb "github.com/AbhiramiRajeev/pulse-chat-platform/proto/authpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthGRPCServer struct {
	authpb.UnimplementedAuthServiceServer

	authService *service.AuthService
}

func NewAuthGRPCServer(authService *service.AuthService,) *AuthGRPCServer {
	return &AuthGRPCServer{
		authService: authService,
	}
}

func (s *AuthGRPCServer) Register(ctx context.Context,req *authpb.RegisterRequest,) (*authpb.RegisterResponse, error) {

	user, err := s.authService.Register(
		ctx,
		req.GetUsername(),
		req.GetEmail(),
		req.GetPassword(),
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserAlreadyExists):
			return nil, status.Error(
				codes.AlreadyExists,
				err.Error(),
			)

		case errors.Is(err, service.ErrInvalidInput):
			return nil, status.Error(
				codes.InvalidArgument,
				err.Error(),
			)

		default:
			return nil, status.Error(
				codes.Internal,
				"internal server error",
			)
		}
	}

	return &authpb.RegisterResponse{
		Id:       user.ID.String(),
		Username: user.Username,
		Email:    user.Email,
	}, nil
}

func (s *AuthGRPCServer) Login(
	ctx context.Context,
	req *authpb.LoginRequest,
) (*authpb.LoginResponse, error) {

	result, err := s.authService.Login(
		ctx,
		req.GetEmail(),
		req.GetPassword(),
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			return nil, status.Error(
				codes.Unauthenticated,
				err.Error(),
			)

		case errors.Is(err, service.ErrInvalidInput):
			return nil, status.Error(
				codes.InvalidArgument,
				err.Error(),
			)

		default:
			return nil, status.Error(
				codes.Internal,
				"internal server error",
			)
		}
	}

	return &authpb.LoginResponse{
		Id:       result.User.ID.String(),
		Username: result.User.Username,
		Email:    result.User.Email,
		Token:    result.Token,
	}, nil
}