package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/models"
	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/repository"
	"github.com/google/uuid"
)

type fakeMessageRepository struct {
	createdMessage *models.Message
	createErr      error
	messages       []*models.Message
	createCalls    int
}

func (r *fakeMessageRepository) CreateMessage(
	context.Context,
	uuid.UUID,
	uuid.UUID,
	string,
	uuid.UUID,
) (*models.Message, error) {
	r.createCalls++
	return r.createdMessage, r.createErr
}

func (r *fakeMessageRepository) GetMessagesByRoomID(
	context.Context,
	uuid.UUID,
) ([]*models.Message, error) {
	return r.messages, nil
}

type fakeMessagePublisher struct {
	calls   int
	channel string
	payload any
	err     error
}

type fakeRoomMembershipChecker struct {
	isMember bool
	err      error
}

func (c *fakeRoomMembershipChecker) IsUserInRoom(
	context.Context,
	uuid.UUID,
	uuid.UUID,
) (bool, error) {
	return c.isMember, c.err
}

func (p *fakeMessagePublisher) Publish(
	_ context.Context,
	channel string,
	payload any,
) error {
	p.calls++
	p.channel = channel
	p.payload = payload
	return p.err
}

func TestCreateMessagePublishesNewMessage(t *testing.T) {
	roomID := uuid.New()
	message := &models.Message{
		ID:              uuid.New(),
		RoomID:          roomID,
		UserID:          uuid.New(),
		Content:         "hello",
		ClientMessageID: uuid.New(),
		CreatedAt:       time.Date(2026, time.October, 27, 8, 0, 0, 0, time.UTC),
	}
	repo := &fakeMessageRepository{createdMessage: message}
	publisher := &fakeMessagePublisher{}
	service := NewMessageService(repo, publisher, &fakeRoomMembershipChecker{isMember: true})

	got, err := service.CreateMessage(
		context.Background(),
		message.RoomID.String(),
		message.UserID.String(),
		message.Content,
		message.ClientMessageID.String(),
	)
	if err != nil {
		t.Fatalf("CreateMessage() error = %v", err)
	}
	if got != message {
		t.Fatalf("CreateMessage() returned a different message")
	}
	if publisher.calls != 1 {
		t.Fatalf("Publish() calls = %d, want 1", publisher.calls)
	}
	if publisher.channel != MessageChannelPrefix+roomID.String() {
		t.Fatalf("Publish() channel = %q", publisher.channel)
	}
}

func TestCreateMessageDuplicateReturnsExistingWithoutPublishing(t *testing.T) {
	message := &models.Message{
		ID:              uuid.New(),
		RoomID:          uuid.New(),
		UserID:          uuid.New(),
		Content:         "hello",
		ClientMessageID: uuid.New(),
		CreatedAt:       time.Now().UTC(),
	}
	repo := &fakeMessageRepository{
		createdMessage: message,
		createErr:      repository.ErrDuplicateMessage,
	}
	publisher := &fakeMessagePublisher{}
	service := NewMessageService(repo, publisher, &fakeRoomMembershipChecker{isMember: true})

	got, err := service.CreateMessage(
		context.Background(),
		message.RoomID.String(),
		message.UserID.String(),
		message.Content,
		message.ClientMessageID.String(),
	)
	if err != nil {
		t.Fatalf("CreateMessage() error = %v", err)
	}
	if got != message {
		t.Fatalf("CreateMessage() returned a different message")
	}
	if publisher.calls != 0 {
		t.Fatalf("Publish() calls = %d, want 0", publisher.calls)
	}
}

func TestCreateMessageReturnsPublishFailure(t *testing.T) {
	message := &models.Message{
		ID:              uuid.New(),
		RoomID:          uuid.New(),
		UserID:          uuid.New(),
		Content:         "hello",
		ClientMessageID: uuid.New(),
		CreatedAt:       time.Now().UTC(),
	}
	publishErr := errors.New("redis unavailable")
	service := NewMessageService(
		&fakeMessageRepository{createdMessage: message},
		&fakeMessagePublisher{err: publishErr},
		&fakeRoomMembershipChecker{isMember: true},
	)

	_, err := service.CreateMessage(
		context.Background(),
		message.RoomID.String(),
		message.UserID.String(),
		message.Content,
		message.ClientMessageID.String(),
	)
	if !errors.Is(err, publishErr) {
		t.Fatalf("CreateMessage() error = %v, want wrapped publish error", err)
	}
}

func TestCreateMessageRejectsInvalidInputBeforeStorage(t *testing.T) {
	repo := &fakeMessageRepository{}
	publisher := &fakeMessagePublisher{}
	service := NewMessageService(repo, publisher, &fakeRoomMembershipChecker{isMember: true})

	_, err := service.CreateMessage(
		context.Background(),
		uuid.NewString(),
		uuid.NewString(),
		" ",
		uuid.NewString(),
	)
	if !errors.Is(err, ErrInvalidMessageInput) {
		t.Fatalf("CreateMessage() error = %v, want ErrInvalidMessageInput", err)
	}
	if publisher.calls != 0 {
		t.Fatalf("Publish() calls = %d, want 0", publisher.calls)
	}
}

func TestCreateMessageRejectsNonMemberBeforeStorage(t *testing.T) {
	repo := &fakeMessageRepository{}
	publisher := &fakeMessagePublisher{}
	service := NewMessageService(
		repo,
		publisher,
		&fakeRoomMembershipChecker{isMember: false},
	)

	_, err := service.CreateMessage(
		context.Background(),
		uuid.NewString(),
		uuid.NewString(),
		"hello",
		uuid.NewString(),
	)
	if !errors.Is(err, ErrNotRoomMember) {
		t.Fatalf("CreateMessage() error = %v, want ErrNotRoomMember", err)
	}
	if repo.createCalls != 0 {
		t.Fatalf("CreateMessage repository calls = %d, want 0", repo.createCalls)
	}
	if publisher.calls != 0 {
		t.Fatalf("Publish() calls = %d, want 0", publisher.calls)
	}
}
