package clean_layers

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// --- Domain ---

type TaskStatus string

const (
	StatusTodo  TaskStatus = "todo"
	StatusDoing TaskStatus = "doing"
	StatusDone  TaskStatus = "done"
)

type Task struct {
	ID        int
	Title     string
	Status    TaskStatus
	CreatedAt time.Time
}

var (
	ErrNotFound      = errors.New("task not found")
	ErrEmptyTitle    = errors.New("title is required")
	ErrInvalidStatus = errors.New("invalid status")
)

// --- Repository Interface ---

type TaskRepository interface {
	Save(ctx context.Context, task *Task) error
	FindByID(ctx context.Context, id int) (*Task, error)
	FindAll(ctx context.Context) ([]*Task, error)
	Update(ctx context.Context, task *Task) error
	Delete(ctx context.Context, id int) error
}

// --- Use Case Interface ---

type TaskUseCase interface {
	Create(ctx context.Context, title string) (*Task, error)
	GetByID(ctx context.Context, id int) (*Task, error)
	List(ctx context.Context) ([]*Task, error)
	UpdateStatus(ctx context.Context, id int, status TaskStatus) error
	Delete(ctx context.Context, id int) error
}

// --- In-Memory Repository ---
// TODO: реализуй все методы TaskRepository

type InMemoryRepo struct {
	mu    sync.RWMutex
	tasks map[int]*Task
	seq   atomic.Int32
}

func NewInMemoryRepo() *InMemoryRepo {
	return &InMemoryRepo{tasks: make(map[int]*Task)}
}

func (r *InMemoryRepo) Save(ctx context.Context, task *Task) error {
	return nil // TODO
}

func (r *InMemoryRepo) FindByID(ctx context.Context, id int) (*Task, error) {
	return nil, ErrNotFound // TODO
}

func (r *InMemoryRepo) FindAll(ctx context.Context) ([]*Task, error) {
	return nil, nil // TODO
}

func (r *InMemoryRepo) Update(ctx context.Context, task *Task) error {
	return nil // TODO
}

func (r *InMemoryRepo) Delete(ctx context.Context, id int) error {
	return nil // TODO
}

// --- Use Case Implementation ---
// TODO: реализуй бизнес-логику + валидацию

type TaskService struct {
	repo TaskRepository
}

func NewTaskService(repo TaskRepository) *TaskService {
	return &TaskService{repo: repo}
}

func (s *TaskService) Create(ctx context.Context, title string) (*Task, error) {
	return nil, nil // TODO: validate title, create task with StatusTodo
}

func (s *TaskService) GetByID(ctx context.Context, id int) (*Task, error) {
	return nil, nil // TODO
}

func (s *TaskService) List(ctx context.Context) ([]*Task, error) {
	return nil, nil // TODO
}

func (s *TaskService) UpdateStatus(ctx context.Context, id int, status TaskStatus) error {
	return nil // TODO: validate status, get task, update
}

func (s *TaskService) Delete(ctx context.Context, id int) error {
	return nil // TODO
}
