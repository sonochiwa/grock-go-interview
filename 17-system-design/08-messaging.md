# Messaging и Event Streaming

## Message Queue vs Event Stream

```
Message Queue (RabbitMQ, SQS):
  - Сообщение доставляется ОДНОМУ consumer
  - После обработки — удаляется
  - Point-to-point коммуникация
  - Пример: отправка email, обработка заказа

Event Stream (Kafka, Kinesis):
  - Событие доставляется ВСЕМ consumers (каждой группе)
  - Хранится на диске (retention period)
  - Можно перечитать (replay)
  - Пример: activity feed, audit log, CDC
```

| | Message Queue | Event Stream |
|---|---|---|
| Модель | Competing consumers | Pub/Sub + Consumer Groups |
| Хранение | До обработки | По retention (дни/недели) |
| Replay | Нет | Да |
| Ordering | Нет гарантий (или FIFO) | В рамках partition |
| Когда | Task queue, RPC | Event-driven, analytics |

## Apache Kafka — подробно

### Архитектура

```
Producer → [Kafka Cluster] → Consumer

Kafka Cluster:
  ┌─────────────────────────────────────────┐
  │  Broker 1        Broker 2    Broker 3   │
  │  ┌──────────┐   ┌─────────┐            │
  │  │Topic: orders│ │Topic: orders│         │
  │  │ Part 0 (L)│  │ Part 0 (F)│          │
  │  │ Part 1 (F)│  │ Part 1 (L)│          │
  │  │ Part 2 (L)│  │ Part 2 (F)│          │
  │  └──────────┘   └─────────┘            │
  └─────────────────────────────────────────┘
  L = Leader, F = Follower (replica)

Ключевые сущности:
  - Broker: сервер Kafka (обычно 3-5 в кластере)
  - Topic: логическая категория сообщений (orders, payments, users)
  - Partition: единица параллелизма внутри topic
  - Offset: порядковый номер сообщения в partition (0, 1, 2, ...)
  - Consumer Group: группа consumers, каждая partition → одному consumer
  - Replication Factor: сколько копий каждой partition (обычно 3)
```

### Partitions и ordering

```
Topic "orders" (3 partitions):

  Partition 0: [msg0, msg1, msg2, msg5, msg8]  → Consumer A
  Partition 1: [msg3, msg4, msg6]               → Consumer B
  Partition 2: [msg7, msg9, msg10]              → Consumer C

Правила:
  1. Ordering гарантирован ТОЛЬКО внутри одной partition
  2. Partition выбирается по key: hash(key) % num_partitions
  3. Сообщения с одним key ВСЕГДА попадают в одну partition
  4. Больше partitions = больше параллелизм

Пример:
  key = user_id → все события пользователя в порядке
  key = order_id → все события заказа в порядке
  key = null → round-robin (нет гарантий порядка)
```

### Consumer Groups

```
Topic "orders" (4 partitions):

Consumer Group "order-service":
  Consumer 1 ← Partition 0, 1
  Consumer 2 ← Partition 2, 3
  → Каждое сообщение обрабатывается ОДНИМ consumer в группе

Consumer Group "analytics":
  Consumer A ← Partition 0, 1, 2, 3
  → Получает ВСЕ те же сообщения независимо

Правила:
  - Consumers в группе > partitions → лишние consumers idle
  - Consumer падает → rebalance (его partitions раздаются другим)
  - Максимальный параллелизм = количество partitions
```

### Offsets и коммиты

```
Partition 0: [0] [1] [2] [3] [4] [5] [6] [7]
                              ↑              ↑
                        committed offset   latest offset
                        (обработано)       (записано)

Стратегии коммита:
  1. Auto-commit (enable.auto.commit=true)
     - Коммит каждые auto.commit.interval.ms (5s default)
     - Риск: crash между обработкой и коммитом → потеря или дубли

  2. Manual commit (рекомендуется):
     - commitSync() — блокирующий, точный
     - commitAsync() — неблокирующий, может потерять

  3. At-least-once (default):
     - Обработал → коммит
     - Crash до коммита → повторная обработка (дубли!)
     - Решение: idempotent consumer (дедупликация по ID)

  4. At-most-once:
     - Коммит → обработка
     - Crash после коммита → потеря сообщения

  5. Exactly-once (transactional):
     - Producer: enable.idempotence=true + transactional.id
     - Consumer: isolation.level=read_committed
     - Kafka Streams / Flink делают это прозрачно
```

### Producer

```go
// Go: github.com/IBM/sarama или github.com/segmentio/kafka-go

// Конфигурация producer
config := sarama.NewConfig()
config.Producer.RequiredAcks = sarama.WaitForAll  // acks=all
config.Producer.Retry.Max = 3
config.Producer.Return.Successes = true
config.Producer.Idempotent = true                 // exactly-once
config.Net.MaxOpenRequests = 1                     // required for idempotent

producer, _ := sarama.NewSyncProducer(brokers, config)
defer producer.Close()

msg := &sarama.ProducerMessage{
    Topic: "orders",
    Key:   sarama.StringEncoder(orderID),    // определяет partition
    Value: sarama.ByteEncoder(jsonBytes),
}
partition, offset, err := producer.SendMessage(msg)
```

```
Producer Acknowledgments (acks):
  acks=0  — fire and forget (быстро, может потерять)
  acks=1  — leader подтвердил (баланс)
  acks=all — все replicas подтвердили (надёжно, медленно)

Batching:
  batch.size = 16KB (default) — размер батча
  linger.ms = 0 (default) — задержка для накопления батча
  compression.type = snappy/lz4/zstd — сжатие (рекомендуется)
```

### Consumer

