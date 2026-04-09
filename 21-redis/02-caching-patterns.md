# Паттерны кэширования

## Стратегии кэширования

```
┌─────────────────────────────────────────────────────┐
│                   Стратегии чтения                   │
├─────────────────┬───────────────────────────────────┤
│  Cache-Aside    │  App читает кэш → miss → БД →    │
│  (Lazy Loading) │  записывает в кэш                 │
├─────────────────┼───────────────────────────────────┤
│  Read-Through   │  Кэш сам ходит в БД при miss     │
│                 │  (app не знает про БД)            │
├─────────────────┴───────────────────────────────────┤
│                   Стратегии записи                   │
├─────────────────┬───────────────────────────────────┤
│  Write-Through  │  Запись в кэш + БД синхронно      │
├─────────────────┼───────────────────────────────────┤
│  Write-Behind   │  Запись в кэш, БД — асинхронно    │
│  (Write-Back)   │  (батчами/с задержкой)            │
├─────────────────┼───────────────────────────────────┤
│  Write-Around   │  Запись только в БД, кэш не       │
│                 │  обновляется (ждёт miss)           │
└─────────────────┴───────────────────────────────────┘
```

## Cache-Aside (самый частый)

Приложение управляет кэшем явно: сначала проверяет кэш, при промахе идёт в БД и кладёт результат в кэш.

```go
func (s *UserService) GetUser(ctx context.Context, id int64) (*User, error) {
    key := fmt.Sprintf("user:%d", id)

    // 1. Try cache
    cached, err := s.rdb.Get(ctx, key).Bytes()
    if err == nil {
        var user User
        if err := json.Unmarshal(cached, &user); err == nil {
            return &user, nil
        }
    }

    // 2. Cache miss — go to DB
    user, err := s.db.GetUser(ctx, id)
    if err != nil {
        return nil, err
    }

    // 3. Populate cache
    data, _ := json.Marshal(user)
    s.rdb.Set(ctx, key, data, 15*time.Minute)

    return user, nil
}
```

**Плюсы**: простой, кэшируются только запрошенные данные.
**Минусы**: первый запрос всегда медленный (cold start), данные могут устареть.

## Write-Through

Запись всегда идёт и в кэш, и в БД. Кэш всегда актуален.

```go
func (s *UserService) UpdateUser(ctx context.Context, user *User) error {
    // 1. Write to DB
    if err := s.db.UpdateUser(ctx, user); err != nil {
        return err
    }

    // 2. Write to cache (synchronously)
    key := fmt.Sprintf("user:%d", user.ID)
    data, _ := json.Marshal(user)
    return s.rdb.Set(ctx, key, data, 15*time.Minute).Err()
}
```

**Плюсы**: кэш всегда свежий.
**Минусы**: каждая запись медленнее (два хопа), кэшируются данные которые могут не читаться.

## Write-Behind (Write-Back)

Запись только в кэш, а в БД — асинхронно (батчами). Используется для write-heavy нагрузок.

```go
func (s *UserService) UpdateUser(ctx context.Context, user *User) error {
    key := fmt.Sprintf("user:%d", user.ID)
    data, _ := json.Marshal(user)

    // 1. Write to cache only
    if err := s.rdb.Set(ctx, key, data, 15*time.Minute).Err(); err != nil {
        return err
    }

    // 2. Queue for async DB write
    s.rdb.LPush(ctx, "write_queue", data)
    return nil
}

// Background worker flushes queue to DB
func (s *UserService) FlushWorker(ctx context.Context) {
    for {
        // Blocking pop — waits for items
        result, err := s.rdb.BRPop(ctx, 5*time.Second, "write_queue").Result()
        if err != nil {
            continue
        }

        var user User
        json.Unmarshal([]byte(result[1]), &user)
        s.db.UpdateUser(ctx, &user)
    }
}
```

**Плюсы**: записи очень быстрые, можно батчить.
**Минусы**: риск потери данных при падении Redis до flush.

## TTL стратегии

```go
// Фиксированный TTL
rdb.Set(ctx, "user:1", data, 15*time.Minute)

// Jitter — разброс чтобы избежать массового expiry
ttl := 15*time.Minute + time.Duration(rand.Intn(60))*time.Second
rdb.Set(ctx, "user:1", data, ttl)

// Sliding TTL — продлевается при каждом чтении
func Get(ctx context.Context, key string) (string, error) {
    val, err := rdb.Get(ctx, key).Result()
    if err != nil {
        return "", err
    }
    // Refresh TTL on read
    rdb.Expire(ctx, key, 15*time.Minute)
    return val, nil
}
```

