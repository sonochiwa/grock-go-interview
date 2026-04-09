package jwt_auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var testSecret = []byte("test-secret-key-32bytes-long!!!!!")

func TestGenerateAndValidate(t *testing.T) {
	token, err := GenerateJWT("user123", testSecret, time.Hour)
	if err != nil {
		t.Fatalf("GenerateJWT error: %v", err)
	}
	if token == "" {
		t.Fatal("empty token")
	}

	claims, err := ValidateJWT(token, testSecret)
	if err != nil {
		t.Fatalf("ValidateJWT error: %v", err)
	}
	if claims.UserID != "user123" {
		t.Errorf("UserID = %q, want user123", claims.UserID)
	}
}

func TestExpired(t *testing.T) {
	token, _ := GenerateJWT("user", testSecret, -time.Hour) // already expired
	_, err := ValidateJWT(token, testSecret)
	if !errors.Is(err, ErrExpiredToken) {
		t.Errorf("expected ErrExpiredToken, got %v", err)
	}
}

func TestWrongSecret(t *testing.T) {
	token, _ := GenerateJWT("user", testSecret, time.Hour)
	_, err := ValidateJWT(token, []byte("wrong-secret"))
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestTamperedToken(t *testing.T) {
	token, _ := GenerateJWT("user", testSecret, time.Hour)
	tampered := token + "x"
	_, err := ValidateJWT(tampered, testSecret)
	if err == nil {
		t.Error("expected error for tampered token")
	}
}

func TestMiddleware(t *testing.T) {
	token, _ := GenerateJWT("user123", testSecret, time.Hour)

	handler := AuthMiddleware(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			t.Error("claims not in context")
		}
		if claims.UserID != "user123" {
			t.Errorf("UserID = %q", claims.UserID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestMiddlewareNoToken(t *testing.T) {
	handler := AuthMiddleware(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach handler")
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}
