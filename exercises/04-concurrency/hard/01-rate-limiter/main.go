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

// TODO: пополни токены на основе прошедшего времени, попробуй взять один
func (rl *RateLimiter) Allow() bool {
	return false
}

// TODO: жди пока токен не станет доступен или ctx отменён
func (rl *RateLimiter) Wait(ctx context.Context) error {
	return nil
}
