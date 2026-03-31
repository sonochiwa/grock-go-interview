package clean_layers

import (
	"context"
	"errors"
	"testing"
)

func TestCreateTask(t *testing.T) {
	repo := NewInMemoryRepo()
	svc := NewTaskService(repo)
	ctx := context.Background()

	task, err := svc.Create(ctx, "Write tests")
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if task.Title != "Write tests" {
		t.Errorf("Title = %q, want 'Write tests'", task.Title)
	}
	if task.Status != StatusTodo {
		t.Errorf("Status = %q, want todo", task.Status)
	}
	if task.ID == 0 {
		t.Error("ID should not be 0")
	}
}

func TestCreateEmptyTitle(t *testing.T) {
	svc := NewTaskService(NewInMemoryRepo())
	_, err := svc.Create(context.Background(), "")
	if !errors.Is(err, ErrEmptyTitle) {
		t.Errorf("expected ErrEmptyTitle, got %v", err)
	}
}

func TestGetByID(t *testing.T) {
	repo := NewInMemoryRepo()
	svc := NewTaskService(repo)
	ctx := context.Background()

	created, _ := svc.Create(ctx, "Test task")
	got, err := svc.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID error: %v", err)
	}
	if got.Title != "Test task" {
		t.Errorf("Title = %q", got.Title)
	}
}

func TestGetByIDNotFound(t *testing.T) {
	svc := NewTaskService(NewInMemoryRepo())
	_, err := svc.GetByID(context.Background(), 999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateStatus(t *testing.T) {
	repo := NewInMemoryRepo()
	svc := NewTaskService(repo)
	ctx := context.Background()

	task, _ := svc.Create(ctx, "Task")
	err := svc.UpdateStatus(ctx, task.ID, StatusDoing)
	if err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}

	updated, _ := svc.GetByID(ctx, task.ID)
	if updated.Status != StatusDoing {
		t.Errorf("Status = %q, want doing", updated.Status)
	}
}

func TestUpdateStatusInvalid(t *testing.T) {
	repo := NewInMemoryRepo()
	svc := NewTaskService(repo)
	ctx := context.Background()

	task, _ := svc.Create(ctx, "Task")
	err := svc.UpdateStatus(ctx, task.ID, "invalid")
	if !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}
}

func TestList(t *testing.T) {
	repo := NewInMemoryRepo()
	svc := NewTaskService(repo)
	ctx := context.Background()

	svc.Create(ctx, "A")
	svc.Create(ctx, "B")

	tasks, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("List len = %d, want 2", len(tasks))
	}
}

func TestDelete(t *testing.T) {
	repo := NewInMemoryRepo()
	svc := NewTaskService(repo)
	ctx := context.Background()

	task, _ := svc.Create(ctx, "To delete")
	err := svc.Delete(ctx, task.ID)
	if err != nil {
		t.Fatalf("Delete error: %v", err)
	}

	_, err = svc.GetByID(ctx, task.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Error("expected ErrNotFound after delete")
	}
}
