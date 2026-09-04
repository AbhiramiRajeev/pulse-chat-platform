# Pulse Chat Platform

A real-time chat backend built with Go microservices. Features JWT authentication, room-based messaging, room membership management, and live message delivery via WebSockets and Redis Pub/Sub.

---

## Architecture

```
Client
   │
   │  HTTP / WebSocket
   ▼
API Gateway  (:8080)
   │
   │  gRPC (:50051)
   ▼
Chat Service
   │
   ├──► PostgreSQL  (persistent storage)
   │
   └──► Redis Pub/Sub
              │
              ▼
        API Gateway
        Redis Subscriber
              │
              ▼
         WebSocket Hub
              │
              ▼
        Connected Clients
```

### Services

| Service | Port | Role |
|---|---|---|
| `api-gateway` | 8080 (HTTP/WS) | Public-facing gateway; JWT auth, request routing |
| `chat-service` | 50051 (gRPC) | Business logic, PostgreSQL, Redis publishing |
| PostgreSQL | 5432 (5433 on host) | Persistent storage for users, rooms, messages |
| Redis | 6379 | Pub/Sub for real-time message fanout |

---

## Technologies

- **Go 1.23** — both services
- **gRPC / Protocol Buffers** — inter-service communication
- **PostgreSQL 16** — relational storage (pgx driver)
- **Redis 7** — Pub/Sub for real-time delivery
- **WebSockets** — Gorilla WebSocket
- **JWT** — `golang-jwt/jwt/v5` for authentication
- **Docker / Docker Compose** — container orchestration

---

## Features

- User registration and login
- JWT authentication (HTTP `Authorization: Bearer` header)
- Create and list chat rooms
- Room membership management (creator can add members)
- Room membership enforcement on all protected routes
- Send and retrieve messages
- Real-time message delivery via WebSockets (Redis Pub/Sub fanout)
- Message idempotency via `client_message_id` (duplicate-safe retries)
- Graceful shutdown for both services

---

## Running Locally (without Docker)

### Prerequisites

- Go 1.23+
- PostgreSQL 16 running on `localhost:5432` (or `5433`)
- Redis running on `localhost:6379`

### 1 — Apply database migrations

```bash
psql -U postgres -d pulse_chat -f migrations/001_initial_schema.sql
psql -U postgres -d pulse_chat -f migrations/002_create_rooms.sql
psql -U postgres -d pulse_chat -f migrations/003_create_room_members.sql
psql -U postgres -d pulse_chat -f migrations/004_create_messages.sql
```

### 2 — Start the Chat Service

```bash
export JWT_SECRET=my-secret-key
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5432   # or 5433 if you used the Docker-mapped port
export POSTGRES_USER=postgres
export POSTGRES_PASSWORD=postgres
export POSTGRES_DB=pulse_chat
export REDIS_HOST=localhost
export REDIS_PORT=6379

go run ./chat-service/cmd/chat
```

### 3 — Start the API Gateway

```bash
export JWT_SECRET=my-secret-key
export CHAT_SERVICE_ADDR=localhost:50051
export REDIS_HOST=localhost
export REDIS_PORT=6379
export HTTP_PORT=8080

go run ./api-gateway/cmd/gateway
```

---

## Running with Docker Compose

```bash
# Copy and optionally edit environment variables
cp .env.example .env

# Build and start all services
docker compose -f deployments/docker-compose.yaml up --build

# Stop
docker compose -f deployments/docker-compose.yaml down
```

This starts PostgreSQL, Redis, Chat Service, and API Gateway with correct inter-container networking. The database schema is applied automatically on the first run.

---

## Environment Variables

### API Gateway

| Variable | Default | Description |
|---|---|---|
| `HTTP_PORT` | `8080` | HTTP server port |
| `CHAT_SERVICE_ADDR` | `localhost:50051` | gRPC address of chat-service |
| `JWT_SECRET` | *(required)* | JWT signing secret |
| `REDIS_HOST` | `localhost` | Redis hostname |
| `REDIS_PORT` | `6379` | Redis port |

### Chat Service

