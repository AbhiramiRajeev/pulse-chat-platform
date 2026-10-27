# Pulse Chat Platform

A real-time chat backend in Go. You can register, log in, create rooms, add people to them and chat live over WebSockets.

It's split into two services:

- **api-gateway** takes HTTP and WebSocket traffic, checks the JWT, and forwards requests to the chat service over gRPC.
- **chat-service** has the business logic. Users, rooms and membership are in PostgreSQL, messages are in MongoDB, and every new message is published to Redis so the gateway can push it to connected clients.

```
Client
  |  HTTP / WebSocket
  v
API Gateway (:8080)
  |  gRPC (:50051)
  v
Chat Service ---> PostgreSQL  (users, rooms, members)
  |          ---> MongoDB     (messages)
  |
  +--> Redis pub/sub --> Gateway subscriber --> WebSocket hub --> clients
```

Stack: Go 1.26, gRPC/protobuf, PostgreSQL (pgx), MongoDB (official driver), Redis, gorilla/websocket, golang-jwt, Docker Compose.

## What it does

- Register and log in (JWT)
- Create rooms and list the ones you're in
- Room creator can add members
- Send messages and fetch room history, members only
- Live delivery over WebSockets
- Safe retries when sending a message (see `client_message_id` below)
- Graceful shutdown on both services

## Running with Docker

```bash
cp .env.example .env
docker compose -f deployments/docker-compose.yaml up --build
```

This starts Postgres, MongoDB, Redis and both services. The gateway is on `localhost:8080`. Postgres is mapped to `5433` on the host so it doesn't clash with a local Postgres on 5432. If any port clashes on your machine, change the matching `*_HOST_PORT` value in `.env`.

Stop it with:

```bash
docker compose -f deployments/docker-compose.yaml down
```

## Running without Docker

You need Go 1.26+, PostgreSQL, MongoDB and Redis running locally.

Apply the SQL migrations first:

```bash
psql -U postgres -d pulse_chat -f migrations/001_initial_schema.sql
psql -U postgres -d pulse_chat -f migrations/002_create_rooms.sql
psql -U postgres -d pulse_chat -f migrations/003_create_room_members.sql
```

There's no step for messages. They live in MongoDB now, and the chat service creates the collection indexes when it starts. `004_create_messages.sql` is just an empty file so the numbering doesn't change.

Then start the chat service:

```bash
export JWT_SECRET=my-secret-key
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5432
export POSTGRES_USER=postgres
export POSTGRES_PASSWORD=postgres
export POSTGRES_DB=pulse_chat
export REDIS_HOST=localhost
export MONGO_URI=mongodb://localhost:27017
export MONGO_DATABASE=pulse_chat

go run ./chat-service/cmd/chat
```

And the gateway in another terminal:

```bash
export JWT_SECRET=my-secret-key
export CHAT_SERVICE_ADDR=localhost:50051
export REDIS_HOST=localhost

go run ./api-gateway/cmd/gateway
```

## Environment variables

Gateway:

| Variable | Default | |
|---|---|---|
| `HTTP_PORT` | `8080` | |
| `CHAT_SERVICE_ADDR` | `localhost:50051` | gRPC address of the chat service |
| `JWT_SECRET` | none | required |
| `REDIS_HOST` / `REDIS_PORT` | `localhost` / `6379` | |

Chat service:

| Variable | Default | |
|---|---|---|
| `GRPC_PORT` | `50051` | |
| `POSTGRES_HOST` / `POSTGRES_PORT` | `localhost` / `5432` | |
| `POSTGRES_USER` / `POSTGRES_PASSWORD` | `postgres` / `postgres` | |
| `POSTGRES_DB` | `pulse_chat` | |
| `MONGO_URI` | `mongodb://localhost:27017` | |
| `MONGO_DATABASE` | `pulse_chat` | |
| `REDIS_HOST` / `REDIS_PORT` | `localhost` / `6379` | |
| `JWT_SECRET` | none | required, must match the gateway |
| `JWT_EXPIRY` | `24h` | |

## API

Everything except `/register` and `/login` needs an `Authorization: Bearer <token>` header.

