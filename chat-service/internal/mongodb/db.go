package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/AbhiramiRajeev/pulse-chat-platform/chat-service/internal/config"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

func NewClient(ctx context.Context, cfg config.Config) (*mongo.Client, *mongo.Database, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		return nil, nil, fmt.Errorf("connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		disconnectCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.Disconnect(disconnectCtx)
		return nil, nil, fmt.Errorf("ping MongoDB: %w", err)
	}

	return client, client.Database(cfg.MongoDatabase), nil
}