| Variable | Default | Description |
|---|---|---|
| `GRPC_PORT` | `50051` | gRPC server port |
| `POSTGRES_HOST` | `localhost` | PostgreSQL hostname |
| `POSTGRES_PORT` | `5432` | PostgreSQL port |
| `POSTGRES_USER` | `postgres` | PostgreSQL user |
| `POSTGRES_PASSWORD` | `postgres` | PostgreSQL password |
| `POSTGRES_DB` | `pulse_chat` | PostgreSQL database name |
| `REDIS_HOST` | `localhost` | Redis hostname |
| `REDIS_PORT` | `6379` | Redis port |
| `JWT_SECRET` | *(required)* | JWT signing secret (must match gateway) |
| `JWT_EXPIRY` | `24h` | JWT token expiry duration |

---

## API Endpoints

### Auth

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/register` | None | Register a new user |
| `POST` | `/login` | None | Login and receive JWT |

**Register:**
```json
POST /register
{ "username": "alice", "email": "alice@example.com", "password": "secret" }
```

**Login:**
```json
POST /login
{ "email": "alice@example.com", "password": "secret" }
// Response includes "token"
```

### Rooms

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/rooms` | Bearer JWT | Create a room (creator auto-joined) |
| `GET` | `/rooms` | Bearer JWT | List rooms the user belongs to |
| `POST` | `/rooms/{roomID}/members` | Bearer JWT + member | Add a user to a room (creator only) |
| `GET` | `/rooms/{roomID}/members` | Bearer JWT + member | List room members |

**Create room:**
```json
POST /rooms
Authorization: Bearer <token>
{ "name": "general" }
```

**Add member:**
```json
POST /rooms/{roomID}/members
Authorization: Bearer <token>
{ "user_id": "<uuid>" }
```

### Messages

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/rooms/{roomID}/messages` | Bearer JWT + member | Send a message |
| `GET` | `/rooms/{roomID}/messages` | Bearer JWT + member | Get room message history |

**Send message:**
```json
POST /rooms/{roomID}/messages
Authorization: Bearer <token>
{ "content": "Hello!", "client_message_id": "<uuid>" }
```

`client_message_id` is a UUID you generate client-side. Retrying with the same `client_message_id` returns the existing message — no duplicates are created.

### WebSocket

```
GET /ws/rooms/{roomID}?token=<JWT>
```

Authentication uses the query parameter `token` (JWT). Room membership is verified **before** the HTTP connection is upgraded to WebSocket. Non-members receive `403 Forbidden` at the HTTP level.

---

## WebSocket Usage

```bash
# Install wscat
npm install -g wscat

# Connect (replace values)
wscat -c "ws://localhost:8080/ws/rooms/<ROOM_ID>?token=<JWT>"
```

Once connected, incoming messages are pushed as JSON:

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

---

## Real-Time Message Flow

```
Client A: POST /rooms/{roomID}/messages
    │
    ▼
API Gateway
    │  validates JWT + room membership
    │
    │  gRPC CreateMessage
    ▼
Chat Service
    │  saves to PostgreSQL
    │
    │  PUBLISH room:<roomID>  →  Redis
    ▼
Redis Pub/Sub
    │
    │  PSubscribe("room:*")
    ▼
API Gateway — Redis Subscriber goroutine
    │
    │  hub.Broadcast(roomID, payload)
    ▼
WebSocket Hub
    │  writes to all connections in that room
    ▼
