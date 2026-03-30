# Data Patterns

## CQRS (Command Query Responsibility Segregation)

```
Разделение модели на запись (Command) и чтение (Query)

Без CQRS:
  [Client] → [API] → [Service] → [Database]
  Одна модель для чтения и записи

С CQRS:
  [Client] → [Command API] → [Write Model] → [Write DB (PostgreSQL)]
                                    ↓ events
  [Client] → [Query API]  → [Read Model]  → [Read DB (Elasticsearch/Redis)]

Зачем:
  - Разные оптимизации для read и write
  - Read model: денормализованные данные, быстрый поиск
  - Write model: нормализованные, бизнес-логика
  - Масштабирование чтения отдельно от записи

Когда:
  ✅ Read/write ratio > 10:1
  ✅ Сложные запросы на чтение (поиск, фильтры, агрегации)
  ✅ Разные хранилища для read и write

Когда НЕ:
  ❌ Простой CRUD
  ❌ Strong consistency обязательна
  ❌ Маленький проект (overhead не оправдан)
```

## Event Sourcing

```
Вместо хранения текущего состояния — хранить ВСЕ события

Традиционно:
  UPDATE accounts SET balance = 150 WHERE id = 1

Event Sourcing:
  Event 1: AccountCreated { id: 1, balance: 0 }
  Event 2: MoneyDeposited { id: 1, amount: 200 }
  Event 3: MoneyWithdrawn { id: 1, amount: 50 }
  → Текущее состояние = replay всех events = 150

Преимущества:
  + Полный audit log
  + Можно восстановить состояние на любой момент времени
  + Debug: "почему баланс такой?" → просмотреть события
  + Легко добавить новые read models (replay events)

Недостатки:
  - Event store растёт бесконечно → snapshots
  - Eventual consistency (read model обновляется асинхронно)
  - Schema evolution (события неизменяемы!)
  - Сложность разработки

Snapshot:
  Каждые N событий сохранять текущее состояние
  Восстановление: snapshot + events после snapshot
  Пример: snapshot каждые 100 событий
```

```go
// Пример Event Sourcing
type Event struct {
    ID        string
    Type      string
    Payload   json.RawMessage
    Timestamp time.Time
    Version   int
}

type Account struct {
    ID      string
    Balance int
    Version int
}

func (a *Account) Apply(event Event) {
    switch event.Type {
    case "MoneyDeposited":
        var e MoneyDeposited
        json.Unmarshal(event.Payload, &e)
        a.Balance += e.Amount
    case "MoneyWithdrawn":
        var e MoneyWithdrawn
        json.Unmarshal(event.Payload, &e)
        a.Balance -= e.Amount
    }
    a.Version = event.Version
}

// Восстановление из событий
func LoadAccount(events []Event) *Account {
    a := &Account{}
    for _, e := range events {
        a.Apply(e)
    }
    return a
}
```

## Transactional Outbox

```
Проблема: нужно записать в БД И отправить событие в Kafka
  Двойная запись → inconsistency

  1. DB commit OK + Kafka fail → БД обновлена, событие потеряно
  2. Kafka OK + DB fail → событие отправлено, БД не обновлена

Решение — Outbox Pattern:

  BEGIN TRANSACTION;
    INSERT INTO orders (id, ...) VALUES (...);
    INSERT INTO outbox (id, topic, payload) VALUES (...);
  COMMIT;

  [Outbox Relay] → читает outbox таблицу → публикует в Kafka → помечает sent

  Варианты relay:
    1. Polling: SELECT * FROM outbox WHERE sent = false (простой, задержка)
    2. CDC: Debezium читает WAL → Kafka (real-time, без polling)

  Таблица outbox:
    id          UUID PRIMARY KEY
    topic       TEXT
    key         TEXT
    payload     JSONB
    created_at  TIMESTAMP
    sent_at     TIMESTAMP NULL
```

## Saga Pattern (подробнее)

