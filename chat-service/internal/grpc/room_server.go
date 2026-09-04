package grpc

import (
	"context"
	"strings"
	"time"

	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/service"
	roompb "github.com/AbhiramiRajeev/pulse-chat-platform/proto/roompb"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	if strings.TrimSpace(req.GetName()) == "" {
		return nil, status.Error(codes.InvalidArgument, "room name is required")
	}

	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	room, err := s.roomService.CreateRoom(
		ctx,
		req.GetName(),
		userID,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "internal server error")
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
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
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

func (s *RoomServer) IsUserInRoom(
	ctx context.Context,
	req *roompb.IsUserInRoomRequest,
) (*roompb.IsUserInRoomResponse, error) {
	isMember, err := s.roomService.IsUserInRoom(
		ctx,
		req.GetRoomId(),
		req.GetUserId(),
	)
	if err != nil {
		return nil, err
	}

	return &roompb.IsUserInRoomResponse{
		IsMember: isMember,
	}, nil
}

func (s *RoomServer) AddMember(
	ctx context.Context,
	req *roompb.AddMemberRequest,
) (*roompb.AddMemberResponse, error) {
	err := s.roomService.AddMember(
		ctx,
		req.GetRoomId(),
		req.GetUserId(),
		req.GetRequestingUserId(),
	)
	if err != nil {
		msg := err.Error()
		switch {
		case strings.Contains(msg, "only room creator can add members"):
			return nil, status.Error(codes.PermissionDenied, "only room creator can add members")
		case strings.Contains(msg, "user is already a member"):
			return nil, status.Error(codes.AlreadyExists, "user is already a member")
		case strings.Contains(msg, "invalid"):
			return nil, status.Error(codes.InvalidArgument, msg)
		default:
			return nil, status.Error(codes.Internal, "internal server error")
		}
	}

	return &roompb.AddMemberResponse{
		Success: true,
		Message: "member added successfully",
	}, nil
}

func (s *RoomServer) ListMembers(
	ctx context.Context,
	req *roompb.ListMembersRequest,
) (*roompb.ListMembersResponse, error) {
	members, err := s.roomService.ListMembers(
		ctx,
		req.GetRoomId(),
	)
	if err != nil {
		return nil, err
	}

	response := &roompb.ListMembersResponse{
		Members: make([]*roompb.RoomMember, 0, len(members)),
	}

	for _, member := range members {
		response.Members = append(
			response.Members,
			&roompb.RoomMember{
				UserId:   member.UserID.String(),
				Username: member.Username,
				Email:    member.Email,
				JoinedAt: member.JoinedAt.Format(time.RFC3339),
			},
		)
	}

	return response, nil
}