Client B (WebSocket) receives the JSON event
```

---

## Project Structure

```
pulse-chat-platform/
├── api-gateway/
│   ├── cmd/gateway/main.go          # Entry point + graceful shutdown
│   └── internal/
│       ├── config/                  # Config from environment variables
│       ├── grpc/                    # gRPC client (connects to chat-service)
│       ├── handler/                 # HTTP handlers (auth, rooms, messages)
│       ├── middleware/              # Auth, WebSocketAuth, RoomMembership
│       ├── redis/                   # Redis client + Pub/Sub subscriber
│       └── websocket/               # Hub (room connections) + WS handler
├── chat-service/
│   ├── cmd/chat/main.go             # Entry point + graceful shutdown
│   └── internal/
│       ├── auth/                    # JWT manager
│       ├── config/                  # Config from environment variables
│       ├── grpc/                    # gRPC servers (auth, room, message)
│       ├── models/                  # Domain models (User, Room, Message)
│       ├── postgres/                # PostgreSQL connection pool
│       ├── redis/                   # Redis client
│       ├── repository/              # Data access (SQL queries)
│       └── service/                 # Business logic
├── proto/
│   ├── auth.proto                   # AuthService RPC definitions
│   ├── room.proto                   # RoomService RPC definitions
│   ├── message.proto                # MessageService RPC definitions
│   ├── authpb/                      # Generated Go code
│   ├── roompb/
│   └── messagepb/
├── migrations/
│   ├── 001_initial_schema.sql       # users table
│   ├── 002_create_rooms.sql         # rooms table
│   ├── 003_create_room_members.sql  # room_members table
│   ├── 004_create_messages.sql      # messages table
│   └── init.sql                     # Combined init script (used by Docker)
├── deployments/
│   └── docker-compose.yaml
├── .env.example
├── go.mod
└── README.md
```

---

## Design Decisions

### Why API Gateway?
Clients interact with a single HTTP/WebSocket endpoint. The gateway handles authentication, routing, and protocol translation (HTTP → gRPC). Chat Service is never exposed directly.

### Why gRPC?
Strongly-typed contracts between services. Protocol Buffers are more efficient than JSON for inter-service communication. Service definitions double as documentation.

### Why Redis Pub/Sub?
When a message is saved, Chat Service publishes it to `room:<roomID>`. API Gateway subscribes with `PSubscribe("room:*")` and fans out to all WebSocket connections. This decouples message storage from delivery and allows horizontal scaling of the gateway (each instance subscribes to all channels).

**Important limitation:** Redis Pub/Sub is ephemeral. Messages published while no subscriber is connected are lost. For guaranteed delivery, Redis Streams or Kafka would be appropriate.

### Why WebSockets?
Full-duplex TCP connection ideal for real-time push. Membership is verified before the HTTP upgrade — rejections happen at the HTTP level with proper status codes.

### Why PostgreSQL?
Relational model fits chat data well (users, rooms with FK constraints, messages with FK to rooms and users). `UNIQUE(user_id, client_message_id)` constraint enforces idempotency at the database level.

---

## Complete Test Walkthrough

See [`docs/architecture.md`](docs/architecture.md) for full request flow documentation.

```bash
BASE=http://localhost:8080

# 1. Register user Alice
curl -s -X POST $BASE/register \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","email":"alice@example.com","password":"secret123"}'

# 2. Register user Bob
curl -s -X POST $BASE/register \
  -H "Content-Type: application/json" \
  -d '{"username":"bob","email":"bob@example.com","password":"secret123"}'

# 3. Login as Alice (save the token)
TOKEN=$(curl -s -X POST $BASE/login \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"secret123"}' \
  | jq -r '.token')

BOB_ID=$(curl -s -X POST $BASE/login \
  -H "Content-Type: application/json" \
  -d '{"email":"bob@example.com","password":"secret123"}' \
  | jq -r '.id')

# 4. Create a room
ROOM_ID=$(curl -s -X POST $BASE/rooms \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"general"}' \
  | jq -r '.id')

# 5. Add Bob to the room
curl -s -X POST $BASE/rooms/$ROOM_ID/members \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"user_id\":\"$BOB_ID\"}"

# 6. List members
curl -s $BASE/rooms/$ROOM_ID/members \
  -H "Authorization: Bearer $TOKEN"

# 7. Send a message
curl -s -X POST $BASE/rooms/$ROOM_ID/messages \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"content":"Hello room!","client_message_id":"11111111-1111-1111-1111-111111111111"}'

# 8. Try as non-member (should 403)
BOB_TOKEN=$(curl -s -X POST $BASE/login \
  -H "Content-Type: application/json" \
  -d '{"email":"bob@example.com","password":"secret123"}' \
  | jq -r '.token')
# Create another room that Bob doesn't belong to
OTHER_ROOM=$(curl -s -X POST $BASE/rooms \
  -H "Authorization: Bearer $BOB_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"bobs-room"}' | jq -r '.id')
# Alice tries to post to Bob's room (403)
curl -s -o /dev/null -w "%{http_code}" \
  -X POST $BASE/rooms/$OTHER_ROOM/messages \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"content":"sneaky","client_message_id":"22222222-2222-2222-2222-222222222222"}'

# 9. WebSocket — connect as Alice
wscat -c "ws://localhost:8080/ws/rooms/$ROOM_ID?token=$TOKEN"
# In another terminal, send a message — it should appear in wscat output.
```