| Method | Path | Notes |
|---|---|---|
| POST | `/register` | `{"username", "email", "password"}` |
| POST | `/login` | `{"email", "password"}`, returns a token |
| POST | `/rooms` | `{"name"}`, creator is added automatically |
| GET | `/rooms` | rooms you belong to |
| POST | `/rooms/{roomID}/members` | `{"user_id"}`, creator only |
| GET | `/rooms/{roomID}/members` | members only |
| POST | `/rooms/{roomID}/messages` | `{"content", "client_message_id"}`, members only |
| GET | `/rooms/{roomID}/messages` | room history, members only |

`client_message_id` is a UUID the client generates. If the same one is sent again (a retry after a timeout, for example) you get the original message back. No duplicate is created and nothing is broadcast a second time.

### WebSocket

```
ws://localhost:8080/ws/rooms/{roomID}?token=<JWT>
```

The token goes in the query string because browsers can't set headers on a WebSocket. Membership is checked before the upgrade, so a non-member gets a normal 403. Try it with wscat:

```bash
npm install -g wscat
wscat -c "ws://localhost:8080/ws/rooms/<ROOM_ID>?token=<JWT>"
```

Messages arrive as JSON:

```json
{
  "id": "...",
  "room_id": "...",
  "user_id": "...",
  "content": "Hello!",
  "client_message_id": "...",
  "created_at": "2024-01-01T00:00:00Z"
}
```

## How a message gets delivered

1. Client A sends `POST /rooms/{roomID}/messages` to the gateway.
2. The gateway checks the JWT and room membership, then calls `CreateMessage` on the chat service over gRPC.
3. The chat service checks membership again, saves the message in MongoDB, and publishes it to the Redis channel `room:<roomID>`.
4. The gateway has a goroutine subscribed to `room:*`. When something arrives it hands it to the WebSocket hub.
5. The hub writes it to every connection in that room, so Client B gets it.

## Project structure

```
api-gateway/
  cmd/gateway/        entry point
  internal/
    config/ grpc/ handler/ middleware/ redis/ websocket/
chat-service/
  cmd/chat/           entry point
  internal/
    auth/ config/ grpc/ models/ mongodb/ postgres/ redis/ repository/ service/
proto/                .proto files and generated Go code
migrations/           SQL files, init.sql is the combined one Docker uses
deployments/          docker-compose.yaml
docs/architecture.md
```

## Notes

- Redis pub/sub doesn't keep anything. If the gateway isn't subscribed at the moment a message is published, that live delivery is lost. The message is still saved and shows up in the history endpoint. Redis Streams or Kafka would fix this if guaranteed delivery is ever needed.
- Postgres and MongoDB split: Postgres holds users, rooms and membership, since those are relational and drive permissions. MongoDB holds messages. On startup the chat service creates a unique index on `(user_id, client_message_id)`, which is what stops duplicates, and an index on `(room_id, created_at, _id)` for reading history in order.
- More on the request flow is in [`docs/architecture.md`](docs/architecture.md).

## Quick test with curl

```bash
BASE=http://localhost:8080

# register two users
curl -s -X POST $BASE/register -H "Content-Type: application/json" \
  -d '{"username":"alice","email":"alice@example.com","password":"secret123"}'
curl -s -X POST $BASE/register -H "Content-Type: application/json" \
  -d '{"username":"bob","email":"bob@example.com","password":"secret123"}'

# log in as alice
TOKEN=$(curl -s -X POST $BASE/login -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"secret123"}' | jq -r '.token')

# need bob's id to add him to a room
BOB_ID=$(curl -s -X POST $BASE/login -H "Content-Type: application/json" \
  -d '{"email":"bob@example.com","password":"secret123"}' | jq -r '.id')

# create a room and add bob
ROOM_ID=$(curl -s -X POST $BASE/rooms -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" -d '{"name":"general"}' | jq -r '.id')

curl -s -X POST $BASE/rooms/$ROOM_ID/members -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" -d "{\"user_id\":\"$BOB_ID\"}"

# send a message
curl -s -X POST $BASE/rooms/$ROOM_ID/messages -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"content":"Hello room!","client_message_id":"11111111-1111-1111-1111-111111111111"}'

# read it back
curl -s $BASE/rooms/$ROOM_ID/messages -H "Authorization: Bearer $TOKEN"
```

To see live delivery, open a wscat connection to the room in another terminal before sending the message.
