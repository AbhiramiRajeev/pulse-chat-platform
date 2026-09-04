# Architecture

## Overview

Pulse Chat Platform is a Go microservices application with two services:

- **api-gateway** — public HTTP/WebSocket edge service
- **chat-service** — internal gRPC service owning all business logic and data

Infrastructure:

- **PostgreSQL** — persistent relational storage
- **Redis** — Pub/Sub for real-time message fanout

---

## Service Responsibilities

### API Gateway

- Validates JWT tokens
- Enforces room membership before forwarding requests
- Translates HTTP requests into gRPC calls to chat-service
- Upgrades WebSocket connections and manages the connection Hub
- Subscribes to Redis Pub/Sub and fans out messages to WebSocket clients

### Chat Service

- Implements gRPC servers: `AuthService`, `RoomService`, `MessageService`
- Owns all database operations (via pgx connection pool)
- Publishes new messages to Redis after writing to PostgreSQL
- Generates and validates JWTs

---

## Request Flows

### Auth Flow — Register

```
Client
  │
  │  POST /register { username, email, password }
  ▼
API Gateway
  │  no auth required
  │  gRPC: AuthService.Register
  ▼
Chat Service (AuthGRPCServer)
  │
  ▼
AuthService
  │  validates input, hashes password (bcrypt)
  │  checks for duplicate email
  ▼
UserRepository
  │  INSERT INTO users
  ▼
PostgreSQL
  │
  ▼
Response: { id, username, email }
```

### Auth Flow — Login

```
Client
  │
  │  POST /login { email, password }
  ▼
API Gateway
  │  gRPC: AuthService.Login
  ▼
Chat Service (AuthGRPCServer)
  │
  ▼
AuthService
  │  fetches user by email
  │  bcrypt.CompareHashAndPassword
  │  generates JWT (UserID claim, expiry)
  ▼
Response: { id, username, email, token }
```

### Create Room Flow

```
Client
  │
  │  POST /rooms { name }
  │  Authorization: Bearer <JWT>
  ▼
API Gateway
  │
  ├─ Auth middleware
  │    reads Authorization header
  │    validates JWT
  │    stores userID in request context
  │
  │  gRPC: RoomService.CreateRoom
  ▼
Chat Service (RoomServer)
  │
  ▼
RoomService → RoomRepository
  │  BEGIN TRANSACTION
  │  INSERT INTO rooms (name, created_by)
  │  INSERT INTO room_members (user_id, room_id)  ← creator is auto-added
  │  COMMIT
  ▼
PostgreSQL
  │
  ▼
Response: { id, name, created_by, created_at }
```

### Add Member Flow

```
Client
  │
  │  POST /rooms/{roomID}/members { user_id }
  │  Authorization: Bearer <JWT>
  ▼
API Gateway
  │
  ├─ Auth middleware (validates JWT, stores userID)
  │
  ├─ RoomMembership middleware
  │    calls gRPC: RoomService.IsUserInRoom(roomID, requestingUserID)
  │    returns 403 if requester is not in the room
  │
  │  gRPC: RoomService.AddMember
  ▼
Chat Service (RoomServer)
  │
  ▼
RoomRepository.AddMember
  │  SELECT created_by FROM rooms WHERE id = roomID
  │  if requester != creator → PermissionDenied (403)
  │  if user already member → AlreadyExists (409)
  │  INSERT INTO room_members (user_id, room_id)
  ▼
PostgreSQL
```

### Send Message Flow

```
Client
  │
  │  POST /rooms/{roomID}/messages { content, client_message_id }
  │  Authorization: Bearer <JWT>
  ▼
API Gateway
  │
  ├─ Auth middleware (validates JWT)
  │
  ├─ RoomMembership middleware (checks membership via gRPC)
  │
  │  gRPC: MessageService.CreateMessage
  ▼
Chat Service (MessageGRPCServer)
  │
  ▼
MessageService.CreateMessage
  │
  ▼
MessageRepository.CreateMessage
  │  INSERT INTO messages (room_id, user_id, content, client_message_id)
  │
  │  On UNIQUE(user_id, client_message_id) violation:
  │    → fetch existing message (idempotent retry)
  │    → return it without re-publishing
  ▼
PostgreSQL
  │
  ▼
(if new message) Redis PUBLISH room:<roomID> { id, room_id, user_id, content, ... }
  │
  ▼
Redis Pub/Sub
  │
  ├─ API Gateway subscriber goroutine (PSubscribe "room:*")
  │    extracts roomID from channel name
  │    hub.Broadcast(roomID, payload)
  │
  ▼
WebSocket Hub
  │  acquires read lock, snapshots connection list, releases lock
  │  for each conn: conn.writeMessage(TextMessage, payload)
  │    (each conn has its own write mutex — gorilla WS safe)
  │  removes failed connections under write lock
  ▼
Connected WebSocket clients receive JSON event
```

