package rate_limiter

import (
	"context"
	"sync"
	"time"
)

type RateLimiter struct {
	mu     sync.Mutex
	rate   float64
	burst  int
	tokens float64
	last   time.Time
}

func NewRateLimiter(rate float64, burst int) *RateLimiter {
	return &RateLimiter{
		rate:   rate,
		burst:  burst,
		tokens: float64(burst),
		last:   time.Now(),
	}
}

func (rl *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(rl.last).Seconds()
	rl.tokens += elapsed * rl.rate
	if rl.tokens > float64(rl.burst) {
		rl.tokens = float64(rl.burst)
	}
	rl.last = now
}

func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.refill()
	if rl.tokens >= 1 {
		rl.tokens--
		return true
	}
	return false
}

func (rl *RateLimiter) Wait(ctx context.Context) error {
	for {
		if rl.Allow() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Millisecond * 5):
			// retry
		}
	}
}
