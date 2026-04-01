package rate_limiter

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestAllowBurst(t *testing.T) {
	rl := NewRateLimiter(10, 5) // 10/sec, burst 5
	allowed := 0
	for range 10 {
		if rl.Allow() {
			allowed++
		}
	}
	if allowed != 5 {
		t.Errorf("expected 5 allowed (burst), got %d", allowed)
	}
}

func TestAllowRefill(t *testing.T) {
	rl := NewRateLimiter(100, 1)
	rl.Allow() // consume the burst
	if rl.Allow() {
		t.Error("should be denied immediately")
	}
	time.Sleep(15 * time.Millisecond)
	if !rl.Allow() {
		t.Error("should be allowed after refill")
	}
}

func TestWaitContextCancel(t *testing.T) {
	rl := NewRateLimiter(1, 1)
	rl.Allow() // consume

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := rl.Wait(ctx)
	if err == nil {
		t.Error("expected context error")
	}
}

func TestWaitSuccess(t *testing.T) {
	rl := NewRateLimiter(100, 1)
	rl.Allow()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := rl.Wait(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConcurrentAccess(t *testing.T) {
	rl := NewRateLimiter(1000, 10)
	var allowed atomic.Int64

	done := make(chan struct{})
	for range 50 {
		go func() {
			defer func() { done <- struct{}{} }()
			if rl.Allow() {
				allowed.Add(1)
			}
		}()
	}
	for range 50 {
		<-done
	}
	if a := allowed.Load(); a > 10 {
		t.Errorf("allowed %d > burst 10", a)
	}
}
