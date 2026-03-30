# Reliability и Fault Tolerance

## Circuit Breaker

```
Паттерн из электротехники: предохранитель

Состояния:
  CLOSED → запросы проходят нормально
    ↓ (N ошибок подряд)
  OPEN → запросы отклоняются сразу (fail fast)
    ↓ (через timeout)
  HALF-OPEN → пропускает 1 запрос для проверки
    ↓ успех → CLOSED
    ↓ ошибка → OPEN

Зачем:
  - Не долбить упавший сервис
  - Быстрый ответ клиенту (вместо timeout)
  - Дать сервису время восстановиться
  - Каскадные отказы: A → B → C, если C упал → без CB все ресурсы A заняты ожиданием
```

```go
// sony/gobreaker
import "github.com/sony/gobreaker/v2"

cb := gobreaker.NewCircuitBreaker[[]byte](gobreaker.Settings{
    Name:        "payment-api",
    MaxRequests: 3,                        // запросов в HALF-OPEN
    Interval:    10 * time.Second,         // период сброса счётчиков в CLOSED
    Timeout:     30 * time.Second,         // время в OPEN до HALF-OPEN
    ReadyToTrip: func(counts gobreaker.Counts) bool {
        return counts.ConsecutiveFailures > 5
    },
    OnStateChange: func(name string, from, to gobreaker.State) {
        log.Printf("circuit breaker %s: %s → %s", name, from, to)
    },
})

result, err := cb.Execute(func() ([]byte, error) {
    return callPaymentAPI()
})
if err != nil {
    // gobreaker.ErrOpenState — circuit breaker open
    // gobreaker.ErrTooManyRequests — half-open limit
}
```

## Retry с Backoff

```
Стратегии:
  1. Fixed delay: wait 1s, 1s, 1s
  2. Exponential backoff: 1s, 2s, 4s, 8s, 16s
  3. Exponential + jitter (рекомендуется):
     wait = min(cap, base * 2^attempt) + random(0, base)
     Jitter предотвращает thundering herd

Что НЕ ретраить:
  - 400 Bad Request (ошибка клиента)
  - 401/403 (auth проблема)
  - 404 (не найдено)

Что ретраить:
  - 429 Too Many Requests (с Retry-After header)
  - 500, 502, 503, 504 (серверные ошибки)
  - Timeout / connection refused
```

```go
func withRetry[T any](ctx context.Context, maxRetries int, fn func() (T, error)) (T, error) {
    var zero T
    for attempt := 0; attempt <= maxRetries; attempt++ {
        result, err := fn()
        if err == nil {
            return result, nil
        }
        if attempt == maxRetries {
            return zero, fmt.Errorf("after %d retries: %w", maxRetries, err)
        }
        // Exponential backoff + jitter
        backoff := time.Duration(1<<attempt) * 100 * time.Millisecond
        jitter := time.Duration(rand.Int64N(int64(backoff / 2)))
        select {
        case <-ctx.Done():
            return zero, ctx.Err()
        case <-time.After(backoff + jitter):
        }
    }
    return zero, errors.New("unreachable")
}
```

## Bulkhead (переборка)

```
Из кораблестроения: отсеки изолированы, пробоина в одном не топит весь корабль

Виды:
  1. Thread pool isolation:
     - Отдельный пул горутин для каждого downstream сервиса
     - Payment API: макс 20 горутин
     - Inventory API: макс 30 горутин
     - Один сервис тормозит → не влияет на другие

  2. Semaphore isolation:
     - Ограничение concurrent requests
     - Проще, без отдельного пула
```

```go
// Bulkhead через semaphore
type Bulkhead struct {
    sem chan struct{}
}

func NewBulkhead(maxConcurrent int) *Bulkhead {
    return &Bulkhead{sem: make(chan struct{}, maxConcurrent)}
}

func (b *Bulkhead) Execute(ctx context.Context, fn func() error) error {
    select {
    case b.sem <- struct{}{}:
        defer func() { <-b.sem }()
        return fn()
    case <-ctx.Done():
        return ctx.Err()
    default:
        return errors.New("bulkhead: rejected, too many concurrent requests")
    }
}

// Использование
paymentBulkhead := NewBulkhead(20)
inventoryBulkhead := NewBulkhead(30)
```

## Timeout

```
Правила:
  1. ВСЕГДА ставь timeout на внешние вызовы
  2. Timeout должен быть меньше, чем у вызывающего
     Client (10s) → API Gateway (8s) → Service A (5s) → DB (3s)
  3. Используй context для propagation
  4. Deadline > Timeout (deadline учитывает время в очереди)
```

