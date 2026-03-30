# Kafka: Consumer паттерны

## Consumer Group Handler (sarama)

```go
type orderConsumer struct {
    db    *sql.DB
    ready chan bool
}

// Setup вызывается при старте consumer session (после rebalance)
func (c *orderConsumer) Setup(session sarama.ConsumerGroupSession) error {
    close(c.ready) // сигнал что consumer готов
    return nil
}

// Cleanup вызывается при завершении session (перед rebalance)
func (c *orderConsumer) Cleanup(session sarama.ConsumerGroupSession) error {
    return nil
}

// ConsumeClaim — основной цикл обработки сообщений
func (c *orderConsumer) ConsumeClaim(
    session sarama.ConsumerGroupSession,
    claim sarama.ConsumerGroupClaim,
) error {
    for msg := range claim.Messages() {
        log.Printf("topic=%s partition=%d offset=%d key=%s",
            msg.Topic, msg.Partition, msg.Offset, msg.Key)

        if err := c.processMessage(session.Context(), msg); err != nil {
            log.Error("process failed", "err", err, "offset", msg.Offset)
            // НЕ коммитим → будет повторная обработка
            continue
        }

        // Коммит offset (at-least-once)
        session.MarkMessage(msg, "") // помечает для коммита
        // Реальный коммит происходит периодически (auto-commit interval)
    }
    return nil
}
```

## Запуск Consumer Group

```go
func runConsumer(ctx context.Context, brokers []string) error {
    config := sarama.NewConfig()
    config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
        sarama.NewBalanceStrategyRoundRobin(),
    }
    config.Consumer.Offsets.Initial = sarama.OffsetNewest // или OffsetOldest
    config.Consumer.Offsets.AutoCommit.Enable = true
    config.Consumer.Offsets.AutoCommit.Interval = 1 * time.Second

    group, err := sarama.NewConsumerGroup(brokers, "order-service", config)
    if err != nil {
        return err
    }
    defer group.Close()

    consumer := &orderConsumer{
        db:    db,
        ready: make(chan bool),
    }

    // Обработка ошибок consumer group
    go func() {
        for err := range group.Errors() {
            log.Error("consumer group error", "err", err)
        }
    }()

    topics := []string{"orders", "payments"}
    for {
        // Consume блокируется до rebalance или context cancel
        if err := group.Consume(ctx, topics, consumer); err != nil {
            if errors.Is(err, sarama.ErrClosedConsumerGroup) {
                return nil
            }
            log.Error("consume error", "err", err)
        }
        if ctx.Err() != nil {
            return ctx.Err()
        }
        consumer.ready = make(chan bool) // reset для следующего rebalance
    }
}
```

## Rebalance стратегии

```
Когда происходит rebalance:
  - Consumer присоединяется к группе
  - Consumer покидает группу (crash, shutdown)
  - Новые partitions добавлены в topic
  - session.timeout.ms превышен (нет heartbeat)
  - max.poll.interval.ms превышен (слишком долгая обработка)

Стратегии:
  RoundRobin: partitions распределяются по кругу
    P0→C0, P1→C1, P2→C0, P3→C1

  Range (default): partitions делятся блоками
    Topic1: P0,P1→C0, P2,P3→C1
    Может быть неравномерно при нескольких topics

  Sticky: минимизирует перемещение partitions
    При rebalance: сохранить максимум текущих назначений
    Лучший выбор для production

  CooperativeSticky: incremental rebalance
    Не отзывает ВСЕ partitions при rebalance
    Только перемещаемые partitions кратковременно недоступны
```

```go
config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
    sarama.NewBalanceStrategySticky(), // рекомендуется
}
```

## Offset Management

```go
// Auto-commit (default)
config.Consumer.Offsets.AutoCommit.Enable = true
config.Consumer.Offsets.AutoCommit.Interval = 1 * time.Second
// session.MarkMessage(msg, "") — помечает offset
// Автоматически коммитится каждую секунду

// Manual commit (больше контроля)
config.Consumer.Offsets.AutoCommit.Enable = false
// В ConsumeClaim:
session.MarkMessage(msg, "")
session.Commit() // явный коммит

// Batch commit (эффективнее)
count := 0
for msg := range claim.Messages() {
    process(msg)
    session.MarkMessage(msg, "")
    count++
    if count%100 == 0 {
        session.Commit()
    }
}

// Reset offset (перечитать с начала)
config.Consumer.Offsets.Initial = sarama.OffsetOldest
// Или через kafka CLI:
// kafka-consumer-groups.sh --reset-offsets --to-earliest --group order-service --topic orders
```

## Конкурентная обработка внутри partition

```go
// Проблема: одна partition → один consumer → один поток
// Если обработка медленная → lag растёт

// Решение: worker pool внутри consumer
func (c *orderConsumer) ConsumeClaim(
    session sarama.ConsumerGroupSession,
    claim sarama.ConsumerGroupClaim,
) error {
    const workers = 10
    sem := make(chan struct{}, workers)
    var mu sync.Mutex
    var lastOffset int64

    for msg := range claim.Messages() {
        sem <- struct{}{} // acquire

        msg := msg // capture
        go func() {
            defer func() { <-sem }() // release

            if err := c.processMessage(session.Context(), msg); err != nil {
                log.Error("process failed", "err", err)
                return
            }

            // ОСТОРОЖНО: offset должен быть последовательным!
            // Если msg offset=5 обработан раньше offset=3 → MarkMessage(5) → при crash offset=3 потеряется
            mu.Lock()
            if msg.Offset > lastOffset {
                session.MarkMessage(msg, "")
                lastOffset = msg.Offset
            }
            mu.Unlock()
        }()
    }

    // Дождаться завершения всех workers
    for i := 0; i < workers; i++ {
        sem <- struct{}{}
    }
    return nil
}

// Лучший подход: batch по partition
// Читаем batch → обрабатываем параллельно → коммитим последний offset
```

## Consumer Lag мониторинг

```
Consumer Lag = Latest Offset - Committed Offset

Мониторинг:
  1. Burrow (LinkedIn) — dedicated lag monitoring
  2. kafka-consumer-groups.sh --describe --group order-service
  3. Prometheus + kafka_exporter

Alerts:
  - Lag > 10000 → WARNING
  - Lag > 100000 → CRITICAL
  - Lag растёт consistently → consumer не справляется

Решения при высоком lag:
  1. Добавить consumers (до кол-ва partitions)
  2. Увеличить partitions + consumers
  3. Оптимизировать обработку сообщений
  4. Параллельная обработка внутри partition (с осторожностью)
  5. Batch processing вместо поодиночке
```

## Частые вопросы

**Q: Что если consumer обрабатывает сообщение дольше max.poll.interval.ms?**
A: Broker считает consumer мёртвым → rebalance → partition отдаётся другому consumer → текущий consumer получит ошибку. Увеличить max.poll.interval.ms или уменьшить max.poll.records.

**Q: Как обработать poison pill (невалидное сообщение)?**
A: Не блокировать consumer! Варианты: skip + log, отправить в DLQ topic, парсить с fallback. Никогда не retry бесконечно.

**Q: auto.offset.reset: earliest vs latest?**
A: `earliest` — обработать все сообщения с начала (для нового consumer group). `latest` — только новые. Для production: `earliest` для первого запуска, потом offset хранится в Kafka.
