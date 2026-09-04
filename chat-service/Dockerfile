# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go module files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source tree (needed because it's a monorepo module)
COPY . .

# Build the chat-service binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /chat-service ./chat-service/cmd/chat

# ---- Final stage ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates

COPY --from=builder /chat-service /chat-service

EXPOSE 50051

ENTRYPOINT ["/chat-service"]
