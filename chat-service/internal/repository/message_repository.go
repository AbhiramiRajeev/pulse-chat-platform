package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrDuplicateMessage is returned when a message with the same (user_id, client_message_id) already exists.
var ErrDuplicateMessage = errors.New("duplicate message")

type MessageRepository struct {
	db *pgxpool.Pool
}

func NewMessageRepository(db *pgxpool.Pool) *MessageRepository {
	return &MessageRepository{
		db: db,
	}
}

func (r *MessageRepository) CreateMessage(
	ctx context.Context,
	roomID uuid.UUID,
	userID uuid.UUID,
	content string,
	clientMessageID uuid.UUID,
) (*models.Message, error) {

	query := `
		INSERT INTO messages (
			room_id,
			user_id,
			content,
			client_message_id
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id, room_id, user_id, content, client_message_id, created_at
	`

	message := &models.Message{}

	err := r.db.QueryRow(
		ctx,
		query,
		roomID,
		userID,
		content,
		clientMessageID,
	).Scan(
		&message.ID,
		&message.RoomID,
		&message.UserID,
		&message.Content,
		&message.ClientMessageID,
		&message.CreatedAt,
	)

	if err != nil {
		// PostgreSQL unique_violation error code is 23505.
		// When (user_id, client_message_id) already exists, fetch and return the existing message.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			existing, fetchErr := r.getMessageByClientID(ctx, userID, clientMessageID)
			if fetchErr != nil {
				return nil, fmt.Errorf("create message: %w", err)
			}
			return existing, ErrDuplicateMessage
		}
		return nil, fmt.Errorf("create message: %w", err)
	}

	return message, nil
}

// getMessageByClientID fetches an existing message by (user_id, client_message_id).
func (r *MessageRepository) getMessageByClientID(
	ctx context.Context,
	userID uuid.UUID,
	clientMessageID uuid.UUID,
) (*models.Message, error) {
	query := `
		SELECT id, room_id, user_id, content, client_message_id, created_at
		FROM messages
		WHERE user_id = $1 AND client_message_id = $2
	`
	message := &models.Message{}
	err := r.db.QueryRow(ctx, query, userID, clientMessageID).Scan(
		&message.ID,
		&message.RoomID,
		&message.UserID,
		&message.Content,
		&message.ClientMessageID,
		&message.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("message not found after duplicate error")
	}
	if err != nil {
		return nil, fmt.Errorf("fetch existing message: %w", err)
	}
	return message, nil
}

func (r *MessageRepository) GetMessagesByRoomID(
	ctx context.Context,
	roomID uuid.UUID,
) ([]*models.Message, error) {

	query := `
		SELECT id, room_id, user_id, content, client_message_id, created_at
		FROM messages
		WHERE room_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(ctx, query, roomID)
	if err != nil {
		return nil, fmt.Errorf("get messages by room ID: %w", err)
	}
	defer rows.Close()

	var messages []*models.Message

	for rows.Next() {
		message := &models.Message{}

		err := rows.Scan(
			&message.ID,
			&message.RoomID,
			&message.UserID,
			&message.Content,
			&message.ClientMessageID,
			&message.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}

		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate messages: %w", err)
	}

	return messages, nil
}