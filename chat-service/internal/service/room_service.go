package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/models"
	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/repository"
)

var ErrRoomNotFound = errors.New("room not found")

type RoomService struct {
	roomRepo *repository.RoomRepository
}

func NewRoomService(
	roomRepo *repository.RoomRepository,
) *RoomService {
	return &RoomService{
		roomRepo: roomRepo,
	}
}

func (s *RoomService) CreateRoom(
	ctx context.Context,
	name string,
	userID uuid.UUID,
) (*models.Room, error) {

	name = strings.TrimSpace(name)

	if name == "" {
		return nil, ErrInvalidInput
	}

	room, err := s.roomRepo.CreateRoom(
		ctx,
		name,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("create room: %w", err)
	}

	return room, nil
}

func (s *RoomService) GetRoomsByUserID(
	ctx context.Context,
	userID uuid.UUID,
) ([]models.Room, error) {

	rooms, err := s.roomRepo.GetRoomsByUserID(
		ctx,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get rooms: %w", err)
	}

	return rooms, nil
}
