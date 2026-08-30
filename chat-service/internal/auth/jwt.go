package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid token")
)

type JWTManager struct {
	secret []byte
	expiry time.Duration
}

// Claims represents the data stored in the JWT payload. A JWT contains claims (pieces of information about the token).
// Claims is our application's custom struct; the name "Claims" itself is not required by JWT.
// UserID is our custom claim. It tells us which user the tokenbelongs to.

// jwt.RegisteredClaims provides standard JWT claims such as:
// - IssuedAt (iat): when the token was created
// - ExpiresAt (exp): when the token expires
// - Issuer (iss)
// - Subject (sub)
// - Audience (aud)
//
// We embed RegisteredClaims so we can use standard JWT claims
// together with our custom UserID claim.
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

func NewJWTManager(secret string, expiry time.Duration) *JWTManager {
	return &JWTManager{
		secret: []byte(secret),
		expiry: expiry,
	}
}

/*
1. Get the user ID
2. Create claims
3. Add expiry information
4. Create a JWT using those claims
5. Sign the JWT with the secret
6. Return the signed token
*/

func (j *JWTManager) GenerateToken(userID uuid.UUID) (string, error) {
	now := time.Now()

	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.expiry)),
		},
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	signedToken, err := token.SignedString(j.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return signedToken, nil
}

// ValidateToken verifies that a JWT is genuine and valid.
//
// It checks:
//   - The token was signed using our expected signing method.
//   - The signature is valid using our JWT secret.
//   - Standard claims, including expiry, are valid.
//
// If successful, it returns the claims, including the UserID.
func (j *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString,claims,func(token *jwt.Token) (interface{}, error) {
			// Ensure the token uses the signing method we expect.
			if token.Method != jwt.SigningMethodHS256 {
				return nil, ErrInvalidToken
			}

			return j.secret, nil
		},
	)

	if err != nil {
		return nil, ErrInvalidToken
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
