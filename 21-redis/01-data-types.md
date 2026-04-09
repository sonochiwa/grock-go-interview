# Типы данных Redis

## Подключение к Redis

Все примеры используют библиотеку `github.com/redis/go-redis/v9`.

```go
import (
    "context"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
)

func newClient() *redis.Client {
    return redis.NewClient(&redis.Options{
        Addr:     "localhost:6379",
        Password: "",
        DB:       0,
    })
}
```

---

## 1. Strings

Самый базовый тип данных Redis. Строка может содержать любые данные: текст, JSON, бинарные данные. Максимальный размер значения -- 512 MB.

### Основные команды

| Команда | Описание | Сложность |
|---------|----------|-----------|
| `SET key value` | Установить значение | O(1) |
| `GET key` | Получить значение | O(1) |
| `MSET k1 v1 k2 v2` | Установить несколько ключей | O(N) |
| `MGET k1 k2` | Получить несколько ключей | O(N) |
| `INCR key` | Атомарный инкремент на 1 | O(1) |
| `DECR key` | Атомарный декремент на 1 | O(1) |
| `INCRBY key n` | Атомарный инкремент на n | O(1) |
| `SETEX key sec value` | SET с TTL в секундах | O(1) |
| `SETNX key value` | SET только если ключ не существует | O(1) |
| `GETSET key value` | Установить новое, вернуть старое | O(1) |

### Go-примеры

```go
func stringsExample(ctx context.Context, rdb *redis.Client) error {
    // Basic SET/GET
    err := rdb.Set(ctx, "user:1:name", "Alice", 0).Err()
    if err != nil {
        return err
    }

    name, err := rdb.Get(ctx, "user:1:name").Result()
    if err != nil {
        return err
    }
    fmt.Println("Name:", name) // Name: Alice

    // SET with TTL (equivalent to SETEX)
    err = rdb.Set(ctx, "session:abc123", "user_data", 30*time.Minute).Err()
    if err != nil {
        return err
    }

    // SETNX -- set only if key does not exist
    wasSet, err := rdb.SetNX(ctx, "lock:resource", "owner1", 10*time.Second).Result()
    if err != nil {
        return err
    }
    fmt.Println("Lock acquired:", wasSet)

    // MSET/MGET -- batch operations
    err = rdb.MSet(ctx, "key1", "val1", "key2", "val2", "key3", "val3").Err()
    if err != nil {
        return err
    }

    vals, err := rdb.MGet(ctx, "key1", "key2", "key3").Result()
    if err != nil {
        return err
    }
    for i, v := range vals {
        fmt.Printf("key%d = %v\n", i+1, v)
    }

    // Atomic counters
    rdb.Set(ctx, "page:views", "0", 0)

    newVal, err := rdb.Incr(ctx, "page:views").Result()
    if err != nil {
        return err
    }
    fmt.Println("Views:", newVal) // Views: 1

    newVal, err = rdb.IncrBy(ctx, "page:views", 10).Result()
    if err != nil {
        return err
    }
    fmt.Println("Views:", newVal) // Views: 11

    newVal, err = rdb.Decr(ctx, "page:views").Result()
    if err != nil {
        return err
    }
    fmt.Println("Views:", newVal) // Views: 10

    return nil
}
```

### Когда использовать Strings

- Кэширование сериализованных объектов (JSON)
- Атомарные счётчики (просмотры, лайки)
- Хранение сессий
- Распределённые блокировки (SETNX)

---

## 2. Lists

Двусвязный список строк. Поддерживает операции с обоих концов за O(1). Полезен для очередей, стеков и хранения последних N элементов.

### Основные команды

