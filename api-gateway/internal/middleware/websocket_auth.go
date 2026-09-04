package middleware

import (
	"context"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func WebSocketAuth(
	secret string,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {

				tokenString := r.URL.Query().Get("token")

				if tokenString == "" {
					http.Error(
						w,
						"missing token",
						http.StatusUnauthorized,
					)
					return
				}

				claims := &Claims{}

				token, err := jwt.ParseWithClaims(
					tokenString,
					claims,
					func(token *jwt.Token) (interface{}, error) {
						return []byte(secret), nil
					},
				)

				if err != nil || !token.Valid {
					http.Error(
						w,
						"invalid or expired token",
						http.StatusUnauthorized,
					)
					return
				}

				userID, err := uuid.Parse(
					claims.UserID.String(),
				)
				if err != nil {
					http.Error(
						w,
						"invalid user ID",
						http.StatusUnauthorized,
					)
					return
				}

				ctx := context.WithValue(
					r.Context(),
					UserIDKey,
					userID,
				)

				next.ServeHTTP(
					w,
					r.WithContext(ctx),
				)
			},
		)
	}
}
