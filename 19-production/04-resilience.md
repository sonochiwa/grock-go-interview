# Resilience Patterns

## Комбинация паттернов

```
Типичный production setup для вызова external service:

  [Request]
    → Rate Limiter (не перегрузить)
      → Circuit Breaker (fail fast если сервис down)
        → Timeout (не ждать вечно)
          → Retry с backoff (transient errors)
            → Bulkhead (изоляция ресурсов)
              → [External Service]
```

```go
// Комбинация: circuit breaker + retry + timeout
func (c *PaymentClient) Charge(ctx context.Context, req *ChargeRequest) (*ChargeResponse, error) {
    // Timeout
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    // Circuit breaker
    result, err := c.breaker.Execute(func() (any, error) {
        // Retry
        return withRetry(ctx, 3, func() (*ChargeResponse, error) {
            return c.doCharge(ctx, req)
        })
    })
    if err != nil {
        return nil, fmt.Errorf("payment charge: %w", err)
    }
    return result.(*ChargeResponse), nil
}
```

## Health Checks (подробнее)

```go
type HealthChecker struct {
    checks map[string]func(ctx context.Context) error
}

func NewHealthChecker() *HealthChecker {
    return &HealthChecker{checks: make(map[string]func(ctx context.Context) error)}
}

func (h *HealthChecker) Register(name string, check func(ctx context.Context) error) {
    h.checks[name] = check
}

// Liveness: сервис жив?
func (h *HealthChecker) LivenessHandler() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        fmt.Fprint(w, "ok")
    }
}

// Readiness: сервис готов принимать трафик?
func (h *HealthChecker) ReadinessHandler() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
        defer cancel()

        results := make(map[string]string)
        healthy := true

        for name, check := range h.checks {
            if err := check(ctx); err != nil {
                results[name] = err.Error()
                healthy = false
            } else {
                results[name] = "ok"
            }
        }

        status := http.StatusOK
        if !healthy {
            status = http.StatusServiceUnavailable
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(status)
        json.NewEncoder(w).Encode(results)
    }
}

// Регистрация проверок
hc := NewHealthChecker()
hc.Register("postgres", func(ctx context.Context) error {
    return db.PingContext(ctx)
})
hc.Register("redis", func(ctx context.Context) error {
    return redisClient.Ping(ctx).Err()
})
hc.Register("kafka", func(ctx context.Context) error {
    _, err := kafkaClient.Brokers()
    return err
})

http.HandleFunc("/healthz", hc.LivenessHandler())
http.HandleFunc("/readyz", hc.ReadinessHandler())
```

## Connection Pool Monitoring

```go
// Database
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(10)
db.SetConnMaxLifetime(5 * time.Minute)
db.SetConnMaxIdleTime(1 * time.Minute)

// Метрики
go func() {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    for range ticker.C {
        stats := db.Stats()
        dbOpenConns.Set(float64(stats.OpenConnections))
        dbIdleConns.Set(float64(stats.Idle))
        dbInUseConns.Set(float64(stats.InUse))
        dbWaitCount.Add(float64(stats.WaitCount))
        dbWaitDuration.Observe(stats.WaitDuration.Seconds())
    }
}()
```

## Panic Recovery в HTTP

```go
func recoveryMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                slog.Error("panic recovered",
                    "err", err,
                    "stack", string(debug.Stack()),
                    "method", r.Method,
                    "path", r.URL.Path,
                )
                w.WriteHeader(http.StatusInternalServerError)
                json.NewEncoder(w).Encode(map[string]string{
                    "error": "internal server error",
                })
            }
        }()
        next.ServeHTTP(w, r)
    })
}
```