| Команда | Описание | Сложность |
|---------|----------|-----------|
| `LPUSH key val` | Добавить в начало | O(1) |
| `RPUSH key val` | Добавить в конец | O(1) |
| `LPOP key` | Извлечь из начала | O(1) |
| `RPOP key` | Извлечь из конца | O(1) |
| `LRANGE key start stop` | Получить диапазон элементов | O(S+N) |
| `LLEN key` | Длина списка | O(1) |
| `BLPOP key timeout` | Блокирующий LPOP | O(1) |
| `BRPOP key timeout` | Блокирующий RPOP | O(1) |
| `LREM key count val` | Удалить элементы по значению | O(N) |

### Go-примеры

```go
func listsExample(ctx context.Context, rdb *redis.Client) error {
    key := "task:queue"
    rdb.Del(ctx, key)

    // RPUSH -- add to the end (enqueue)
    rdb.RPush(ctx, key, "task1", "task2", "task3")

    // LPUSH -- add to the beginning
    rdb.LPush(ctx, key, "urgent-task")

    // LRANGE -- get all elements (0 to -1 = all)
    tasks, err := rdb.LRange(ctx, key, 0, -1).Result()
    if err != nil {
        return err
    }
    fmt.Println("Tasks:", tasks)
    // Tasks: [urgent-task task1 task2 task3]

    // LPOP -- dequeue from front (FIFO queue: RPUSH + LPOP)
    task, err := rdb.LPop(ctx, key).Result()
    if err != nil {
        return err
    }
    fmt.Println("Processing:", task) // Processing: urgent-task

    // LLEN -- list length
    length, _ := rdb.LLen(ctx, key).Result()
    fmt.Println("Remaining:", length) // Remaining: 3

    return nil
}

// Worker with blocking pop -- waits for new tasks
func blockingWorker(ctx context.Context, rdb *redis.Client) {
    for {
        // BLPOP blocks until an element is available or timeout
        result, err := rdb.BLPop(ctx, 5*time.Second, "job:queue").Result()
        if err == redis.Nil {
            fmt.Println("No jobs, waiting...")
            continue
        }
        if err != nil {
            fmt.Println("Error:", err)
            return
        }
        // result[0] = key name, result[1] = value
        fmt.Printf("Got job from %s: %s\n", result[0], result[1])
    }
}
```

### Когда использовать Lists

- Очереди задач (RPUSH + LPOP -- FIFO)
- Стеки (LPUSH + LPOP -- LIFO)
- Хранение последних N элементов (LPUSH + LTRIM)
- Блокирующие очереди для воркеров (BLPOP/BRPOP)

---

## 3. Sets

Неупорядоченная коллекция уникальных строк. Поддерживает операции объединения, пересечения и разности множеств.

### Основные команды

| Команда | Описание | Сложность |
|---------|----------|-----------|
| `SADD key member` | Добавить элемент | O(1) |
| `SREM key member` | Удалить элемент | O(1) |
| `SMEMBERS key` | Получить все элементы | O(N) |
| `SISMEMBER key member` | Проверить наличие элемента | O(1) |
| `SCARD key` | Количество элементов | O(1) |
| `SUNION key1 key2` | Объединение множеств | O(N) |
| `SINTER key1 key2` | Пересечение множеств | O(N*M) |
| `SDIFF key1 key2` | Разность множеств | O(N) |

### Go-примеры

```go
func setsExample(ctx context.Context, rdb *redis.Client) error {
    // Tags for articles
    rdb.SAdd(ctx, "article:1:tags", "go", "redis", "backend")
    rdb.SAdd(ctx, "article:2:tags", "go", "kafka", "backend")
    rdb.SAdd(ctx, "article:3:tags", "react", "frontend")

    // Check if article has a specific tag
    hasTag, _ := rdb.SIsMember(ctx, "article:1:tags", "redis").Result()
    fmt.Println("Article 1 has redis tag:", hasTag) // true

    // Get all tags for an article
    tags, _ := rdb.SMembers(ctx, "article:1:tags").Result()
    fmt.Println("Article 1 tags:", tags)

    // Common tags between articles (intersection)
    common, _ := rdb.SInter(ctx, "article:1:tags", "article:2:tags").Result()
    fmt.Println("Common tags:", common) // [go backend]

    // All unique tags (union)
    all, _ := rdb.SUnion(ctx, "article:1:tags", "article:2:tags").Result()
    fmt.Println("All tags:", all)

    // Tags in article 1 but not in article 2 (difference)
    diff, _ := rdb.SDiff(ctx, "article:1:tags", "article:2:tags").Result()
    fmt.Println("Unique to article 1:", diff) // [redis]

    // Online users tracking
    rdb.SAdd(ctx, "online:users", "user:1", "user:2", "user:3")
    rdb.SRem(ctx, "online:users", "user:2") // user went offline

    count, _ := rdb.SCard(ctx, "online:users").Result()
    fmt.Println("Online users:", count) // 2

    return nil
}
```

