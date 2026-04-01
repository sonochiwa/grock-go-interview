# Health Check

Реализуй систему health checks:

- `NewHealthChecker() *HealthChecker`
- `AddCheck(name string, check func(ctx context.Context) error)` — добавить проверку
- `Handler() http.HandlerFunc` — HTTP handler для `/healthz`

Ответ JSON:
```json
{"status": "healthy", "checks": {"db": {"status": "up", "latency": "2ms"}, "redis": {"status": "down", "error": "connection refused", "latency": "5ms"}}}
```

200 если все OK, 503 если хоть один failed. Каждый check с timeout 5 секунд.