## Cache Stampede (Thundering Herd)

Когда TTL истекает и сотни горутин одновременно идут в БД.

### Решение 1: singleflight

```go
import "golang.org/x/sync/singleflight"

var sf singleflight.Group

func (s *UserService) GetUser(ctx context.Context, id int64) (*User, error) {
    key := fmt.Sprintf("user:%d", id)

    // Check cache
    cached, err := s.rdb.Get(ctx, key).Bytes()
    if err == nil {
        var user User
        json.Unmarshal(cached, &user)
        return &user, nil
    }

    // Deduplicate concurrent DB calls for same key
    result, err, _ := sf.Do(key, func() (interface{}, error) {
        user, err := s.db.GetUser(ctx, id)
        if err != nil {
            return nil, err
        }

        data, _ := json.Marshal(user)
        s.rdb.Set(ctx, key, data, 15*time.Minute)
        return user, nil
    })
    if err != nil {
        return nil, err
    }

    return result.(*User), nil
}
```

### Решение 2: Lock-based (SETNX)

```go
func (s *UserService) GetWithLock(ctx context.Context, id int64) (*User, error) {
    key := fmt.Sprintf("user:%d", id)

    cached, err := s.rdb.Get(ctx, key).Bytes()
    if err == nil {
        var user User
        json.Unmarshal(cached, &user)
        return &user, nil
    }

    // Try to acquire refresh lock
    lockKey := key + ":lock"
    acquired, _ := s.rdb.SetNX(ctx, lockKey, "1", 10*time.Second).Result()

    if acquired {
        // Winner — fetch from DB and populate cache
        defer s.rdb.Del(ctx, lockKey)
        user, err := s.db.GetUser(ctx, id)
        if err != nil {
            return nil, err
        }
        data, _ := json.Marshal(user)
        s.rdb.Set(ctx, key, data, 15*time.Minute)
        return user, nil
    }

    // Losers — wait and retry cache
    time.Sleep(50 * time.Millisecond)
    return s.GetWithLock(ctx, id)
}
```

## Инвалидация кэша

| Стратегия | Описание | Когда |
|-----------|----------|-------|
| TTL expiry | Кэш сам истекает | Read-heavy, можно терпеть stale data |
| Explicit delete | `DEL key` при изменении | Нужна актуальность |
| Pub/Sub invalidation | Уведомление сервисам удалить | Распределённые сервисы |
| Версионирование | `user:1:v5` — меняем ключ | Атомарная инвалидация |

```go
// Explicit invalidation on update
func (s *UserService) UpdateUser(ctx context.Context, user *User) error {
    if err := s.db.UpdateUser(ctx, user); err != nil {
        return err
    }

    // Delete — next read will repopulate (Cache-Aside)
    key := fmt.Sprintf("user:%d", user.ID)
    return s.rdb.Del(ctx, key).Err()
}
```

> **Правило**: проще и надёжнее удалять ключ (`DEL`), чем обновлять. Это избавляет от race conditions между UPDATE + SET.

## Cache Warming

Предзаполнение кэша при старте сервиса.

```go
func (s *Service) WarmCache(ctx context.Context) error {
    // Load hot data at startup
    users, err := s.db.GetTopUsers(ctx, 1000)
    if err != nil {
        return err
    }

    pipe := s.rdb.Pipeline()
    for _, u := range users {
        data, _ := json.Marshal(u)
        pipe.Set(ctx, fmt.Sprintf("user:%d", u.ID), data, 30*time.Minute)
    }

    _, err = pipe.Exec(ctx)
    return err
}
```

## Частые вопросы на собеседовании

1. **Как избежать cache stampede?** — `singleflight`, lock через SETNX, early expiry (обновление до истечения TTL).

2. **Cache-Aside vs Write-Through — когда что?** — Cache-Aside для read-heavy (лениво), Write-Through когда важна актуальность кэша.

3. **Почему лучше DEL а не SET при обновлении?** — Избегаем race condition: между чтением из БД и SET кто-то мог обновить запись. DEL безопасен — следующий read заполнит актуальное значение.

4. **Что делать при падении Redis?** — Fallback на БД (degraded mode), circuit breaker, не падать с ошибкой.