### Когда использовать Sets

- Теги, категории, метки
- Отслеживание уникальных посетителей
- Отношения (друзья, подписчики)
- Проверка уникальности (дедупликация)

---

## 4. Sorted Sets (ZSets)

Множество с весом (score) для каждого элемента. Элементы автоматически сортируются по score. Идеальный тип для рейтингов и таймлайнов.

### Основные команды

| Команда | Описание | Сложность |
|---------|----------|-----------|
| `ZADD key score member` | Добавить с весом | O(log N) |
| `ZREM key member` | Удалить элемент | O(log N) |
| `ZRANGE key start stop` | Элементы по позиции (asc) | O(log N + M) |
| `ZREVRANGE key start stop` | Элементы по позиции (desc) | O(log N + M) |
| `ZRANGEBYSCORE key min max` | Элементы по диапазону score | O(log N + M) |
| `ZRANK key member` | Позиция элемента (asc) | O(log N) |
| `ZREVRANK key member` | Позиция элемента (desc) | O(log N) |
| `ZSCORE key member` | Получить score элемента | O(1) |
| `ZINCRBY key incr member` | Увеличить score | O(log N) |
| `ZCARD key` | Количество элементов | O(1) |

### Go-примеры

```go
func sortedSetsExample(ctx context.Context, rdb *redis.Client) error {
    key := "leaderboard:weekly"
    rdb.Del(ctx, key)

    // Add players with scores
    rdb.ZAdd(ctx, key,
        redis.Z{Score: 1500, Member: "alice"},
        redis.Z{Score: 2300, Member: "bob"},
        redis.Z{Score: 1800, Member: "charlie"},
        redis.Z{Score: 3100, Member: "diana"},
        redis.Z{Score: 2100, Member: "eve"},
    )

    // Top 3 players (descending order)
    top3, _ := rdb.ZRevRangeWithScores(ctx, key, 0, 2).Result()
    fmt.Println("=== Top 3 Players ===")
    for i, z := range top3 {
        fmt.Printf("#%d %s -- %.0f points\n", i+1, z.Member, z.Score)
    }
    // #1 diana -- 3100 points
    // #2 bob -- 2300 points
    // #3 eve -- 2100 points

    // Get player rank (0-based, descending)
    rank, _ := rdb.ZRevRank(ctx, key, "charlie").Result()
    fmt.Printf("Charlie's rank: #%d\n", rank+1) // #4

    // Get player score
    score, _ := rdb.ZScore(ctx, key, "alice").Result()
    fmt.Printf("Alice's score: %.0f\n", score) // 1500

    // Increment score (alice wins a match)
    newScore, _ := rdb.ZIncrBy(ctx, key, 500, "alice").Result()
    fmt.Printf("Alice's new score: %.0f\n", newScore) // 2000

    // Players with score between 2000 and 3000
    opt := &redis.ZRangeBy{
        Min: "2000",
        Max: "3000",
    }
    midRange, _ := rdb.ZRangeByScoreWithScores(ctx, key, opt).Result()
    fmt.Println("Players (2000-3000):")
    for _, z := range midRange {
        fmt.Printf("  %s: %.0f\n", z.Member, z.Score)
    }

    return nil
}
```

