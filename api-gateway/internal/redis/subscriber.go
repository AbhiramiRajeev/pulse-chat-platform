package redis

import (
	"context"
	"strings"

	chatws "github.com/AbhiramiRajeev/pulse-chat-platform/api-gateway/internal/websocket"
	goredis "github.com/redis/go-redis/v9"
)

const MessageChannelPattern = "room:*"

func SubscribeToMessages(
	ctx context.Context,
	client *goredis.Client,
	hub *chatws.Hub,
) {
	pubsub := client.PSubscribe(
		ctx,
		MessageChannelPattern,
	)

	ch := pubsub.Channel()
	//Chat Service publishes to: room:6b4536e4-4b48-4ecc-99c9-d67e6790f7e4
	for message := range ch {
		roomID := strings.TrimPrefix(
			message.Channel,
			"room:",
		)

		//sends that payload to all WebSocket connections currently stored for that room in this Gateway.
		hub.Broadcast(
			roomID,
			[]byte(message.Payload),
		)
	}
}