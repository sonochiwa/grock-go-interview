# Кэширование

## Уровни кэширования

```
[Client] → [CDN] → [API Gateway Cache] → [Application Cache] → [Database Cache]
                                              ↓
                                          [Redis/Memcached]
```

## Стратегии кэширования

### Cache-Aside (Lazy Loading) — самый частый

```go
func GetUser(ctx context.Context, id int64) (*User, error) {
    // 1. Читаем из кэша
    cached, err := redis.Get(ctx, fmt.Sprintf("user:%d", id))
    if err == nil {
        return deserialize(cached), nil
    }

    // 2. Cache miss — читаем из БД
    user, err := db.GetUser(ctx, id)
    if err != nil {
        return nil, err
    }

    // 3. Записываем в кэш
    redis.Set(ctx, fmt.Sprintf("user:%d", id), serialize(user), 10*time.Minute)
    return user, nil
}
```
```
+ Кэшируются только запрашиваемые данные
+ Простая реализация
- Cache miss = дополнительный RTT к БД
- Stale data (данные в кэше устарели)
```

### Write-Through

```
Write → [Cache] → [DB]  (синхронно)
Read  → [Cache]
```
```
+ Кэш всегда актуален
- Каждая запись медленнее (2 операции)
- Кэшируются даже неиспользуемые данные
```

### Write-Behind (Write-Back)

```
Write → [Cache] → (async) → [DB]
```
```
+ Быстрые записи
+ Batch writes в БД
- Риск потери данных при крэше кэша
- Сложность обеспечения consistency
```

### Read-Through

```
Read → [Cache] ──miss──→ [DB]
         └───── автоматически загружает из БД
```
```
+ Приложение не знает о кэше (абстракция)
- Первый запрос всегда медленный
```

## Invalidation (инвалидация)

Самая сложная проблема кэширования.

```
TTL (Time To Live):
  - Устанавливаем срок жизни (5 мин, 1 час)
  - Просто, но stale data до истечения TTL
  - TTL слишком маленький → нет пользы от кэша
  - TTL слишком большой → stale данные

Event-based:
  - При обновлении данных → delete cache key
  - Точнее, но сложнее
  - Race condition: delete + set одновременно

Versioned:
  - key = "user:42:v5"
  - При обновлении → increment version
  - Старые записи expired по TTL
```

### Cache Stampede (Thundering Herd)

```
Проблема:
  1000 запросов → cache miss (TTL expired)
  → 1000 одновременных запросов в БД!

Решения:
  1. singleflight — дедупликация (см. 05-sync/09-singleflight.md)
  2. Lock + refresh — один поток обновляет, остальные ждут
  3. Stale-while-revalidate — отдавать старые данные пока обновляем
  4. Randomized TTL — TTL ± random offset (не все expire одновременно)
```

```go
// singleflight для cache stampede prevention
var group singleflight.Group

func GetUser(ctx context.Context, id int64) (*User, error) {
    key := fmt.Sprintf("user:%d", id)

    // Проверяем кэш
    if cached, err := redis.Get(ctx, key); err == nil {
        return deserialize(cached), nil
    }

    // singleflight: один запрос к БД
    result, err, _ := group.Do(key, func() (any, error) {
        user, err := db.GetUser(ctx, id)
        if err != nil { return nil, err }
        redis.Set(ctx, key, serialize(user), 10*time.Minute)
        return user, nil
    })
    if err != nil { return nil, err }
    return result.(*User), nil
}
```

## Redis

```
Структуры данных:
  String:     GET/SET — кэш, счётчики, сессии
  Hash:       HGET/HSET — объекты с полями
  List:       LPUSH/RPOP — очереди, последние N
  Set:        SADD/SMEMBERS — уникальные значения, теги
  Sorted Set: ZADD/ZRANGE — рейтинги, leaderboards
  Stream:     XADD/XREAD — event log (lite Kafka)

Фичи:
  - TTL на любой ключ
  - Pub/Sub
  - Lua scripting (атомарные операции)
  - Cluster mode (горизонтальное масштабирование)
  - Sentinel (HA, failover)
  - Persistence: RDB (snapshots) + AOF (write log)

Производительность:
  - Single thread: 100K-200K ops/sec
  - Cluster: 1M+ ops/sec
  - Latency: <1ms (same DC)
```

## Частые вопросы

**Q: Как обеспечить consistency кэша и БД?**
A: Cache-aside + event-based invalidation. При обновлении: сначала update DB, потом delete cache (не set!). Delete + lazy load безопаснее чем update cache.

**Q: Cache-aside vs Write-through?**
A: Cache-aside проще и кэширует только нужное. Write-through лучше для write-heavy с частыми reads тех же данных.

**Q: Redis vs Memcached?**
A: Redis: rich data structures, persistence, Lua, pub/sub, cluster. Memcached: проще, multi-thread (лучше для simple key-value). По умолчанию — Redis.
