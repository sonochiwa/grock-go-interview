package health_check

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type CheckResult struct {
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
	Latency string `json:"latency"`
}

type HealthResponse struct {
	Status string                 `json:"status"`
	Checks map[string]CheckResult `json:"checks"`
}

type HealthChecker struct {
	mu     sync.RWMutex
	checks map[string]func(ctx context.Context) error
}

func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		checks: make(map[string]func(ctx context.Context) error),
	}
}

func (hc *HealthChecker) AddCheck(name string, check func(ctx context.Context) error) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.checks[name] = check
}

// TODO: реализуй Handler
// 1. Выполни все checks параллельно (каждый с timeout 5s)
// 2. Собери результаты в HealthResponse
// 3. 200 если все "up", 503 если хоть один "down"
// 4. Запиши JSON в response
func (hc *HealthChecker) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w)
		_ = time.Now()
		_ = context.Background()
		w.WriteHeader(http.StatusNotImplemented)
	}
}
