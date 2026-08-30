package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/google/uuid"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

func (r *UserRepository) CreateUser(ctx context.Context,username string,email string,passwordHash string,) (*models.User, error) {
	query := `
		INSERT INTO users (
			username,
			email,
			password_hash
		)
		VALUES ($1, $2, $3)
		RETURNING id, username, email, password_hash, created_at, updated_at
	`

	var user models.User

	err := r.db.QueryRow(
		ctx,
		query,
		username,
		email,
		passwordHash,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) GetUserByEmail(
	ctx context.Context,
	email string,
) (*models.User, error) {
	query := `
		SELECT
			id,
			username,
			email,
			password_hash,
			created_at,
			updated_at
		FROM users
		WHERE email = $1
	`

	var user models.User

	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) GetUserByID(
	ctx context.Context,
	id uuid.UUID,
) (*models.User, error) {
	query := `
		SELECT
			id,
			username,
			email,
			password_hash,
			created_at,
			updated_at
		FROM users
		WHERE id = $1
	`

	var user models.User

	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("get user by ID: %w", err)
	}

	return &user, nil
}