### Когда использовать Sorted Sets

- Лидерборды и рейтинги
- Таймлайны (score = timestamp)
- Rate limiting (sliding window)
- Приоритетные очереди
- Автокомплит с весами (score = частота)

---

## 5. Hashes

Коллекция пар field-value, привязанная к одному ключу. Идеально подходит для представления объектов. Экономнее по памяти, чем отдельные строки для каждого поля.

### Основные команды

| Команда | Описание | Сложность |
|---------|----------|-----------|
| `HSET key field value` | Установить поле | O(1) |
| `HGET key field` | Получить поле | O(1) |
| `HGETALL key` | Получить все поля и значения | O(N) |
| `HMSET key f1 v1 f2 v2` | Установить несколько полей | O(N) |
| `HDEL key field` | Удалить поле | O(1) |
| `HEXISTS key field` | Проверить наличие поля | O(1) |
| `HINCRBY key field n` | Атомарный инкремент поля | O(1) |
| `HKEYS key` | Получить все ключи полей | O(N) |
| `HVALS key` | Получить все значения | O(N) |
| `HLEN key` | Количество полей | O(1) |

### Go-примеры

```go
func hashesExample(ctx context.Context, rdb *redis.Client) error {
    key := "user:1000"
    rdb.Del(ctx, key)

    // Store user profile as hash
    rdb.HSet(ctx, key,
        "name", "Alice",
        "email", "alice@example.com",
        "age", 28,
        "login_count", 0,
    )

    // Get single field
    email, _ := rdb.HGet(ctx, key, "email").Result()
    fmt.Println("Email:", email)

    // Get all fields
    profile, _ := rdb.HGetAll(ctx, key).Result()
    fmt.Println("Profile:", profile)
    // map[name:Alice email:alice@example.com age:28 login_count:0]

    // Increment login counter atomically
    rdb.HIncrBy(ctx, key, "login_count", 1)

    // Check if field exists
    exists, _ := rdb.HExists(ctx, key, "phone").Result()
    fmt.Println("Has phone:", exists) // false

    // Delete a field
    rdb.HDel(ctx, key, "age")

    // Scan hash into struct
    var user struct {
        Name       string `redis:"name"`
        Email      string `redis:"email"`
        LoginCount int    `redis:"login_count"`
    }
    err := rdb.HGetAll(ctx, key).Scan(&user)
    if err != nil {
        return err
    }
    fmt.Printf("User: %+v\n", user)

    return nil
}
```

### Когда использовать Hashes

- Профили пользователей и объекты
- Конфигурации и настройки
- Корзина товаров (field = product_id, value = quantity)
- Счётчики по категориям (HINCRBY)

### Strings vs Hashes для объектов

| Подход | Плюсы | Минусы |
|--------|-------|--------|
| `SET user:1 "{json}"` | Просто, один ключ | Нужно десериализовать весь объект |
| `SET user:1:name "Alice"` | Отдельные поля | Много ключей, дорого по памяти |
| `HSET user:1 name "Alice"` | Атомарные поля, экономно | Нет вложенности |

Для большинства случаев **Hashes** -- лучший выбор для хранения объектов в Redis.

---

## 6. Streams

Append-only лог с поддержкой consumer groups. Появились в Redis 5.0. Подходят для event sourcing, логирования и обмена сообщениями.

### Основные команды

| Команда | Описание |
|---------|----------|
| `XADD stream * field value` | Добавить запись (auto-ID) |
| `XREAD COUNT n STREAMS stream id` | Читать записи после ID |
| `XRANGE stream start end` | Диапазон записей |
| `XLEN stream` | Количество записей |
| `XGROUP CREATE stream group id` | Создать consumer group |
| `XREADGROUP GROUP g consumer COUNT n STREAMS stream >` | Читать как consumer |
| `XACK stream group id` | Подтвердить обработку |

