package repository

import (
	"context"
	"fmt"
	"time"

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

func (r *RoomRepository) IsUserInRoom(
	ctx context.Context,
	roomID uuid.UUID,
	userID uuid.UUID,
) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM room_members
			WHERE room_id = $1
			AND user_id = $2
		)
	`

	var isMember bool

	err := r.db.QueryRow(
		ctx,
		query,
		roomID,
		userID,
	).Scan(&isMember)

	if err != nil {
		return false, fmt.Errorf(
			"check room membership: %w",
			err,
		)
	}

	return isMember, nil
}

func (r *RoomRepository) AddMember(
	ctx context.Context,
	roomID uuid.UUID,
	userID uuid.UUID,
	requestingUserID uuid.UUID,
) error {
	// Verify room exists and check if requesting user is the creator
	var createdBy uuid.UUID
	query := `
		SELECT created_by FROM rooms WHERE id = $1
	`
	err := r.db.QueryRow(ctx, query, roomID).Scan(&createdBy)
	if err != nil {
		return fmt.Errorf("get room: %w", err)
	}

	// Only room creator can add members
	if createdBy != requestingUserID {
		return fmt.Errorf("only room creator can add members")
	}

	// Check if user is already a member
	isMember, err := r.IsUserInRoom(ctx, roomID, userID)
	if err != nil {
		return err
	}
	if isMember {
		return fmt.Errorf("user is already a member")
	}

	// Add member
	insertQuery := `
		INSERT INTO room_members (user_id, room_id)
		VALUES ($1, $2)
	`
	_, err = r.db.Exec(ctx, insertQuery, userID, roomID)
	if err != nil {
		return fmt.Errorf("add member: %w", err)
	}

	return nil
}

type RoomMember struct {
	UserID   uuid.UUID
	Username string
	Email    string
	JoinedAt time.Time
}

func (r *RoomRepository) ListMembers(
	ctx context.Context,
	roomID uuid.UUID,
) ([]RoomMember, error) {
	query := `
		SELECT u.id, u.username, u.email, rm.joined_at
		FROM room_members rm
		INNER JOIN users u ON rm.user_id = u.id
		WHERE rm.room_id = $1
		ORDER BY rm.joined_at ASC
	`

	rows, err := r.db.Query(ctx, query, roomID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	var members []RoomMember
	for rows.Next() {
		var member RoomMember
		err := rows.Scan(&member.UserID, &member.Username, &member.Email, &member.JoinedAt)
		if err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		members = append(members, member)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate members: %w", err)
	}

	return members, nil
}
