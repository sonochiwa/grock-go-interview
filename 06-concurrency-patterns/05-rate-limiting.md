# Rate Limiting

## Обзор

Ограничение скорости операций. Защита внешних API от перегрузки, соблюдение rate limits.

## Концепции

### Простой: time.Ticker

```go
// 10 запросов в секунду
limiter := time.NewTicker(100 * time.Millisecond)
defer limiter.Stop()

for _, url := range urls {
    <-limiter.C // ждём тик
    go fetch(url)
}
```

### Token Bucket: x/time/rate

```go
import "golang.org/x/time/rate"

// 10 запросов/сек, burst до 30
limiter := rate.NewLimiter(rate.Limit(10), 30)

for _, url := range urls {
    // Wait блокирует до получения токена
    if err := limiter.Wait(ctx); err != nil {
        return err // context cancelled
    }
    go fetch(url)
}

// Неблокирующая проверка
if limiter.Allow() {
    process() // токен доступен
} else {
    drop() // превышен лимит
}

// Резервация (для задач, которые начнутся позже)
reservation := limiter.Reserve()
delay := reservation.Delay()
time.Sleep(delay)
process()
```

### Rate limiting в HTTP middleware

```go
func RateLimitMiddleware(rps float64, burst int) func(http.Handler) http.Handler {
    limiter := rate.NewLimiter(rate.Limit(rps), burst)
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if !limiter.Allow() {
                http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

// Per-IP rate limiting
type IPLimiter struct {
    mu       sync.Mutex
    limiters map[string]*rate.Limiter
}

func (l *IPLimiter) GetLimiter(ip string) *rate.Limiter {
    l.mu.Lock()
    defer l.mu.Unlock()
    if lim, ok := l.limiters[ip]; ok {
        return lim
    }
    lim := rate.NewLimiter(10, 30) // 10 rps per IP
    l.limiters[ip] = lim
    return lim
}
```

## Частые вопросы на собеседованиях

**Q: Что такое token bucket?**
A: Алгоритм: токены добавляются с фиксированной скоростью (rate), burst — максимум токенов. Запрос забирает токен. Нет токенов — ждём или отклоняем.

**Q: Как сделать per-user rate limiting?**
A: map[userID]*rate.Limiter с мьютексом. Или sync.Map для конкурентного доступа.
