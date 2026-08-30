package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/auth"
	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/models"
	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type AuthService struct {
	userRepo   *repository.UserRepository
	jwtManager *auth.JWTManager
}

func NewAuthService(userRepo *repository.UserRepository, jwtManager *auth.JWTManager) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		jwtManager: jwtManager,
	}
}

type LoginResult struct {
	User  *models.User
	Token string
}

func (s *AuthService) Register(ctx context.Context, username string, email string, password string) (*models.User, error) {
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(strings.ToLower(email))

	if username == "" || email == "" || password == "" {
		return nil, ErrInvalidInput
	}

	existingUser, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("check existing user: %w", err)
	}

	if existingUser != nil {
		return nil, ErrUserAlreadyExists
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.userRepo.CreateUser(
		ctx,
		username,
		email,
		string(passwordHash),
	)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (s *AuthService) Login(ctx context.Context,email string,password string,) (*LoginResult, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	if email == "" || password == "" {
		return nil, ErrInvalidInput
	}

	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	if user == nil {
		return nil, ErrInvalidCredentials
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(password),
	)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Password is correct, so generate a JWT for this user.
	token, err := s.jwtManager.GenerateToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &LoginResult{
		User:  user,
		Token: token,
	}, nil

}