### WebSocket Connect Flow

```
Client
  │
  │  GET /ws/rooms/{roomID}?token=<JWT>
  ▼
API Gateway
  │
  ├─ WebSocketAuth middleware
  │    reads ?token=<JWT>
  │    validates JWT
  │    stores userID in request context
  │
  ├─ RoomMembership middleware
  │    calls gRPC: RoomService.IsUserInRoom
  │    returns 403 if not a member
  │    ← membership check happens BEFORE HTTP upgrade
  │
  ├─ WebSocket handler
  │    upgrader.Upgrade(w, r) → gorilla WebSocket conn
  │    hub.JoinRoom(roomID, conn) → wrapped *conn (with write mutex)
  │
  │  Read loop (blocks until disconnect)
  │    conn.ReadMessage() → error on disconnect
  │
  │  On disconnect:
  │    hub.LeaveRoom(roomID, conn)
  │    conn.close()
  ▼
Client disconnected, resources cleaned up
```

---

## Concurrency Design

### WebSocket Hub

The Hub stores rooms as `map[string]map[*conn]bool`. Access is protected by `sync.RWMutex`.

**Broadcast** snapshots the connection set under an RLock, then releases the lock before writing. Each `*conn` wraps a gorilla WebSocket connection with its own `sync.Mutex` to serialise writes — gorilla WebSocket requires that only one goroutine writes to a connection at a time. Failed connections are removed under a separate write lock after the broadcast pass.

### Redis Subscriber

A single goroutine reads from the Redis PubSub channel and calls `hub.Broadcast`. The Hub handles concurrent callers safely.

---

## Database Schema

```sql
-- Users
users (id UUID PK, username TEXT UNIQUE, email TEXT UNIQUE,
       password_hash TEXT, created_at, updated_at)

-- Rooms
rooms (id UUID PK, name TEXT, created_by UUID FK→users, created_at)

-- Room membership
room_members (user_id UUID FK→users, room_id UUID FK→rooms,
              joined_at, PRIMARY KEY(user_id, room_id))

-- Messages
messages (id UUID PK, room_id UUID FK→rooms, user_id UUID FK→users,
          content TEXT, client_message_id UUID,
          created_at, UNIQUE(user_id, client_message_id))
```

The `UNIQUE(user_id, client_message_id)` constraint is the database-level guarantee for message idempotency. If a client retries a message send, the repository detects the `23505` unique violation, fetches the existing message, and returns it — no duplicate is stored, no second Redis publish occurs.

---

## Security

### Authentication

Every protected HTTP route goes through the `Auth` middleware:
1. Reads `Authorization: Bearer <token>`
2. Validates JWT signature and expiry
3. Extracts `UserID` from claims
4. Stores in `context.Context` using a typed key (`UserIDKey`)

WebSocket routes use `WebSocketAuth` which reads `?token=<JWT>` and follows the same logic.

### Room Membership Authorisation

The `RoomMembership` middleware runs after authentication on all room-scoped routes (`/rooms/{roomID}/...` and `/ws/rooms/{roomID}`). It calls `RoomService.IsUserInRoom` via gRPC. Non-members receive `403 Forbidden`.

For `AddMember`, the repository additionally checks that the requesting user is the room creator (`rooms.created_by`). Non-creators receive `PermissionDenied`.

---

## Known Limitations

1. **Redis Pub/Sub is ephemeral.** If the API Gateway restarts while Chat Service publishes messages, those messages are lost. For stronger delivery guarantees, consider Redis Streams or Kafka.

2. **No message pagination.** `GET /rooms/{roomID}/messages` returns all messages for the room. For large rooms, add `LIMIT/OFFSET` or cursor-based pagination.

3. **No WebSocket ping/keepalive.** Long-idle connections may be silently dropped by load balancers. Production deployments should add periodic pings.

4. **Single gRPC connection.** The API Gateway holds one gRPC connection to Chat Service. For higher throughput, a connection pool or gRPC client-side load balancing would be appropriate.

5. **No rate limiting.** Authentication endpoints are not rate-limited.

6. **Passwords in gRPC.** The `Register` and `Login` RPC calls transmit passwords over the gRPC channel. In production, enable TLS.
