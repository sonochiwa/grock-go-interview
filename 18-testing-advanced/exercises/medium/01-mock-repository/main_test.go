package mock_repository

import (
	"context"
	"testing"
)

// TODO: напиши тесты для UserService используя MockUserRepository
// TestGetByID_Found, TestGetByID_NotFound, TestGetByID_InvalidID
// TestCreate_Success, TestCreate_DuplicateEmail, TestCreate_EmptyName

func TestGetByID_Found(t *testing.T) {
	// TODO: создай mock с данными, проверь что GetByID возвращает user
}

func TestGetByID_NotFound(t *testing.T) {
	// TODO: создай пустой mock, проверь ErrNotFound
}

func TestCreate_EmptyName(t *testing.T) {
	svc := NewUserService(nil)
	_, err := svc.Create(context.Background(), "", "test@test.com")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

// Подсказка: проверяй mock.Calls для верификации вызовов
// Подсказка: используй mock.ForceError для эмуляции ошибок
