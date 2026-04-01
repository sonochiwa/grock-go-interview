package mock_repository

import (
	"context"
	"errors"
)

var (
	ErrNotFound  = errors.New("user not found")
	ErrDuplicate = errors.New("email already exists")
)

type User struct {
	ID    int
	Name  string
	Email string
}

type UserRepository interface {
	GetByID(ctx context.Context, id int) (User, error)
	Create(ctx context.Context, user User) (int, error)
}

type UserService struct {
	repo UserRepository
}

func NewUserService(repo UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetByID(ctx context.Context, id int) (User, error) {
	if id <= 0 {
		return User{}, errors.New("invalid id")
	}
	return s.repo.GetByID(ctx, id)
}

func (s *UserService) Create(ctx context.Context, name, email string) (User, error) {
	if name == "" {
		return User{}, errors.New("name is required")
	}
	if email == "" {
		return User{}, errors.New("email is required")
	}

	id, err := s.repo.Create(ctx, User{Name: name, Email: email})
	if err != nil {
		return User{}, err
	}
	return User{ID: id, Name: name, Email: email}, nil
}

// TODO: реализуй MockUserRepository
// - Хранит users в map
// - Записывает вызовы (Calls []string)
// - Позволяет настроить ошибки (ForceError error)
type MockUserRepository struct {
	// TODO
}
