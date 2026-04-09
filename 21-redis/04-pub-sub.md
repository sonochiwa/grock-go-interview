# Pub/Sub

## Как работает

```
Publisher ──PUBLISH channel msg──▶ Redis ──▶ Subscriber 1
                                       ──▶ Subscriber 2
                                       ──▶ Subscriber 3
```

Сообщения доставляются **в реальном времени** всем подписчикам. Нет персистенции — если подписчик оффлайн, сообщение теряется.

## Базовый пример

```go
// Publisher
func PublishEvent(ctx context.Context, rdb *redis.Client, channel string, event any) error {
    data, err := json.Marshal(event)
    if err != nil {
        return err
    }
    return rdb.Publish(ctx, channel, data).Err()
}

// Subscriber
func Subscribe(ctx context.Context, rdb *redis.Client, channel string) {
    sub := rdb.Subscribe(ctx, channel)
    defer sub.Close()

    // Wait for confirmation
    _, err := sub.Receive(ctx)
    if err != nil {
        log.Fatal(err)
    }

    ch := sub.Channel()
    for msg := range ch {
        fmt.Printf("channel=%s payload=%s\n", msg.Channel, msg.Payload)
    }
}
```

## Pattern Subscribe

Подписка по шаблону — один подписчик на множество каналов.

```go
// Subscribe to all user events: user.created, user.updated, user.deleted
sub := rdb.PSubscribe(ctx, "user.*")
defer sub.Close()

ch := sub.Channel()
for msg := range ch {
    switch msg.Channel {
    case "user.created":
        handleUserCreated(msg.Payload)
    case "user.updated":
        handleUserUpdated(msg.Payload)
    case "user.deleted":
        handleUserDeleted(msg.Payload)
    }
}
```

## Практический пример: инвалидация кэша

```go
// Service A: обновляет пользователя и публикует событие
func (s *UserService) UpdateUser(ctx context.Context, user *User) error {
    if err := s.db.UpdateUser(ctx, user); err != nil {
        return err
    }

    // Notify other instances to invalidate cache
    event := CacheInvalidation{Key: fmt.Sprintf("user:%d", user.ID)}
    data, _ := json.Marshal(event)
    s.rdb.Publish(ctx, "cache.invalidate", data)

    return nil
}

// All instances: listen and invalidate local cache
func (s *UserService) ListenInvalidations(ctx context.Context) {
    sub := s.rdb.Subscribe(ctx, "cache.invalidate")
    defer sub.Close()

    for msg := range sub.Channel() {
        var event CacheInvalidation
        json.Unmarshal([]byte(msg.Payload), &event)

        s.localCache.Delete(event.Key)
        s.rdb.Del(ctx, event.Key)
    }
}
```

## Ограничения Pub/Sub

| Ограничение | Последствие |
|-------------|-------------|
| Нет персистенции | Оффлайн-подписчик пропускает сообщения |
| Fire-and-forget | Нет подтверждения доставки |
| Нет consumer groups | Нельзя распределить нагрузку между подписчиками |
| Нет replay | Нельзя перечитать историю |
| Все получают всё | Нет партиционирования |

## Pub/Sub vs Streams vs Kafka

| | Pub/Sub | Redis Streams | Kafka |
|---|---|---|---|
| **Персистенция** | ❌ | ✅ | ✅ |
| **Consumer groups** | ❌ | ✅ | ✅ |
| **Replay** | ❌ | ✅ | ✅ |
| **Гарантия доставки** | At-most-once | At-least-once | At-least-once / Exactly-once |
| **Latency** | Микросекунды | Микросекунды | Миллисекунды |
| **Масштаб** | Тысячи msg/s | Десятки тысяч msg/s | Миллионы msg/s |
| **Когда** | Real-time уведомления | Легковесная очередь | Серьёзный messaging |

## Redis Streams (кратко)

Если нужна персистенция и consumer groups, но Kafka — overkill.

```go
// Producer
rdb.XAdd(ctx, &redis.XAddArgs{
    Stream: "orders",
    Values: map[string]interface{}{
        "user_id": "123",
        "total":   "99.99",
    },
})

// Consumer group
rdb.XGroupCreateMkStream(ctx, "orders", "order-service", "0")

// Consumer
results, _ := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
    Group:    "order-service",
    Consumer: "worker-1",
    Streams:  []string{"orders", ">"},
    Count:    10,
    Block:    5 * time.Second,
}).Result()

for _, stream := range results {
    for _, msg := range stream.Messages {
        processOrder(msg.Values)
        // Acknowledge
        rdb.XAck(ctx, "orders", "order-service", msg.ID)
    }
}
```

## Частые вопросы

1. **Когда Pub/Sub, а когда Kafka?** — Pub/Sub для real-time нотификаций где потеря допустима (cache invalidation, live updates). Kafka когда нужны гарантии и replay.

2. **Что будет если подписчик тормозит?** — Redis буферизует сообщения в памяти. При переполнении буфера Redis **отключит** медленного подписчика (`client-output-buffer-limit`).

3. **Можно ли использовать Pub/Sub для очереди задач?** — Нет. Все подписчики получают все сообщения (broadcast). Для очередей — Lists (`BRPOP`) или Streams.
