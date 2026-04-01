package clean_layers

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

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

type TaskRepository interface {
	Save(ctx context.Context, task *Task) error
	FindByID(ctx context.Context, id int) (*Task, error)
	FindAll(ctx context.Context) ([]*Task, error)
	Update(ctx context.Context, task *Task) error
	Delete(ctx context.Context, id int) error
}

type TaskUseCase interface {
	Create(ctx context.Context, title string) (*Task, error)
	GetByID(ctx context.Context, id int) (*Task, error)
	List(ctx context.Context) ([]*Task, error)
	UpdateStatus(ctx context.Context, id int, status TaskStatus) error
	Delete(ctx context.Context, id int) error
}

// --- In-Memory Repository ---

type InMemoryRepo struct {
	mu    sync.RWMutex
	tasks map[int]*Task
	seq   atomic.Int32
}

func NewInMemoryRepo() *InMemoryRepo {
	return &InMemoryRepo{tasks: make(map[int]*Task)}
}

func (r *InMemoryRepo) Save(_ context.Context, task *Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := int(r.seq.Add(1))
	task.ID = id
	clone := *task
	r.tasks[id] = &clone
	return nil
}

func (r *InMemoryRepo) FindByID(_ context.Context, id int) (*Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tasks[id]
	if !ok {
		return nil, ErrNotFound
	}
	clone := *t
	return &clone, nil
}

func (r *InMemoryRepo) FindAll(_ context.Context) ([]*Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*Task, 0, len(r.tasks))
	for _, t := range r.tasks {
		clone := *t
		result = append(result, &clone)
	}
	return result, nil
}

func (r *InMemoryRepo) Update(_ context.Context, task *Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tasks[task.ID]; !ok {
		return ErrNotFound
	}
	clone := *task
	r.tasks[task.ID] = &clone
	return nil
}

func (r *InMemoryRepo) Delete(_ context.Context, id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tasks[id]; !ok {
		return ErrNotFound
	}
	delete(r.tasks, id)
	return nil
}

// --- Use Case ---

type TaskService struct {
	repo TaskRepository
}

func NewTaskService(repo TaskRepository) *TaskService {
	return &TaskService{repo: repo}
}

func (s *TaskService) Create(ctx context.Context, title string) (*Task, error) {
	if title == "" {
		return nil, ErrEmptyTitle
	}
	task := &Task{
		Title:     title,
		Status:    StatusTodo,
		CreatedAt: time.Now(),
	}
	if err := s.repo.Save(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *TaskService) GetByID(ctx context.Context, id int) (*Task, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *TaskService) List(ctx context.Context) ([]*Task, error) {
	return s.repo.FindAll(ctx)
}

func validStatus(status TaskStatus) bool {
	return status == StatusTodo || status == StatusDoing || status == StatusDone
}

func (s *TaskService) UpdateStatus(ctx context.Context, id int, status TaskStatus) error {
	if !validStatus(status) {
		return ErrInvalidStatus
	}
	task, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	task.Status = status
	return s.repo.Update(ctx, task)
}

func (s *TaskService) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}
