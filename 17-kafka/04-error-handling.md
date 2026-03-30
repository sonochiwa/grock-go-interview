# Kafka: Обработка ошибок

## Dead Letter Queue (DLQ)

```
Сообщение не обработано после N попыток → отправить в DLQ topic

orders → (fail 3x) → orders.dlq

DLQ topic:
  - Хранит оригинальное сообщение + metadata об ошибке
  - Мониторинг: alert если в DLQ появляются сообщения
  - Ручная обработка или retry из DLQ
```

```go
type retryableConsumer struct {
    maxRetries int
    dlqWriter  *kafka.Writer
}

func (c *retryableConsumer) processWithRetry(ctx context.Context, msg *sarama.ConsumerMessage) error {
    var lastErr error

    for attempt := 0; attempt <= c.maxRetries; attempt++ {
        lastErr = c.process(ctx, msg)
        if lastErr == nil {
            return nil
        }

        // Не ретраить permanent errors
        if isPermanentError(lastErr) {
            break
        }

        if attempt < c.maxRetries {
            time.Sleep(time.Duration(1<<attempt) * 100 * time.Millisecond)
        }
    }

    // Все попытки исчерпаны → DLQ
    return c.sendToDLQ(ctx, msg, lastErr)
}

func (c *retryableConsumer) sendToDLQ(ctx context.Context, msg *sarama.ConsumerMessage, err error) error {
    dlqMsg := kafka.Message{
        Key:   msg.Key,
        Value: msg.Value,
        Headers: []kafka.Header{
            {Key: "original-topic", Value: []byte(msg.Topic)},
            {Key: "original-partition", Value: []byte(fmt.Sprintf("%d", msg.Partition))},
            {Key: "original-offset", Value: []byte(fmt.Sprintf("%d", msg.Offset))},
            {Key: "error", Value: []byte(err.Error())},
            {Key: "failed-at", Value: []byte(time.Now().UTC().Format(time.RFC3339))},
            {Key: "retry-count", Value: []byte(fmt.Sprintf("%d", c.maxRetries))},
        },
    }
    return c.dlqWriter.WriteMessages(ctx, dlqMsg)
}

func isPermanentError(err error) bool {
    // Ошибки которые не исправятся при retry
    return errors.Is(err, ErrInvalidPayload) ||
        errors.Is(err, ErrValidation) ||
        errors.Is(err, ErrDeserialize)
}
```

## Retry Topics

```
Более гибкий подход: цепочка retry topics с увеличивающейся задержкой

orders → (fail) → orders.retry-1 (delay 1m) →
                   (fail) → orders.retry-2 (delay 10m) →
                             (fail) → orders.retry-3 (delay 1h) →
                                       (fail) → orders.dlq

Реализация задержки:
  - Timestamp в header → consumer проверяет: если рано → sleep или re-publish
  - Или отдельные consumer groups с разными poll intervals
```

```go
type retryRouter struct {
    retryTopics []string // ["orders.retry-1", "orders.retry-2", "orders.retry-3"]
    dlqTopic    string   // "orders.dlq"
    delays      []time.Duration // [1m, 10m, 1h]
    writer      *kafka.Writer
}

func (r *retryRouter) route(ctx context.Context, msg *sarama.ConsumerMessage, err error) error {
    retryCount := getRetryCount(msg)

    if retryCount >= len(r.retryTopics) {
        // Все retry исчерпаны → DLQ
        return r.publish(ctx, r.dlqTopic, msg, retryCount, err)
    }

    return r.publish(ctx, r.retryTopics[retryCount], msg, retryCount+1, err)
}

func (r *retryRouter) publish(ctx context.Context, topic string, msg *sarama.ConsumerMessage, retryCount int, err error) error {
    return r.writer.WriteMessages(ctx, kafka.Message{
        Topic: topic,
        Key:   msg.Key,
        Value: msg.Value,
        Headers: []kafka.Header{
            {Key: "retry-count", Value: []byte(strconv.Itoa(retryCount))},
            {Key: "error", Value: []byte(err.Error())},
            {Key: "process-after", Value: []byte(
                time.Now().Add(r.delays[retryCount-1]).Format(time.RFC3339),
            )},
        },
    })
}

func getRetryCount(msg *sarama.ConsumerMessage) int {
    for _, h := range msg.Headers {
        if string(h.Key) == "retry-count" {
            n, _ := strconv.Atoi(string(h.Value))
            return n
        }
    }
    return 0
}
```

## Idempotent Consumer

```go
// Проблема: at-least-once → дубли возможны
// Решение: дедупликация по уникальному ID

type idempotentProcessor struct {
    db *sql.DB
}

func (p *idempotentProcessor) process(ctx context.Context, msg *sarama.ConsumerMessage) error {
    eventID := getEventID(msg) // из header или из payload

    // Одна транзакция: проверка + обработка
    tx, err := p.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // INSERT ... ON CONFLICT DO NOTHING (PostgreSQL)
    result, err := tx.ExecContext(ctx,
        `INSERT INTO processed_events (event_id, processed_at)
         VALUES ($1, NOW()) ON CONFLICT (event_id) DO NOTHING`,
        eventID,
    )
    if err != nil {
        return err
    }

    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        // Уже обработано → skip
        log.Debug("duplicate event skipped", "event_id", eventID)
        return nil
    }

    // Бизнес-логика в той же транзакции
    if err := p.handleOrder(ctx, tx, msg.Value); err != nil {
        return err
    }

    return tx.Commit()
}

// Таблица:
// CREATE TABLE processed_events (
//     event_id VARCHAR(255) PRIMARY KEY,
//     processed_at TIMESTAMP NOT NULL
// );
// CREATE INDEX idx_processed_events_at ON processed_events(processed_at);
// Периодическая очистка: DELETE WHERE processed_at < NOW() - INTERVAL '7 days'
```

## Graceful Shutdown

```go
func main() {
    ctx, cancel := signal.NotifyContext(context.Background(),
        syscall.SIGINT, syscall.SIGTERM)
    defer cancel()

    group, _ := sarama.NewConsumerGroup(brokers, "order-service", config)

    var wg sync.WaitGroup
    wg.Add(1)
    go func() {
        defer wg.Done()
        for {
            if err := group.Consume(ctx, topics, consumer); err != nil {
                if errors.Is(err, sarama.ErrClosedConsumerGroup) {
                    return
                }
                log.Error("consume error", "err", err)
            }
            if ctx.Err() != nil {
                return
            }
        }
    }()

    <-ctx.Done()
    log.Info("shutting down consumer...")

    // Close отправит LeaveGroup → rebalance
    // Дожидается завершения текущей обработки
    group.Close()
    wg.Wait()
    log.Info("consumer stopped")
}
```

## Частые вопросы

**Q: Сколько хранить processed_events для дедупликации?**
A: Зависит от retention topic + максимального lag. Обычно 7 дней достаточно. Используй TTL или cron для очистки.

**Q: DLQ vs retry topic?**
A: DLQ — простой (fail → DLQ). Retry topics — для transient errors с backoff. Комбинация: retry topics для retriable errors, DLQ как последний рубеж.

**Q: Как мониторить ошибки consumer?**
A: Метрики: error rate, DLQ message count, retry rate. Alerts: DLQ не пустой, consumer lag растёт, error rate > threshold.
