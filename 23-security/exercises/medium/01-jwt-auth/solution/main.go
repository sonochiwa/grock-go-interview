package jwt_auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
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

func b64Encode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func b64Decode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

func sign(data, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write(data)
	return mac.Sum(nil)
}

func GenerateJWT(userID string, secret []byte, ttl time.Duration) (string, error) {
	header := b64Encode([]byte(`{"alg":"HS256","typ":"JWT"}`))

	claims := Claims{
		UserID:    userID,
		ExpiresAt: time.Now().Add(ttl),
	}
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payload := b64Encode(payloadJSON)

	sigInput := header + "." + payload
	signature := b64Encode(sign([]byte(sigInput), secret))

	return sigInput + "." + signature, nil
}

func ValidateJWT(tokenStr string, secret []byte) (Claims, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return Claims{}, ErrInvalidToken
	}

	sigInput := parts[0] + "." + parts[1]
	expectedSig := sign([]byte(sigInput), secret)
	gotSig, err := b64Decode(parts[2])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	if !hmac.Equal(expectedSig, gotSig) {
		return Claims{}, ErrInvalidToken
	}

	payloadJSON, err := b64Decode(parts[1])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	var claims Claims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if time.Now().After(claims.ExpiresAt) {
		return Claims{}, ErrExpiredToken
	}
	return claims, nil
}

func AuthMiddleware(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			tokenStr := strings.TrimPrefix(auth, "Bearer ")

			claims, err := ValidateJWT(tokenStr, secret)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	c, ok := ctx.Value(claimsKey).(Claims)
	return c, ok
}
