package handler

import (
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func handleGRPCError(w http.ResponseWriter, err error) {
	grpcStatus, ok := status.FromError(err)

	if !ok {
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	switch grpcStatus.Code() {
	case codes.InvalidArgument:
		http.Error(w, grpcStatus.Message(), http.StatusBadRequest)

	case codes.AlreadyExists:
		http.Error(w, grpcStatus.Message(), http.StatusConflict)

	case codes.Unauthenticated:
		http.Error(w, grpcStatus.Message(), http.StatusUnauthorized)

	case codes.PermissionDenied:
		http.Error(w, grpcStatus.Message(), http.StatusForbidden)

	case codes.NotFound:
		http.Error(w, grpcStatus.Message(), http.StatusNotFound)

	default:
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}