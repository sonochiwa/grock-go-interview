# Конфигурация и эксплуатация

## Eviction Policies

Когда Redis достигает `maxmemory`, он выбирает что удалить по политике вытеснения.

| Политика | Область | Описание |
|----------|---------|----------|
| `noeviction` | — | Ошибка OOM при записи (по умолчанию) |
| `allkeys-lru` | Все ключи | Удаляет наименее недавно использованные |
| `volatile-lru` | Только с TTL | LRU среди ключей с expiry |
| `allkeys-lfu` | Все ключи | Удаляет наименее часто используемые |
| `volatile-lfu` | Только с TTL | LFU среди ключей с expiry |
| `allkeys-random` | Все ключи | Случайное удаление |
| `volatile-random` | Только с TTL | Случайное среди ключей с expiry |
| `volatile-ttl` | Только с TTL | Удаляет с наименьшим оставшимся TTL |

> **Для кэша**: `allkeys-lfu` или `allkeys-lru`. Для сессий с TTL: `volatile-lru`.

```bash
# Настройка
redis-cli CONFIG SET maxmemory 2gb
redis-cli CONFIG SET maxmemory-policy allkeys-lfu
```

### LRU vs LFU

- **LRU** (Least Recently Used) — вытесняет то, что давно не читали. Хорош для «горячих» данных.
- **LFU** (Least Frequently Used) — вытесняет то, что редко читают. Лучше когда есть долгоживущие популярные ключи.

> Redis использует **приблизительный** LRU/LFU — семплирует `maxmemory-samples` (по умолчанию 5) ключей и вытесняет худший из них.

## Persistence

### RDB (снимки)

Периодический дамп всей БД на диск.

```bash
# redis.conf
save 900 1        # Snapshot if 1 key changed in 900 sec
save 300 10       # Snapshot if 10 keys changed in 300 sec
save 60 10000     # Snapshot if 10000 keys changed in 60 sec

dbfilename dump.rdb
dir /var/lib/redis
```

**Плюсы**: компактный файл, быстрый restart.
**Минусы**: можно потерять данные между снимками.

### AOF (Append-Only File)

Логирует каждую операцию записи.

```bash
# redis.conf
appendonly yes
appendfsync everysec   # Компромисс: fsync раз в секунду
# appendfsync always   # Максимальная надёжность (медленно)
# appendfsync no       # OS решает когда flush
```

**Плюсы**: потеря максимум 1 секунды данных.
**Минусы**: файл больше RDB, медленнее restart.

### RDB + AOF (рекомендуется)

```bash
appendonly yes
appendfsync everysec
save 900 1
```

При рестарте Redis загружает AOF (полнее). RDB — для бэкапов.

## Redis Sentinel (High Availability)

Автоматический failover при падении master.

```
┌──────────┐     ┌──────────┐     ┌──────────┐
│ Sentinel │     │ Sentinel │     │ Sentinel │
└────┬─────┘     └────┬─────┘     └────┬─────┘
     │                │                │
     ▼                ▼                ▼
┌──────────┐     ┌──────────┐     ┌──────────┐
│  Master  │────▶│ Replica  │     │ Replica  │
└──────────┘     └──────────┘     └──────────┘
```

```go
// go-redis подключение через Sentinel
rdb := redis.NewFailoverClient(&redis.FailoverOptions{
    MasterName:    "mymaster",
    SentinelAddrs: []string{
        "sentinel-1:26379",
        "sentinel-2:26379",
        "sentinel-3:26379",
    },
    Password:      "secret",
    DB:            0,
    PoolSize:      10,
})
```

Sentinel решает:
- **Мониторинг** — проверяет что master/replicas живы
- **Уведомление** — оповещает клиентов о смене master
- **Failover** — промоутит replica в master при недоступности

## Redis Cluster (шардирование)

Данные распределяются по нодам через **hash slots** (16384 слота).

```
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│   Node A    │  │   Node B    │  │   Node C    │
│ slots 0-5460│  │slots 5461-  │  │slots 10923- │
│             │  │    10922    │  │    16383    │
│  + Replica  │  │  + Replica  │  │  + Replica  │
└─────────────┘  └─────────────┘  └─────────────┘

slot = CRC16(key) % 16384
```

```go
rdb := redis.NewClusterClient(&redis.ClusterOptions{
    Addrs: []string{
        "node-1:6379",
        "node-2:6379",
        "node-3:6379",
    },
    Password:     "secret",
    PoolSize:     10,
    ReadOnly:     true,           // Read from replicas
    RouteByLatency: true,         // Route reads to nearest
})
```

### Hash Tags

Для multi-key операций ключи должны быть на одной ноде:

```go
// {user:123} — Redis хеширует только содержимое в {}
rdb.Set(ctx, "{user:123}:profile", profileData, 0)
rdb.Set(ctx, "{user:123}:settings", settingsData, 0)
// Оба ключа на одной ноде — можно использовать в pipeline/transaction
```

## Пул соединений в go-redis

```go
rdb := redis.NewClient(&redis.Options{
    Addr:     "localhost:6379",
    Password: "",
    DB:       0,

    // Pool settings
    PoolSize:     10,               // Max connections (default: 10 * NumCPU)
    MinIdleConns: 5,                // Keep warm connections
    MaxIdleConns: 10,               // Max idle connections
    PoolTimeout:  30 * time.Second, // Wait for available connection

    // Timeouts
    DialTimeout:  5 * time.Second,
    ReadTimeout:  3 * time.Second,
    WriteTimeout: 3 * time.Second,

    // Retry
    MaxRetries:      3,
    MinRetryBackoff: 8 * time.Millisecond,
    MaxRetryBackoff: 512 * time.Millisecond,
})

// Health check
if err := rdb.Ping(ctx).Err(); err != nil {
    log.Fatal("redis unavailable:", err)
}

// Pool stats
stats := rdb.PoolStats()
fmt.Printf("Hits=%d Misses=%d Timeouts=%d TotalConns=%d IdleConns=%d\n",
    stats.Hits, stats.Misses, stats.Timeouts,
    stats.TotalConns, stats.IdleConns)
```

## Sentinel vs Cluster

| | Sentinel | Cluster |
|---|---|---|
| **Назначение** | High Availability | HA + Шардирование |
| **Данные** | Все на master | Распределены по нодам |
| **Масштаб** | Вертикальный | Горизонтальный |
| **Multi-key ops** | Да | Только с hash tags |
| **Максимум RAM** | Размер одной ноды | Сумма всех нод |
| **Когда** | Данных < RAM одной ноды | Данных больше |

## Частые вопросы

1. **Какую eviction policy выбрать для кэша?** — `allkeys-lfu` (Go 1.18+). Если данные примерно одинаково популярны — `allkeys-lru`.

2. **RDB или AOF?** — Оба: AOF для durability, RDB для бэкапов. Только RDB если можно терпеть потерю нескольких минут.

3. **Sentinel или Cluster?** — Sentinel если данные помещаются в RAM одной ноды. Cluster для горизонтального масштабирования.

4. **Сколько ставить `PoolSize`?** — Зависит от нагрузки. Начать с `10 * NumCPU`, мониторить `PoolStats.Timeouts`.
