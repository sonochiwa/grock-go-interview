package error_wrapping

import (
	"errors"
	"testing"
)

func TestGetUser(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		u, err := GetUser(1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.Name != "Alice" {
			t.Errorf("got %q, want Alice", u.Name)
		}
	})
	t.Run("not found", func(t *testing.T) {
		_, err := GetUser(999)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestGetUserService(t *testing.T) {
	t.Run("wraps error", func(t *testing.T) {
		_, err := GetUserService(999)
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, ErrNotFound) {
			t.Error("errors.Is should find ErrNotFound in chain")
		}
		if err.Error() == ErrNotFound.Error() {
			t.Error("service should add context, not return raw ErrNotFound")
		}
	})
}

func TestHandleGetUser(t *testing.T) {
	tests := []struct {
		name       string
		id         int
		wantStatus int
		wantErr    bool
	}{
		{"found", 1, 200, false},
		{"not found", 999, 404, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, status, err := HandleGetUser(tt.id)
			if status != tt.wantStatus {
				t.Errorf("status = %d, want %d", status, tt.wantStatus)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
