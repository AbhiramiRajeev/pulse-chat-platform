package postgres

import (
	"context"
	"fmt"

	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, cfg config.Config) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		cfg.PostgresUser,
		cfg.PostgresPassword,
		cfg.PostgresHost,
		cfg.PostgresPort,
		cfg.PostgresDB,
	)

	pool,err:= pgxpool.New(ctx,dsn)

	if err!=nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err:= pool.Ping(ctx);err!=nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool,nil
}
