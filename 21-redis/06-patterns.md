# Паттерны использования

## Pipeline

Отправка нескольких команд за один round-trip. Не атомарно, но быстро.

```go
pipe := rdb.Pipeline()

incr := pipe.Incr(ctx, "counter")
expire := pipe.Expire(ctx, "counter", time.Hour)
get := pipe.Get(ctx, "user:1")

_, err := pipe.Exec(ctx) // One round-trip for all commands
if err != nil && !errors.Is(err, redis.Nil) {
    return err
}

fmt.Println(incr.Val())   // Result of INCR
fmt.Println(get.Val())    // Result of GET
```

**Pipeline vs один запрос**: 100 команд через pipeline ≈ 1 round-trip вместо 100.

## Транзакции (MULTI/EXEC)

Атомарное выполнение группы команд. Никто не вклинится между ними.

```go
// TxPipeline = MULTI + commands + EXEC
pipe := rdb.TxPipeline()

pipe.Set(ctx, "balance:from", 900, 0)
pipe.Set(ctx, "balance:to", 1100, 0)

_, err := pipe.Exec(ctx)
```

### Optimistic Locking (WATCH)

```go
// Transfer money with WATCH — retry if key changed
func Transfer(ctx context.Context, rdb *redis.Client, from, to string, amount int64) error {
    txf := func(tx *redis.Tx) error {
        // Read current balances (inside WATCH)
        fromBal, err := tx.Get(ctx, from).Int64()
        if err != nil {
            return err
        }
        toBal, err := tx.Get(ctx, to).Int64()
        if err != nil {
            return err
        }

        if fromBal < amount {
            return fmt.Errorf("insufficient funds")
        }

        // Execute atomically — fails if watched keys changed
        _, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
            pipe.Set(ctx, from, fromBal-amount, 0)
            pipe.Set(ctx, to, toBal+amount, 0)
            return nil
        })
        return err
    }

    // Retry loop — WATCH may fail under contention
    for i := 0; i < 100; i++ {
        err := rdb.Watch(ctx, txf, from, to)
        if err == nil {
            return nil
        }
        if errors.Is(err, redis.TxFailedErr) {
            continue // Key changed — retry
        }
        return err
    }

    return fmt.Errorf("transaction failed after retries")
}
```

## Lua-скрипты

Атомарное выполнение сложной логики прямо в Redis.

```go
// Rate limiter: increment counter and set expiry atomically
var rateLimitScript = redis.NewScript(`
    local current = redis.call("INCR", KEYS[1])
    if current == 1 then
        redis.call("EXPIRE", KEYS[1], ARGV[1])
    end
    return current
`)

func CheckRateLimit(ctx context.Context, rdb *redis.Client, key string, limit int64, window time.Duration) (bool, error) {
    count, err := rateLimitScript.Run(ctx, rdb, []string{key}, int(window.Seconds())).Int64()
    if err != nil {
        return false, err
    }
    return count <= limit, nil
}

// Usage
allowed, _ := CheckRateLimit(ctx, rdb, "rate:user:123", 100, time.Minute)
```

### Почему Lua а не pipeline?

Pipeline не атомарен — между командами могут выполниться чужие. Lua-скрипт выполняется как одна атомарная операция, блокируя Redis на время выполнения.

> **Правило**: Lua-скрипты должны быть **быстрыми**. Длинный скрипт блокирует весь Redis.

## Rate Limiter (Sliding Window)

Точный rate limiter на Sorted Sets.

```go
func SlidingWindowRateLimit(ctx context.Context, rdb *redis.Client, key string, limit int64, window time.Duration) (bool, error) {
    now := time.Now().UnixMicro()
    windowStart := now - window.Microseconds()

    pipe := rdb.Pipeline()

    // Remove expired entries
    pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))

    // Count current entries
    count := pipe.ZCard(ctx, key)

    // Add current request
    pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})

    // Set TTL on the key itself
    pipe.Expire(ctx, key, window)

    _, err := pipe.Exec(ctx)
    if err != nil {
        return false, err
    }

    return count.Val() < limit, nil
}

// Middleware
func RateLimitMiddleware(rdb *redis.Client, limit int64, window time.Duration) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ip := r.RemoteAddr
            key := fmt.Sprintf("rate:%s", ip)

            allowed, err := SlidingWindowRateLimit(r.Context(), rdb, key, limit, window)
            if err != nil || !allowed {
                http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

## Session Storage

```go
type SessionStore struct {
    rdb *redis.Client
    ttl time.Duration
}

