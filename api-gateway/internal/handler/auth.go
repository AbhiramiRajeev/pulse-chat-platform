package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	gatewaygrpc "github.com/AbhiramiRajeev/pulse-chat-platform/api-gateway/internal/grpc"
	authpb "github.com/AbhiramiRajeev/pulse-chat-platform/proto/authpb"
)

type AuthHandler struct {
	authClient authpb.AuthServiceClient
}

func NewAuthHandler(clients *gatewaygrpc.Clients,) *AuthHandler {
	return &AuthHandler{
		authClient: clients.AuthClient,
	}
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Token    string `json:"token"`
}

func (h *AuthHandler) Register(w http.ResponseWriter,r *http.Request,) {
	var req registerRequest


	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resp, err := h.authClient.Register(
		ctx,
		&authpb.RegisterRequest{
			Username: req.Username,
			Email:    req.Email,
			Password: req.Password,
		},
	)
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	response := registerResponse{
		ID:       resp.GetId(),
		Username: resp.GetUsername(),
		Email:    resp.GetEmail(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(response)
}


func (h *AuthHandler) Login(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req loginRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resp, err := h.authClient.Login(
		ctx,
		&authpb.LoginRequest{
			Email:    req.Email,
			Password: req.Password,
		},
	)
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	response := loginResponse{
		ID:       resp.GetId(),
		Username: resp.GetUsername(),
		Email:    resp.GetEmail(),
		Token:    resp.GetToken(),
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(response)
}