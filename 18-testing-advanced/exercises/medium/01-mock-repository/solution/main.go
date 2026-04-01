package mock_repository

import (
	"context"
	"errors"
	"testing"
)

// --- Mock ---

type User struct {
	ID    int
	Name  string
	Email string
}

var (
	ErrNotFound  = errors.New("user not found")
	ErrDuplicate = errors.New("email already exists")
)

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

// --- Mock Implementation ---

type MockUserRepository struct {
	Users      map[int]User
	Emails     map[string]bool
	NextID     int
	Calls      []string
	ForceError error
}

func NewMockRepo() *MockUserRepository {
	return &MockUserRepository{
		Users:  make(map[int]User),
		Emails: make(map[string]bool),
		NextID: 1,
	}
}

func (m *MockUserRepository) GetByID(_ context.Context, id int) (User, error) {
	m.Calls = append(m.Calls, "GetByID")
	if m.ForceError != nil {
		return User{}, m.ForceError
	}
	u, ok := m.Users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

func (m *MockUserRepository) Create(_ context.Context, user User) (int, error) {
	m.Calls = append(m.Calls, "Create")
	if m.ForceError != nil {
		return 0, m.ForceError
	}
	if m.Emails[user.Email] {
		return 0, ErrDuplicate
	}
	id := m.NextID
	m.NextID++
	user.ID = id
	m.Users[id] = user
	m.Emails[user.Email] = true
	return id, nil
}

// --- Solution Tests ---

func TestGetByID_FoundSolution(t *testing.T) {
	mock := NewMockRepo()
	mock.Users[1] = User{ID: 1, Name: "Alice", Email: "alice@test.com"}
	svc := NewUserService(mock)

	u, err := svc.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Name != "Alice" {
		t.Errorf("Name = %q, want Alice", u.Name)
	}
	if len(mock.Calls) != 1 || mock.Calls[0] != "GetByID" {
		t.Errorf("Calls = %v, want [GetByID]", mock.Calls)
	}
}

func TestGetByID_NotFoundSolution(t *testing.T) {
	mock := NewMockRepo()
	svc := NewUserService(mock)

	_, err := svc.GetByID(context.Background(), 999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetByID_InvalidID(t *testing.T) {
	svc := NewUserService(nil)
	_, err := svc.GetByID(context.Background(), -1)
	if err == nil {
		t.Fatal("expected error for negative id")
	}
}

func TestCreate_SuccessSolution(t *testing.T) {
	mock := NewMockRepo()
	svc := NewUserService(mock)

	u, err := svc.Create(context.Background(), "Bob", "bob@test.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != 1 || u.Name != "Bob" {
		t.Errorf("got %+v", u)
	}
}

func TestCreate_DuplicateEmail(t *testing.T) {
	mock := NewMockRepo()
	mock.Emails["dup@test.com"] = true
	svc := NewUserService(mock)

	_, err := svc.Create(context.Background(), "Dup", "dup@test.com")
	if !errors.Is(err, ErrDuplicate) {
		t.Errorf("expected ErrDuplicate, got %v", err)
	}
}

func TestCreate_EmptyNameSolution(t *testing.T) {
	svc := NewUserService(nil)
	_, err := svc.Create(context.Background(), "", "x@test.com")
	if err == nil {
		t.Fatal("expected error")
	}
}