type Session struct {
    UserID    string            `json:"user_id"`
    Role      string            `json:"role"`
    Data      map[string]string `json:"data"`
    CreatedAt time.Time         `json:"created_at"`
}

func (s *SessionStore) Create(ctx context.Context, session *Session) (string, error) {
    id := uuid.New().String()
    session.CreatedAt = time.Now()

    data, err := json.Marshal(session)
    if err != nil {
        return "", err
    }

    key := "session:" + id
    if err := s.rdb.Set(ctx, key, data, s.ttl).Err(); err != nil {
        return "", err
    }

    return id, nil
}

func (s *SessionStore) Get(ctx context.Context, id string) (*Session, error) {
    key := "session:" + id
    data, err := s.rdb.Get(ctx, key).Bytes()
    if errors.Is(err, redis.Nil) {
        return nil, fmt.Errorf("session not found")
    }
    if err != nil {
        return nil, err
    }

    // Sliding expiry — refresh on access
    s.rdb.Expire(ctx, key, s.ttl)

    var session Session
    if err := json.Unmarshal(data, &session); err != nil {
        return nil, err
    }
    return &session, nil
}

func (s *SessionStore) Destroy(ctx context.Context, id string) error {
    return s.rdb.Del(ctx, "session:"+id).Err()
}
```

## Job Queue (Lists)

Простая очередь задач на BRPOP.

```go
type Job struct {
    ID      string `json:"id"`
    Type    string `json:"type"`
    Payload []byte `json:"payload"`
}

// Producer
func Enqueue(ctx context.Context, rdb *redis.Client, queue string, job *Job) error {
    job.ID = uuid.New().String()
    data, _ := json.Marshal(job)
    return rdb.LPush(ctx, queue, data).Err()
}

// Consumer (blocking)
func Dequeue(ctx context.Context, rdb *redis.Client, queue string, timeout time.Duration) (*Job, error) {
    result, err := rdb.BRPop(ctx, timeout, queue).Result()
    if errors.Is(err, redis.Nil) {
        return nil, nil // Timeout
    }
    if err != nil {
        return nil, err
    }

    var job Job
    if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
        return nil, err
    }
    return &job, nil
}

// Reliable queue — RPOPLPUSH for at-least-once
func DequeueReliable(ctx context.Context, rdb *redis.Client, queue, processing string) (*Job, error) {
    // Atomically pop from queue and push to processing list
    data, err := rdb.RPopLPush(ctx, queue, processing).Result()
    if errors.Is(err, redis.Nil) {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }

    var job Job
    json.Unmarshal([]byte(data), &job)
    return &job, nil
}

// After successful processing — remove from processing list
func Ack(ctx context.Context, rdb *redis.Client, processing string, job *Job) error {
    data, _ := json.Marshal(job)
    return rdb.LRem(ctx, processing, 1, data).Err()
}
```

## Idempotency Keys

Защита от дублирования запросов.

```go
func EnsureIdempotent(ctx context.Context, rdb *redis.Client, key string, ttl time.Duration, fn func() (any, error)) (any, error) {
    idempKey := "idempotent:" + key

    // Check if already processed
    cached, err := rdb.Get(ctx, idempKey).Bytes()
    if err == nil {
        var result any
        json.Unmarshal(cached, &result)
        return result, nil
    }

    // Try to acquire
    ok, _ := rdb.SetNX(ctx, idempKey+":lock", "1", 30*time.Second).Result()
    if !ok {
        return nil, fmt.Errorf("request is being processed")
    }
    defer rdb.Del(ctx, idempKey+":lock")

    // Execute
    result, err := fn()
    if err != nil {
        return nil, err
    }

    // Store result
    data, _ := json.Marshal(result)
    rdb.Set(ctx, idempKey, data, ttl)

    return result, nil
}
```

## Частые вопросы

1. **Pipeline vs Transaction?** — Pipeline — батч команд за один round-trip, но не атомарный. Transaction (`MULTI/EXEC`) — атомарный, другие клиенты не вклинятся.

2. **Когда Lua, а когда WATCH?** — Lua проще и надёжнее для «read-modify-write». WATCH — когда логика сложная и не хочется писать Lua.

3. **BRPOP vs Streams для очередей?** — `BRPOP` проще, но один consumer на сообщение. Streams поддерживают consumer groups и replay.

4. **Как гарантировать at-least-once в очереди?** — `RPOPLPUSH` в processing list. При падении — recovery из processing list.
