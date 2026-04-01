package health_check

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAllHealthy(t *testing.T) {
	hc := NewHealthChecker()
	hc.AddCheck("db", func(ctx context.Context) error { return nil })
	hc.AddCheck("redis", func(ctx context.Context) error { return nil })

	rec := httptest.NewRecorder()
	hc.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var resp HealthResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Status != "healthy" {
		t.Errorf("status = %q, want healthy", resp.Status)
	}
	if resp.Checks["db"].Status != "up" {
		t.Errorf("db status = %q, want up", resp.Checks["db"].Status)
	}
}

func TestOneFailing(t *testing.T) {
	hc := NewHealthChecker()
	hc.AddCheck("db", func(ctx context.Context) error { return nil })
	hc.AddCheck("redis", func(ctx context.Context) error { return errors.New("connection refused") })

	rec := httptest.NewRecorder()
	hc.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/healthz", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}

	var resp HealthResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Status != "unhealthy" {
		t.Errorf("status = %q, want unhealthy", resp.Status)
	}
	if resp.Checks["redis"].Status != "down" {
		t.Errorf("redis status = %q, want down", resp.Checks["redis"].Status)
	}
	if resp.Checks["redis"].Error == "" {
		t.Error("redis should have error message")
	}
}

func TestLatencyTracked(t *testing.T) {
	hc := NewHealthChecker()
	hc.AddCheck("slow", func(ctx context.Context) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	rec := httptest.NewRecorder()
	hc.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/healthz", nil))

	var resp HealthResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Checks["slow"].Latency == "" {
		t.Error("latency should be tracked")
	}
}

func TestNoChecks(t *testing.T) {
	hc := NewHealthChecker()
	rec := httptest.NewRecorder()
	hc.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("no checks: status = %d, want 200", rec.Code)
	}
}
