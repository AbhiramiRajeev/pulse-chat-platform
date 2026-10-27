package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/models"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ErrDuplicateMessage is returned when a message with the same (user_id, client_message_id) already exists.
var ErrDuplicateMessage = errors.New("duplicate message")

const messageCollection = "messages"

type MessageRepository struct {
	collection *mongo.Collection
}

type messageDocument struct {
	ID              string    `bson:"_id"`
	RoomID          string    `bson:"room_id"`
	UserID          string    `bson:"user_id"`
	Content         string    `bson:"content"`
	ClientMessageID string    `bson:"client_message_id"`
	CreatedAt       time.Time `bson:"created_at"`
}

func NewMessageRepository(db *mongo.Database) *MessageRepository {
	return &MessageRepository{
		collection: db.Collection(messageCollection),
	}
}

func (r *MessageRepository) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "user_id", Value: 1},
				{Key: "client_message_id", Value: 1},
			},
			Options: options.Index().
				SetName("unique_user_client_message").
				SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "room_id", Value: 1},
				{Key: "created_at", Value: 1},
				{Key: "_id", Value: 1},
			},
			Options: options.Index().SetName("room_message_history"),
		},
	}

	if _, err := r.collection.Indexes().CreateMany(ctx, indexes); err != nil {
		return fmt.Errorf("create message indexes: %w", err)
	}

	return nil
}

func (r *MessageRepository) CreateMessage(
	ctx context.Context,
	roomID uuid.UUID,
	userID uuid.UUID,
	content string,
	clientMessageID uuid.UUID,
) (*models.Message, error) {
	message := &models.Message{
		ID:              uuid.New(),
		RoomID:          roomID,
		UserID:          userID,
		Content:         content,
		ClientMessageID: clientMessageID,
		CreatedAt:       time.Now().UTC(),
	}

	_, err := r.collection.InsertOne(ctx, documentFromMessage(message))
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			existing, fetchErr := r.getMessageByClientID(ctx, userID, clientMessageID)
			if fetchErr != nil {
				return nil, fmt.Errorf("resolve duplicate message after insert failure: %w", fetchErr)
			}
			return existing, ErrDuplicateMessage
		}
		return nil, fmt.Errorf("insert message: %w", err)
	}

	return message, nil
}

func (r *MessageRepository) getMessageByClientID(
	ctx context.Context,
	userID uuid.UUID,
	clientMessageID uuid.UUID,
) (*models.Message, error) {
	var document messageDocument
	err := r.collection.FindOne(ctx, bson.D{
		{Key: "user_id", Value: userID.String()},
		{Key: "client_message_id", Value: clientMessageID.String()},
	}).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, errors.New("message not found after duplicate key error")
	}
	if err != nil {
		return nil, fmt.Errorf("find existing message: %w", err)
	}

	return document.toMessage()
}

func (r *MessageRepository) GetMessagesByRoomID(
	ctx context.Context,
	roomID uuid.UUID,
) ([]*models.Message, error) {
	cursor, err := r.collection.Find(
		ctx,
		bson.D{{Key: "room_id", Value: roomID.String()}},
		options.Find().SetSort(bson.D{
			{Key: "created_at", Value: 1},
			{Key: "_id", Value: 1},
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("find messages by room ID: %w", err)
	}
	defer cursor.Close(ctx)

	var documents []messageDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, fmt.Errorf("decode room messages: %w", err)
	}

	messages := make([]*models.Message, 0, len(documents))
	for _, document := range documents {
		message, err := document.toMessage()
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}

	return messages, nil
}

func documentFromMessage(message *models.Message) messageDocument {
	return messageDocument{
		ID:              message.ID.String(),
		RoomID:          message.RoomID.String(),
		UserID:          message.UserID.String(),
		Content:         message.Content,
		ClientMessageID: message.ClientMessageID.String(),
		CreatedAt:       message.CreatedAt,
	}
}

func (d messageDocument) toMessage() (*models.Message, error) {
	id, err := uuid.Parse(d.ID)
	if err != nil {
		return nil, fmt.Errorf("decode message ID %q: %w", d.ID, err)
	}
	roomID, err := uuid.Parse(d.RoomID)
	if err != nil {
		return nil, fmt.Errorf("decode message room ID %q: %w", d.RoomID, err)
	}
	userID, err := uuid.Parse(d.UserID)
	if err != nil {
		return nil, fmt.Errorf("decode message user ID %q: %w", d.UserID, err)
	}
	clientMessageID, err := uuid.Parse(d.ClientMessageID)
	if err != nil {
		return nil, fmt.Errorf("decode client message ID %q: %w", d.ClientMessageID, err)
	}

	return &models.Message{
		ID:              id,
		RoomID:          roomID,
		UserID:          userID,
		Content:         d.Content,
		ClientMessageID: clientMessageID,
		CreatedAt:       d.CreatedAt,
	}, nil
}
