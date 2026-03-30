package worker_pool

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestBasic(t *testing.T) {
	p := NewPool[int, int](2, func(n int) (int, error) {
		return n * n, nil
	})
	defer p.Close()

	ch := p.Submit(5)
	res := <-ch
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}
	if res.Value != 25 {
		t.Errorf("got %d, want 25", res.Value)
	}
}

func TestError(t *testing.T) {
	errBad := errors.New("bad")
	p := NewPool[int, int](2, func(n int) (int, error) {
		if n < 0 {
			return 0, errBad
		}
		return n, nil
	})
	defer p.Close()

	ch := p.Submit(-1)
	res := <-ch
	if res.Err == nil {
		t.Fatal("expected error")
	}
}

func TestConcurrent(t *testing.T) {
	p := NewPool[int, int](4, func(n int) (int, error) {
		time.Sleep(10 * time.Millisecond)
		return n * 2, nil
	})
	defer p.Close()

	const n = 20
	channels := make([]<-chan Result[int], n)
	for i := range n {
		channels[i] = p.Submit(i)
	}

	for i, ch := range channels {
		res := <-ch
		if res.Err != nil {
			t.Errorf("task %d: unexpected error: %v", i, res.Err)
		}
		if res.Value != i*2 {
			t.Errorf("task %d: got %d, want %d", i, res.Value, i*2)
		}
	}
}

func TestClose(t *testing.T) {
	var mu sync.Mutex
	var count int
	p := NewPool[int, int](2, func(n int) (int, error) {
		time.Sleep(50 * time.Millisecond)
		mu.Lock()
		count++
		mu.Unlock()
		return n, nil
	})

	for range 5 {
		p.Submit(1)
	}
	p.Close()

	mu.Lock()
	defer mu.Unlock()
	if count < 5 {
		t.Errorf("only %d tasks completed before Close returned", count)
	}
}