```
Координация распределённых транзакций без 2PC

Пример: E-commerce заказ
  Сервисы: Order, Payment, Inventory, Shipping

Orchestration Saga:
  [Order Saga Orchestrator]
    1. → Order Service: CreateOrder
    2. → Payment Service: ProcessPayment
    3. → Inventory Service: ReserveStock
    4. → Shipping Service: CreateShipment

    Если шаг 3 упал:
    C2. → Payment Service: RefundPayment (compensate)
    C1. → Order Service: CancelOrder (compensate)

Choreography Saga:
  Order Service: OrderCreated event →
  Payment Service: (listens) → PaymentProcessed event →
  Inventory Service: (listens) → StockReserved event →
  Shipping Service: (listens) → ShipmentCreated event

  Если Inventory Service упал:
  Inventory: StockReservationFailed event →
  Payment: (listens) → PaymentRefunded event →
  Order: (listens) → OrderCancelled event

Orchestration vs Choreography:
  | | Orchestration | Choreography |
  |---|---|---|
  | Сложность | Проще (централизованно) | Сложнее (распределённо) |
  | Coupling | К оркестратору | Между сервисами |
  | Видимость | Весь flow в одном месте | Flow распределён |
  | Масштаб | Оркестратор = bottleneck | Масштабируется лучше |
  | Рекомендация | 3-5 шагов | > 5 шагов |
```

## Change Data Capture (CDC)

```
Захват изменений из БД и публикация в event stream

Способы:
  1. Trigger-based: триггер на INSERT/UPDATE/DELETE → пишет в audit таблицу
     - Медленно, нагружает БД
  2. Query-based: периодический SELECT WHERE updated_at > last_check
     - Не ловит DELETE, задержка
  3. Log-based (рекомендуется): чтение WAL/binlog
     - Debezium → Kafka Connect → Kafka
     - Минимальная нагрузка на БД
     - Real-time

Debezium:
  PostgreSQL (WAL) / MySQL (binlog) → Debezium → Kafka

  Topic: dbserver.schema.table
  Формат сообщения:
  {
    "before": {"id": 1, "name": "old"},  // предыдущее состояние
    "after":  {"id": 1, "name": "new"},  // новое состояние
    "op": "u",                           // c=create, u=update, d=delete
    "ts_ms": 1234567890
  }

Use cases:
  - Синхронизация read model (CQRS)
  - Кэш инвалидация
  - Search index update (→ Elasticsearch)
  - Data warehouse sync
  - Cross-service data replication
```

## Event-Driven Architecture

```
Типы событий:

1. Domain Events:
   OrderCreated, PaymentProcessed, UserRegistered
   Описывают что произошло в бизнес-домене

2. Integration Events:
   Для коммуникации между сервисами
   Содержат минимум данных (ID + необходимое)

3. Notification Events (thin):
   "OrderCreated: {order_id: 123}"
   Consumer сам запрашивает детали (→ API call)
   + Маленькие сообщения
   - Дополнительные API calls

4. Event-Carried State Transfer (fat):
   "OrderCreated: {order_id: 123, items: [...], total: 500}"
   Consumer имеет всё необходимое
   + Нет дополнительных calls
   - Большие сообщения, coupling к структуре

Naming:
  ✅ Past tense: OrderCreated, PaymentFailed
  ❌ Imperative: CreateOrder, ProcessPayment (это commands!)
```

## Data Consistency Patterns

```
1. Dual Writes (антипаттерн!):
   Запись в DB + запись в Kafka/Cache/Search
   → Может разойтись при partial failure
   → Решение: Outbox Pattern или CDC

2. Eventual Consistency:
   Данные становятся консистентными через некоторое время
   UI: показать "processing..." или оптимистичное обновление

3. Compensation:
   Вместо rollback — выполнить обратное действие
   Saga pattern использует compensating transactions

4. Idempotent Consumer:
   Повторная обработка того же события = тот же результат
   Хранить processed_event_ids в БД
   INSERT ... ON CONFLICT DO NOTHING
```

## Частые вопросы

**Q: Event Sourcing vs Event-Driven Architecture?**
A: Event Sourcing — способ ХРАНЕНИЯ данных (все события). Event-Driven — способ КОММУНИКАЦИИ (через события). Можно использовать одно без другого.

**Q: Когда Outbox, когда CDC?**
A: Outbox — если контролируешь схему БД и нужна простота. CDC (Debezium) — если нельзя менять БД, или нужен CDC из legacy системы, или много таблиц.

**Q: CQRS без Event Sourcing — можно?**
A: Да! CQRS — это просто разделение read/write моделей. Event Sourcing — ортогональный паттерн. Часто используются вместе, но не обязательно.
