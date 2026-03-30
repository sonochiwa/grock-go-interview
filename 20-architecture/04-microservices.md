# Microservices Patterns

## API Gateway

```
Единая точка входа для всех клиентов

[Mobile] ──┐
[Web]    ──┤→ [API Gateway] → [User Service]
[3rd party]┘       ↓          [Order Service]
                   ↓          [Payment Service]
              [Auth, Rate Limit, Logging]

Обязанности:
  - Routing (path → service)
  - Authentication / Authorization
  - Rate limiting
  - Request/Response transformation
  - Aggregation (один запрос клиента → несколько сервисов)
  - Caching
  - Circuit breaking

Реализации: Kong, Envoy, AWS API Gateway, Traefik
Go: KrakenD, собственный на net/http

BFF (Backend for Frontend):
  Отдельный gateway для каждого клиента
  [Mobile BFF] → оптимизирован для mobile (меньше данных)
  [Web BFF]    → оптимизирован для web (больше данных)
```

## Service Mesh

```
Sidecar proxy рядом с каждым сервисом

[Service A] ←→ [Envoy Proxy] ←→ [Envoy Proxy] ←→ [Service B]

Что даёт (без изменения кода!):
  - mTLS между сервисами (zero-trust)
  - Traffic management (canary, retries, circuit breaker)
  - Observability (metrics, traces, access logs)
  - Rate limiting
  - Load balancing (L7)

Реализации: Istio, Linkerd, Consul Connect

Когда нужен:
  ✅ > 10 микросервисов
  ✅ Нужен mTLS без изменения кода
  ✅ Сложные traffic policies

Когда НЕ нужен:
  ❌ Монолит или 3-5 сервисов
  ❌ Маленькая команда (overhead управления)
```

## Saga (реализация в Go)

```go
// Orchestration Saga
type SagaStep struct {
    Name       string
    Execute    func(ctx context.Context, data any) error
    Compensate func(ctx context.Context, data any) error
}

type Saga struct {
    steps []SagaStep
}

func (s *Saga) Run(ctx context.Context, data any) error {
    var completedSteps []SagaStep

    for _, step := range s.steps {
        slog.Info("saga step executing", "step", step.Name)

        if err := step.Execute(ctx, data); err != nil {
            slog.Error("saga step failed", "step", step.Name, "err", err)

            // Compensate в обратном порядке
            for i := len(completedSteps) - 1; i >= 0; i-- {
                cs := completedSteps[i]
                slog.Info("compensating", "step", cs.Name)
                if compErr := cs.Compensate(ctx, data); compErr != nil {
                    slog.Error("compensation failed", "step", cs.Name, "err", compErr)
                    // Alert! Manual intervention needed
                }
            }
            return fmt.Errorf("saga failed at step %s: %w", step.Name, err)
        }

        completedSteps = append(completedSteps, step)
    }

    return nil
}

// Использование
saga := &Saga{
    steps: []SagaStep{
        {
            Name:       "create_order",
            Execute:    func(ctx context.Context, d any) error { return orderSvc.Create(ctx, d.(*OrderData)) },
            Compensate: func(ctx context.Context, d any) error { return orderSvc.Cancel(ctx, d.(*OrderData).ID) },
        },
        {
            Name:       "process_payment",
            Execute:    func(ctx context.Context, d any) error { return paymentSvc.Charge(ctx, d.(*OrderData)) },
            Compensate: func(ctx context.Context, d any) error { return paymentSvc.Refund(ctx, d.(*OrderData).PaymentID) },
        },
        {
            Name:       "reserve_inventory",
            Execute:    func(ctx context.Context, d any) error { return inventorySvc.Reserve(ctx, d.(*OrderData)) },
            Compensate: func(ctx context.Context, d any) error { return inventorySvc.Release(ctx, d.(*OrderData)) },
        },
    },
}

err := saga.Run(ctx, orderData)
```

## Outbox Pattern (реализация в Go)

```go
// Outbox writer — в одной транзакции с бизнес-логикой
func (r *orderRepo) CreateWithOutbox(ctx context.Context, order *Order, event OutboxEvent) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    _, err = tx.ExecContext(ctx,
        `INSERT INTO orders (id, user_id, status, total) VALUES ($1, $2, $3, $4)`,
        order.ID, order.UserID, order.Status, order.Total)
    if err != nil {
        return err
    }

    _, err = tx.ExecContext(ctx,
        `INSERT INTO outbox (id, topic, key, payload, created_at)
         VALUES ($1, $2, $3, $4, $5)`,
        event.ID, event.Topic, event.Key, event.Payload, time.Now())
    if err != nil {
        return err
    }

    return tx.Commit()
}

// Outbox relay — отдельный процесс
type OutboxRelay struct {
    db       *sql.DB
    producer sarama.SyncProducer
    interval time.Duration
}

func (r *OutboxRelay) Run(ctx context.Context) {
    ticker := time.NewTicker(r.interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            r.processBatch(ctx)
        }
    }
}

func (r *OutboxRelay) processBatch(ctx context.Context) {
    tx, _ := r.db.BeginTx(ctx, nil)
    defer tx.Rollback()

    rows, _ := tx.QueryContext(ctx,
        `SELECT id, topic, key, payload FROM outbox
         WHERE sent_at IS NULL ORDER BY created_at LIMIT 100
         FOR UPDATE SKIP LOCKED`) // SKIP LOCKED для конкурентных relay
    defer rows.Close()

    var ids []string
    for rows.Next() {
        var evt OutboxEvent
        rows.Scan(&evt.ID, &evt.Topic, &evt.Key, &evt.Payload)

        _, _, err := r.producer.SendMessage(&sarama.ProducerMessage{
            Topic: evt.Topic,
            Key:   sarama.StringEncoder(evt.Key),
            Value: sarama.ByteEncoder(evt.Payload),
        })
        if err != nil {
            slog.Error("outbox relay send failed", "err", err)
            return // retry next tick
        }
        ids = append(ids, evt.ID)
    }

    if len(ids) > 0 {
        tx.ExecContext(ctx,
            `UPDATE outbox SET sent_at = NOW() WHERE id = ANY($1)`, pq.Array(ids))
        tx.Commit()
    }
}
```

## Inter-Service Communication

```
Synchronous (request-response):
  gRPC — для internal, high performance
  REST — для public API, простые случаи

Asynchronous (event-driven):
  Kafka — event streaming, high throughput
  RabbitMQ — task queue, routing

Выбор:
  Нужен ответ сейчас → sync (gRPC/REST)
  Fire and forget → async (Kafka)
  Нужна гарантия доставки → async + outbox
  Real-time → WebSocket / gRPC streaming

Anti-pattern: synchronous chain
  A → B → C → D (каждый ждёт следующего)
  Latency = sum всех, availability = product всех
  Решение: async где возможно, CQRS для чтения
```

## Частые вопросы

**Q: Монолит vs микросервисы?**
A: Начинай с монолита. Микросервисы — когда: разные команды, разные скорости деплоя, разные требования к масштабированию. "Monolith first" — Martin Fowler.

**Q: Как определить границы сервисов?**
A: По bounded contexts (DDD). Один сервис = один bounded context. Критерий: команда может деплоить независимо, минимум sync communication с другими.

**Q: Distributed transactions — как?**
A: Saga pattern (не 2PC). Outbox pattern для гарантии публикации событий. Idempotent consumers для at-least-once delivery.