```go
consumer := &Consumer{} // implements sarama.ConsumerGroupHandler

group, _ := sarama.NewConsumerGroup(brokers, "order-service", config)
defer group.Close()

ctx := context.Background()
for {
    // Consume блокируется до rebalance или cancel
    err := group.Consume(ctx, []string{"orders"}, consumer)
    if err != nil {
        log.Error("consumer error", "err", err)
    }
}

// Handler
type Consumer struct{}

func (c *Consumer) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (c *Consumer) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (c *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession,
    claim sarama.ConsumerGroupClaim) error {
    for msg := range claim.Messages() {
        // Обработка
        processOrder(msg.Value)
        // Manual commit
        session.MarkMessage(msg, "")
    }
    return nil
}
```

### Ключевые конфигурации

```
Broker:
  num.partitions = 6              — partitions по умолчанию
  default.replication.factor = 3  — replicas
  min.insync.replicas = 2         — минимум синхронных реплик
  log.retention.hours = 168       — хранение 7 дней
  log.retention.bytes = -1        — без лимита по размеру
  log.segment.bytes = 1GB         — размер сегмента

Producer:
  acks = all                      — надёжность
  retries = 3                     — повторы
  enable.idempotence = true       — дедупликация
  max.in.flight.requests = 5      — параллельные запросы (1 для strict ordering)

Consumer:
  group.id = "my-service"
  auto.offset.reset = earliest/latest  — что делать с новым consumer
  max.poll.records = 500          — записей за один poll
  session.timeout.ms = 45000     — таймаут heartbeat
  max.poll.interval.ms = 300000  — макс время между poll (5 мин)
```

### Kafka: когда и как

```
Когда использовать Kafka:
  ✅ Event-driven архитектура (микросервисы)
  ✅ Activity tracking / audit log
  ✅ Log aggregation
  ✅ Stream processing (real-time analytics)
  ✅ CDC (Change Data Capture) — Debezium
  ✅ Decoupling микросервисов
  ✅ Буфер между быстрым producer и медленным consumer

Когда НЕ использовать:
  ❌ Простая task queue (→ RabbitMQ, SQS)
  ❌ Мало данных (< 10K msg/s) — overkill
  ❌ Нужен request-response → REST/gRPC
  ❌ Сложный routing по содержимому → RabbitMQ
```

### Производительность Kafka

```
Почему Kafka быстрая:
  1. Sequential I/O — запись append-only в конец лога
     Random I/O: ~100 IOPS (HDD)
     Sequential I/O: ~100 MB/s (HDD), ~500 MB/s (SSD)

  2. Zero-copy — sendfile() syscall
     Без Kafka: Disk → Kernel → User → Kernel → Network
     С zero-copy: Disk → Kernel → Network (минус 2 копирования)

  3. Batching — группировка сообщений
  4. Compression — snappy/lz4/zstd (на уровне batch)
  5. Page cache — OS кэширует файлы в RAM

Типичные числа:
  - Один broker: ~100 MB/s write, ~300 MB/s read
  - Latency: 2-5ms (p99)
  - Один кластер: миллионы msg/s
```

### Kafka Patterns

```
1. Dead Letter Queue (DLQ):
   Сообщение не обработано после N попыток → отправить в DLQ topic
   orders → (fail 3x) → orders-dlq
   Мониторинг + ручная обработка DLQ

2. Compacted Topics:
   Вместо удаления по retention — хранить последнее значение для каждого key
   log.cleanup.policy=compact
   Пример: user-profiles (key=user_id, value=latest profile)

3. Schema Registry (Confluent):
   Avro/Protobuf schema хранится отдельно
   Producer/Consumer валидируют schema
   Schema evolution: backward/forward compatible

4. Transactional Outbox:
   DB Transaction: INSERT order + INSERT outbox_event
   Отдельный процесс: читает outbox → публикует в Kafka → помечает sent
   Гарантирует: БД и Kafka консистентны

5. CDC с Debezium:
   Debezium читает WAL (binlog/WAL) → публикует изменения в Kafka
   Таблица users → topic dbserver.public.users
   Формат: {before: {...}, after: {...}, op: "u"}
```

## RabbitMQ (для сравнения)

```
Модель: AMQP (Advanced Message Queuing Protocol)

Exchange → Routing → Queue → Consumer

Типы Exchange:
  - Direct: точный routing key
  - Topic: wildcard routing (orders.*, orders.#)
  - Fanout: broadcast во все queues
  - Headers: по headers сообщения

Преимущества над Kafka:
  - Гибкий routing
  - Priority queues
  - Delayed messages
  - Проще для простых случаев
  - Message TTL

Недостатки:
  - Нет replay (сообщение удаляется после ack)
  - Хуже throughput (10K-50K msg/s vs миллионы у Kafka)
  - Нет ordering гарантий
```

## Частые вопросы

**Q: Kafka vs RabbitMQ — когда что?**
A: Kafka — event streaming, high throughput, нужен replay, event sourcing. RabbitMQ — task queue, сложный routing, priority, простые сценарии.

**Q: Как гарантировать порядок в Kafka?**
A: Порядок гарантирован только внутри partition. Используй один key для связанных событий (order_id, user_id). max.in.flight.requests=1 для strict ordering.

**Q: Что такое consumer lag?**
A: Разница между latest offset и committed offset. Большой lag = consumer не успевает. Мониторить через Burrow или kafka-consumer-groups.sh.

**Q: Как масштабировать consumer?**
A: Добавить consumers в группу (до кол-ва partitions). Нужно больше? Увеличить partitions (но нельзя уменьшить!).

**Q: Exactly-once в Kafka — реально?**
A: Да, с ограничениями: idempotent producer + transactional API + read_committed isolation. Работает внутри Kafka. Для внешних систем — idempotent consumer (дедупликация).
