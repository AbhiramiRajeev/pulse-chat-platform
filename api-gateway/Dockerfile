# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go module files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source tree (needed because it's a monorepo module)
COPY . .

# Build the api-gateway binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /api-gateway ./api-gateway/cmd/gateway

# ---- Final stage ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates

COPY --from=builder /api-gateway /api-gateway

EXPOSE 8080

ENTRYPOINT ["/api-gateway"]
