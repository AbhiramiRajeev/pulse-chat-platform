package middleware

import (
	"context"
	"net/http"
	"time"

	gatewaygrpc "github.com/AbhiramiRajeev/pulse-chat-platform/api-gateway/internal/grpc"
	roompb "github.com/AbhiramiRajeev/pulse-chat-platform/proto/roompb"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RoomMembership(
	clients *gatewaygrpc.Clients,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			roomID := r.PathValue("roomID")
			if roomID == "" {
				http.Error(w, "room ID is required", http.StatusBadRequest)
				return
			}

			userID, ok := r.Context().Value(UserIDKey).(uuid.UUID)
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()

			response, err := clients.RoomClient.IsUserInRoom(
				ctx,
				&roompb.IsUserInRoomRequest{
					RoomId: roomID,
					UserId: userID.String(),
				},
			)
			if err != nil {
				if grpcStatus, ok := status.FromError(err); ok {
					switch grpcStatus.Code() {
					case codes.InvalidArgument:
						http.Error(w, grpcStatus.Message(), http.StatusBadRequest)
						return
					case codes.Unauthenticated:
						http.Error(w, grpcStatus.Message(), http.StatusUnauthorized)
						return
					case codes.PermissionDenied:
						http.Error(w, grpcStatus.Message(), http.StatusForbidden)
						return
					case codes.NotFound:
						http.Error(w, grpcStatus.Message(), http.StatusNotFound)
						return
					default:
						http.Error(w, "failed to check room membership", http.StatusInternalServerError)
						return
					}
				}

				http.Error(w, "failed to check room membership", http.StatusInternalServerError)
				return
			}

			if !response.GetIsMember() {
				http.Error(w, "forbidden: not a room member", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