```go
// Каскадные timeouts через context
func handleRequest(w http.ResponseWriter, r *http.Request) {
    // Общий deadline на запрос
    ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
    defer cancel()

    // Каждый downstream вызов наследует context
    user, err := userService.Get(ctx, userID)       // вложенный timeout
    orders, err := orderService.List(ctx, userID)    // тот же deadline
}
```

## Rate Limiting

```
Алгоритмы:

1. Token Bucket (чаще всего):
   - Bucket заполняется токенами с фиксированной скоростью
   - Запрос забирает 1 токен
   - Bucket пуст → reject (429)
   - Позволяет burst (размер bucket)

2. Leaky Bucket:
   - Запросы попадают в очередь фиксированного размера
   - Обрабатываются с постоянной скоростью
   - Очередь полна → reject
   - Сглаживает burst

3. Fixed Window:
   - Считаем запросы в окне (1 минута)
   - > limit → reject
   - Проблема: burst на границе окон (2x limit)

4. Sliding Window Log:
   - Храним timestamp каждого запроса
   - Считаем в скользящем окне
   - Точный, но много памяти

5. Sliding Window Counter (рекомендуется):
   - Комбинация fixed window + вес предыдущего окна
   - requests = prev_count * overlap% + current_count
   - Хороший баланс точность/память
```

```go
// Rate limiting с Redis (sliding window)
// KEYS[1] = rate:user:123
// ARGV[1] = window (60s), ARGV[2] = limit (100), ARGV[3] = now

local key = KEYS[1]
local window = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

-- Удалить старые записи
redis.call('ZREMRANGEBYSCORE', key, 0, now - window)
-- Подсчитать текущие
local count = redis.call('ZCARD', key)
if count >= limit then
    return 0 -- rejected
end
-- Добавить текущий запрос
redis.call('ZADD', key, now, now .. math.random())
redis.call('EXPIRE', key, window)
return 1 -- allowed
```

## Health Checks

```
Liveness: "сервис жив?" (перезапустить если нет)
  GET /healthz → 200 OK
  Что проверять: процесс работает, не в deadlock

Readiness: "сервис готов принимать трафик?" (убрать из LB если нет)
  GET /readyz → 200 OK или 503
  Что проверять: DB connected, cache warm, dependencies available

Startup: "сервис запустился?" (не проверять liveness пока не стартовал)
  Для сервисов с долгим стартом

Kubernetes:
  livenessProbe:
    httpGet: { path: /healthz, port: 8080 }
    periodSeconds: 10
    failureThreshold: 3     → после 3 неудач → restart pod
  readinessProbe:
    httpGet: { path: /readyz, port: 8080 }
    periodSeconds: 5
    failureThreshold: 1     → после 1 неудачи → убрать из Service
```

## Graceful Degradation

```
Стратегии когда сервис/зависимость деградирует:

1. Fallback response:
   - Recommendation service down → показать популярные товары
   - Profile image service down → показать default avatar

2. Cached data:
   - Если не можем получить свежие данные → вернуть из кэша
   - Stale данные лучше, чем ошибка

3. Feature flags:
   - Отключить тяжёлые фичи при высокой нагрузке
   - Отключить real-time analytics → batch

4. Load shedding:
   - Отбрасывать часть запросов при перегрузке
   - Приоритет: paid users > free users
   - Приоритет: read > write (при проблемах с БД)
```

## SLA, SLO, SLI

```
SLI (Service Level Indicator):
  Метрика: latency p99, error rate, availability %
  Пример: 99.5% запросов < 200ms

SLO (Service Level Objective):
  Цель: SLI должен быть >= X
  Пример: availability >= 99.9% за месяц

SLA (Service Level Agreement):
  Контракт с клиентом: если SLO нарушен → компенсация
  Обычно SLA слабее SLO (внутренняя цель жёстче)

Availability targets:
  99%    → 7.2 часа downtime/месяц
  99.9%  → 43 минуты/месяц
  99.95% → 22 минуты/месяц
  99.99% → 4.3 минуты/месяц

Error budget:
  SLO = 99.9% → бюджет = 0.1% ошибок
  Бюджет израсходован → замораживаем фичи, фокус на reliability
```

## Частые вопросы

**Q: Зачем circuit breaker если есть timeout?**
A: Timeout ждёт каждый раз. Circuit breaker после N ошибок сразу отклоняет запросы (fail fast), не тратя ресурсы. Плюс даёт upstream время восстановиться.

**Q: Retry storm — что это?**
A: Сервис A ретраит 3 раза → B ретраит 3 раза → C. Итого 9 запросов вместо 1. Решение: retry budget (не больше 10% дополнительных запросов), exponential backoff + jitter.

**Q: Как реализовать distributed rate limiting?**
A: Redis + Lua script (атомарные операции). Sliding window counter — лучший баланс. Token bucket — для простых случаев. Для API gateway — встроенные решения (Kong, Envoy).