### Go-примеры

```go
func streamsExample(ctx context.Context, rdb *redis.Client) error {
    stream := "events:orders"
    rdb.Del(ctx, stream)

    // XADD -- produce events
    for i := 0; i < 5; i++ {
        id, err := rdb.XAdd(ctx, &redis.XAddArgs{
            Stream: stream,
            Values: map[string]interface{}{
                "type":    "order_created",
                "user_id": fmt.Sprintf("user:%d", i),
                "amount":  (i + 1) * 100,
            },
        }).Result()
        if err != nil {
            return err
        }
        fmt.Println("Added event:", id)
    }

    // XRANGE -- read all entries
    entries, _ := rdb.XRange(ctx, stream, "-", "+").Result()
    fmt.Printf("Total entries: %d\n", len(entries))
    for _, e := range entries {
        fmt.Printf("  %s: %v\n", e.ID, e.Values)
    }

    // XLEN -- stream length
    length, _ := rdb.XLen(ctx, stream).Result()
    fmt.Println("Stream length:", length)

    return nil
}

// Consumer group example
func consumerGroupExample(ctx context.Context, rdb *redis.Client) error {
    stream := "events:orders"
    group := "order-processors"
    consumer := "worker-1"

    // Create consumer group (start from beginning)
    err := rdb.XGroupCreateMkStream(ctx, stream, group, "0").Err()
    if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
        return err
    }

    for {
        // Read new messages for this consumer
        results, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
            Group:    group,
            Consumer: consumer,
            Streams:  []string{stream, ">"},
            Count:    10,
            Block:    5 * time.Second,
        }).Result()
        if err == redis.Nil {
            continue // no new messages
        }
        if err != nil {
            return err
        }

        for _, msg := range results[0].Messages {
            fmt.Printf("[%s] Processing: %v\n", consumer, msg.Values)

            // ACK after successful processing
            rdb.XAck(ctx, stream, group, msg.ID)
        }
    }
}
```

### Когда использовать Streams

- Event sourcing и audit log
- Обмен сообщениями между сервисами (с гарантией доставки)
- Обработка событий с consumer groups
- Замена Kafka для небольших нагрузок

---

## Сравнительная таблица типов данных

| Тип | Основное применение | Макс. размер | Сложность доступа |
|-----|--------------------|--------------|--------------------|
| String | Кэш, счётчики, сессии | 512 MB | O(1) |
| List | Очереди, стеки, последние N | 4B элементов | O(1) push/pop |
| Set | Уникальные коллекции, теги | 4B элементов | O(1) add/check |
| Sorted Set | Рейтинги, таймлайны | 4B элементов | O(log N) |
| Hash | Объекты, профили | 4B полей | O(1) по полю |
| Stream | Событийный лог | -- | O(1) append |

---

## Вопросы для собеседования

1. **Чем отличается SETEX от SET с опцией EX?** -- Функционально ничем, `SET key value EX seconds` -- предпочтительный способ начиная с Redis 2.6.12, так как это одна атомарная операция.

2. **Когда использовать Hash вместо String для хранения объекта?** -- Hash лучше, когда нужно читать/обновлять отдельные поля без десериализации всего объекта. String (JSON) лучше для вложенных структур, которые читаются целиком.

3. **В чём разница между LPOP и BLPOP?** -- BLPOP блокирует соединение до появления элемента или таймаута. Используется для реализации воркеров, которые ждут новые задачи без polling.

4. **Зачем нужны Streams, если есть Pub/Sub?** -- Streams обеспечивают персистентность сообщений, consumer groups (распределённая обработка), ACK (гарантия доставки), повторное чтение. Pub/Sub -- fire-and-forget.

5. **Как реализовать rate limiter на Sorted Sets?** -- Используем timestamp как score: ZADD с текущим временем, ZREMRANGEBYSCORE для удаления старых записей, ZCARD для подсчёта запросов в окне.
