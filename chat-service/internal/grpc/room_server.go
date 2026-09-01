package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/service"
	roompb "github.com/AbhiramiRajeev/pulse-chat-platform/proto/roompb"
	"github.com/google/uuid"
)

type RoomServer struct {
	roompb.UnimplementedRoomServiceServer

	roomService *service.RoomService
}

func NewRoomServer(
	roomService *service.RoomService,
) *RoomServer {
	return &RoomServer{
		roomService: roomService,
	}
}

func (s *RoomServer) CreateRoom(
	ctx context.Context,
	req *roompb.CreateRoomRequest,
) (*roompb.CreateRoomResponse, error) {

	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	room, err := s.roomService.CreateRoom(
		ctx,
		req.GetName(),
		userID,
	)
	if err != nil {
		return nil, err
	}

	return &roompb.CreateRoomResponse{
		Id:        room.ID.String(),
		Name:      room.Name,
		CreatedBy: room.CreatedBy.String(),
		CreatedAt: room.CreatedAt.Format(time.RFC3339),
	}, nil
}


func (s *RoomServer) GetRooms(
	ctx context.Context,
	req *roompb.GetRoomsRequest,
) (*roompb.GetRoomsResponse, error) {

	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	rooms, err := s.roomService.GetRoomsByUserID(
		ctx,
		userID,
	)
	if err != nil {
		return nil, err
	}

	response := &roompb.GetRoomsResponse{
		Rooms: make([]*roompb.Room, 0, len(rooms)),
	}

	for _, room := range rooms {
		response.Rooms = append(
			response.Rooms,
			&roompb.Room{
				Id:        room.ID.String(),
				Name:      room.Name,
				CreatedBy: room.CreatedBy.String(),
				CreatedAt: room.CreatedAt.Format(time.RFC3339),
			},
		)
	}

	return response, nil
}