package redis

import (
	"context"
	"fmt"

	"github.com/AbhiramiRajeev/pulse-chat-platform/api-gateway/internal/config"
	goredis "github.com/redis/go-redis/v9"
)

func NewClient(
	ctx context.Context,
	cfg config.Config,
) (*goredis.Client, error) {
	addr := fmt.Sprintf(
		"%s:%s",
		cfg.RedisHost,
		cfg.RedisPort,
	)

	client := goredis.NewClient(
		&goredis.Options{
			Addr: addr,
		},
	)

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf(
			"ping redis: %w",
			err,
		)
	}

	return client, nil
}