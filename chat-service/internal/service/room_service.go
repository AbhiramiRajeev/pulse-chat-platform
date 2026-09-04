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
var ErrInvalidRoomInput = errors.New("invalid room input")

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

func (s *RoomService) IsUserInRoom(
	ctx context.Context,
	roomID string,
	userID string,
) (bool, error) {
	if roomID == "" || userID == "" {
		return false, ErrInvalidRoomInput
	}

	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return false, ErrInvalidRoomInput
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return false, ErrInvalidRoomInput
	}

	isMember, err := s.roomRepo.IsUserInRoom(
		ctx,
		roomUUID,
		userUUID,
	)
	if err != nil {
		return false, fmt.Errorf(
			"check user in room: %w",
			err,
		)
	}

	return isMember, nil
}

func (s *RoomService) AddMember(
	ctx context.Context,
	roomID string,
	userID string,
	requestingUserID string,
) error {
	if roomID == "" || userID == "" || requestingUserID == "" {
		return ErrInvalidRoomInput
	}

	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return ErrInvalidRoomInput
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return ErrInvalidRoomInput
	}

	requestingUUID, err := uuid.Parse(requestingUserID)
	if err != nil {
		return ErrInvalidRoomInput
	}

	err = s.roomRepo.AddMember(ctx, roomUUID, userUUID, requestingUUID)
	if err != nil {
		return fmt.Errorf("add member: %w", err)
	}

	return nil
}

func (s *RoomService) ListMembers(
	ctx context.Context,
	roomID string,
) ([]repository.RoomMember, error) {
	if roomID == "" {
		return nil, ErrInvalidRoomInput
	}

	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return nil, ErrInvalidRoomInput
	}

	members, err := s.roomRepo.ListMembers(ctx, roomUUID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}

	return members, nil
}
