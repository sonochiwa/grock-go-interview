# Распределённые блокировки

## Зачем

Когда несколько инстансов сервиса должны гарантировать, что критическую секцию выполняет только один.

```
Instance A ──┐
Instance B ──┼── Redis Lock ──▶ [критическая секция]
Instance C ──┘     (только один проходит)
```

Примеры: обработка платежа, отправка email, scheduled job.

## Базовая блокировка через SETNX

```go
func AcquireLock(ctx context.Context, rdb *redis.Client, key string, ttl time.Duration) (string, bool) {
    // Unique value — чтобы только владелец мог разблокировать
    value := uuid.New().String()

    ok, err := rdb.SetNX(ctx, key, value, ttl).Result()
    if err != nil || !ok {
        return "", false
    }

    return value, true
}

// IMPORTANT: release only if we still own the lock (Lua for atomicity)
var releaseLockScript = redis.NewScript(`
    if redis.call("GET", KEYS[1]) == ARGV[1] then
        return redis.call("DEL", KEYS[1])
    end
    return 0
`)

func ReleaseLock(ctx context.Context, rdb *redis.Client, key, value string) error {
    _, err := releaseLockScript.Run(ctx, rdb, []string{key}, value).Result()
    return err
}
```

### Почему Lua-скрипт для release?

Без Lua между `GET` и `DEL` может произойти:
1. Горутина A: `GET lock` → видит свой value
2. TTL истекает, горутина B получает lock
3. Горутина A: `DEL lock` → удаляет чужой lock!

Lua-скрипт выполняется атомарно в Redis.

## Полная реализация с retry

```go
type RedisLock struct {
    rdb    *redis.Client
    key    string
    value  string
    ttl    time.Duration
}

func NewRedisLock(rdb *redis.Client, key string, ttl time.Duration) *RedisLock {
    return &RedisLock{
        rdb:   rdb,
        key:   "lock:" + key,
        value: uuid.New().String(),
        ttl:   ttl,
    }
}

func (l *RedisLock) Acquire(ctx context.Context) error {
    for {
        ok, err := l.rdb.SetNX(ctx, l.key, l.value, l.ttl).Result()
        if err != nil {
            return fmt.Errorf("redis error: %w", err)
        }
        if ok {
            return nil
        }

        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(50 * time.Millisecond):
            // Retry
        }
    }
}

func (l *RedisLock) Release(ctx context.Context) error {
    _, err := releaseLockScript.Run(ctx, l.rdb, []string{l.key}, l.value).Result()
    return err
}

// Usage
func ProcessPayment(ctx context.Context, rdb *redis.Client, orderID string) error {
    lock := NewRedisLock(rdb, "payment:"+orderID, 30*time.Second)

    if err := lock.Acquire(ctx); err != nil {
        return fmt.Errorf("failed to acquire lock: %w", err)
    }
    defer lock.Release(ctx)

    // Critical section — only one instance executes this
    return doProcessPayment(ctx, orderID)
}
```

## Lock Renewal (Watchdog)

Если операция длится дольше TTL — lock истекает и другой инстанс может войти. Решение: фоновое продление.

```go
func (l *RedisLock) AcquireWithRenewal(ctx context.Context) (context.Context, context.CancelFunc, error) {
    if err := l.Acquire(ctx); err != nil {
        return nil, nil, err
    }

    ctx, cancel := context.WithCancel(ctx)

    // Background renewal
    go func() {
        ticker := time.NewTicker(l.ttl / 3) // Renew at 1/3 of TTL
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                extended, _ := l.rdb.Expire(ctx, l.key, l.ttl).Result()
                if !extended {
                    cancel() // Lost the lock
                    return
                }
            }
        }
    }()

    return ctx, func() {
        cancel()
        l.Release(context.Background())
    }, nil
}
```

## Redlock (распределённый)

Алгоритм от Salvatore Sanfilippo для кластера из N **независимых** Redis-нод (не реплик).

```
1. Получить текущее время
2. Попытаться взять lock на N/2+1 нодах (majority) с коротким timeout
3. Если majority получено И прошло меньше TTL — lock получен
4. Если нет — разблокировать все ноды
```

### redsync — готовая реализация

```go
import (
    "github.com/go-redsync/redsync/v4"
    "github.com/go-redsync/redsync/v4/redis/goredis/v9"
)

func main() {
    pool := goredis.NewPool(rdb)
    rs := redsync.New(pool)

    mutex := rs.NewMutex("my-lock",
        redsync.WithExpiry(10*time.Second),
        redsync.WithTries(32),
        redsync.WithRetryDelay(100*time.Millisecond),
    )

    if err := mutex.Lock(); err != nil {
        log.Fatal(err)
    }
    defer mutex.Unlock()

    // Critical section
}
```

## Сравнение подходов

| | Redis (SETNX) | Redis (Redlock) | etcd (lease) | ZooKeeper |
|---|---|---|---|---|
| **Сложность** | Простой | Средний | Средний | Сложный |
| **Надёжность** | Один Redis — SPOF | Majority из N нод | Raft consensus | ZAB consensus |
| **Latency** | ~1 ms | ~N ms | ~5-10 ms | ~5-10 ms |
| **Когда** | Некритичные locks | Важные distributed locks | Уже есть etcd (k8s) | Legacy |
| **Fencing token** | Нет из коробки | Нет из коробки | Lease revision | Sequential znode |

> **Важно**: Redlock [критикуется](https://martin.kleppmann.com/2016/02/08/how-to-do-distributed-locking.html) Martin Kleppmann — для **по-настоящему** критичных операций (деньги) лучше fencing token + БД constraint.

## Частые вопросы

1. **Зачем уникальный value в lock?** — Чтобы владелец мог безопасно освободить только свой lock, не чужой.

2. **Что если Redis упал пока lock держится?** — При одном Redis — lock теряется. Redlock решает majority-подходом.

3. **SETNX vs SET NX?** — `SETNX` — старая команда. `SET key value NX EX ttl` — современная, делает всё атомарно.

4. **Можно ли реализовать fair lock (очередь)?** — Да, через Sorted Set с timestamp. Но обычно проще использовать retry с jitter.
