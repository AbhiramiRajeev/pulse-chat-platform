package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/models"
	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/repository"
	"github.com/google/uuid"

	goredis "github.com/redis/go-redis/v9"
)

var ErrInvalidMessageInput = errors.New("invalid message input")

const MessageChannelPrefix = "room:"

type MessageEvent struct {
	ID              string `json:"id"`
	RoomID          string `json:"room_id"`
	UserID          string `json:"user_id"`
	Content         string `json:"content"`
	ClientMessageID string `json:"client_message_id"`
	CreatedAt       string `json:"created_at"`
}

type MessageService struct {
	messageRepo *repository.MessageRepository
	redisClient *goredis.Client
}

func NewMessageService(
	messageRepo *repository.MessageRepository,
	redisClient *goredis.Client,
) *MessageService {
	return &MessageService{
		messageRepo: messageRepo,
		redisClient: redisClient,
	}
}

func (s *MessageService) CreateMessage(
	ctx context.Context,
	roomID string,
	userID string,
	content string,
	clientMessageID string,
) (*models.Message, error) {

	content = strings.TrimSpace(content)

	if roomID == "" || userID == "" || content == "" || clientMessageID == "" {
		return nil, ErrInvalidMessageInput
	}

	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return nil, ErrInvalidMessageInput
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidMessageInput
	}

	clientMessageUUID, err := uuid.Parse(clientMessageID)
	if err != nil {
		return nil, ErrInvalidMessageInput
	}

	message, err := s.messageRepo.CreateMessage(
		ctx,
		roomUUID,
		userUUID,
		content,
		clientMessageUUID,
	)
	if err != nil {
		return nil, fmt.Errorf("create message: %w", err)
	}

	event := MessageEvent{
		ID:              message.ID.String(),
		RoomID:          message.RoomID.String(),
		UserID:          message.UserID.String(),
		Content:         message.Content,
		ClientMessageID: message.ClientMessageID.String(),
		CreatedAt:       message.CreatedAt.Format(time.RFC3339),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshal message event: %w", err)
	}

	channel := MessageChannelPrefix + message.RoomID.String()

	err = s.redisClient.Publish(
		ctx,
		channel,
		payload,
	).Err()
	if err != nil {
		return nil, fmt.Errorf("publish message event: %w", err)
	}

	return message, nil
}

func (s *MessageService) GetMessages(
	ctx context.Context,
	roomID string,
) ([]*models.Message, error) {

	if roomID == "" {
		return nil, ErrInvalidMessageInput
	}

	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return nil, ErrInvalidMessageInput
	}

	messages, err := s.messageRepo.GetMessagesByRoomID(
		ctx,
		roomUUID,
	)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}

	return messages, nil
}
