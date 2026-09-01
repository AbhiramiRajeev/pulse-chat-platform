package repository

import (
	"context"
	"fmt"

	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RoomRepository struct {
	db *pgxpool.Pool
}

func NewRoomRepository(db *pgxpool.Pool) *RoomRepository {
	return &RoomRepository{
		db: db,
	}
}

func (r *RoomRepository) CreateRoom(
	ctx context.Context,
	name string,
	createdBy uuid.UUID,
) (*models.Room, error) {

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	query := `
		INSERT INTO rooms (
			name,
			created_by
		)
		VALUES ($1, $2)
		RETURNING id, name, created_by, created_at
	`

	var room models.Room

	err = tx.QueryRow(
		ctx,
		query,
		name,
		createdBy,
	).Scan(
		&room.ID,
		&room.Name,
		&room.CreatedBy,
		&room.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("create room: %w", err)
	}

	memberQuery := `
		INSERT INTO room_members (
			user_id,
			room_id
		)
		VALUES ($1, $2)
	`

	_, err = tx.Exec(
		ctx,
		memberQuery,
		createdBy,
		room.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("add room creator as member: %w", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &room, nil
}

func (r *RoomRepository) GetRoomsByUserID(
	ctx context.Context,
	userID uuid.UUID,
) ([]models.Room, error) {
	query := `
		SELECT
			r.id,
			r.name,
			r.created_by,
			r.created_at
		FROM rooms r
		INNER JOIN room_members rm
			ON r.id = rm.room_id
		WHERE rm.user_id = $1
		ORDER BY r.created_at DESC
	`

	rows, err := r.db.Query(
		ctx,
		query,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get rooms by user ID: %w", err)
	}
	defer rows.Close()

	var rooms []models.Room

	for rows.Next() {
		var room models.Room

		err := rows.Scan(
			&room.ID,
			&room.Name,
			&room.CreatedBy,
			&room.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan room: %w", err)
		}

		rooms = append(rooms, room)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rooms: %w", err)
	}

	return rooms, nil
}		