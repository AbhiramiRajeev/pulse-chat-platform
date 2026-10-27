package repository

import (
	"testing"
	"time"

	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/models"
	"github.com/google/uuid"
)

func TestMessageDocumentRoundTrip(t *testing.T) {
	message := &models.Message{
		ID:              uuid.New(),
		RoomID:          uuid.New(),
		UserID:          uuid.New(),
		Content:         "hello",
		ClientMessageID: uuid.New(),
		CreatedAt:       time.Date(2026, time.October, 27, 8, 30, 0, 0, time.UTC),
	}

	got, err := documentFromMessage(message).toMessage()
	if err != nil {
		t.Fatalf("toMessage() error = %v", err)
	}
	if *got != *message {
		t.Fatalf("round trip = %#v, want %#v", got, message)
	}
}

func TestMessageDocumentRejectsInvalidUUID(t *testing.T) {
	document := messageDocument{
		ID:              "not-a-uuid",
		RoomID:          uuid.NewString(),
		UserID:          uuid.NewString(),
		ClientMessageID: uuid.NewString(),
	}

	if _, err := document.toMessage(); err == nil {
		t.Fatal("toMessage() error = nil, want invalid UUID error")
	}
}
