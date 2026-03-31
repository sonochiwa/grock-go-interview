package jwt_auth

import (
	"context"
	"errors"
	"net/http"
	"time"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

type Claims struct {
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"exp"`
}

type contextKey string

const claimsKey contextKey = "claims"

// TODO: создай JWT token: base64url(header).base64url(payload).base64url(HMAC-SHA256(header.payload, secret))
// header: {"alg":"HS256","typ":"JWT"}
// payload: JSON(Claims)
func GenerateJWT(userID string, secret []byte, ttl time.Duration) (string, error) {
	return "", nil
}

// TODO: распарси и проверь JWT
// 1. Разбей на 3 части по "."
// 2. Проверь подпись (HMAC-SHA256)
// 3. Декодируй payload → Claims
// 4. Проверь ExpiresAt
func ValidateJWT(tokenStr string, secret []byte) (Claims, error) {
	return Claims{}, ErrInvalidToken
}

// TODO: middleware — извлеки Bearer token из Authorization header, validate, положи claims в context
func AuthMiddleware(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return next
	}
}

// Helper: достать claims из context
func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	c, ok := ctx.Value(claimsKey).(Claims)
	return c, ok
}
