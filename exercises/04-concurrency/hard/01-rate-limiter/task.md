# Rate Limiter

Реализуй token bucket rate limiter:

- `NewRateLimiter(rate float64, burst int)` — rate = tokens/sec, burst = max tokens
- `Allow() bool` — true если можно пропустить запрос (есть токен)
- `Wait(ctx context.Context) error` — блокирует до получения токена или отмены ctx

Goroutine-safe!
