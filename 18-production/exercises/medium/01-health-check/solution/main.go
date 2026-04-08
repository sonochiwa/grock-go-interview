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

func (hc *HealthChecker) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hc.mu.RLock()
		checks := make(map[string]func(ctx context.Context) error, len(hc.checks))
		for k, v := range hc.checks {
			checks[k] = v
		}
		hc.mu.RUnlock()

		results := make(map[string]CheckResult, len(checks))
		var mu sync.Mutex
		var wg sync.WaitGroup

		healthy := true
		for name, check := range checks {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
				defer cancel()

				start := time.Now()
				err := check(ctx)
				latency := time.Since(start)

				result := CheckResult{
					Status:  "up",
					Latency: latency.Round(time.Millisecond).String(),
				}
				if err != nil {
					result.Status = "down"
					result.Error = err.Error()
					mu.Lock()
					healthy = false
					mu.Unlock()
				}

				mu.Lock()
				results[name] = result
				mu.Unlock()
			}()
		}
		wg.Wait()

		resp := HealthResponse{
			Status: "healthy",
			Checks: results,
		}
		if !healthy {
			resp.Status = "unhealthy"
		}

		w.Header().Set("Content-Type", "application/json")
		if !healthy {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		json.NewEncoder(w).Encode(resp)
	}
}
