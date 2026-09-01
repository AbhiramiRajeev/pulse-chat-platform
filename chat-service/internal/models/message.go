package models

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID              uuid.UUID
	RoomID          uuid.UUID
	UserID          uuid.UUID
	Content         string
	ClientMessageID uuid.UUID
	CreatedAt       time.Time
}