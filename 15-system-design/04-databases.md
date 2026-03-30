# Базы данных

## SQL vs NoSQL

| | SQL (PostgreSQL, MySQL) | NoSQL |
|---|---|---|
| Модель | Таблицы, строки, отношения | Document, Key-Value, Column, Graph |
| Schema | Строгая schema | Schema-less / flexible |
| Транзакции | ACID | Обычно eventual consistency |
| Масштабирование | Вертикальное (+ read replicas) | Горизонтальное (шардинг из коробки) |
| Joins | Нативные | Нет / дорогие |
| Когда | Сложные отношения, ACID нужен | Высокая нагрузка, простые запросы |

### Типы NoSQL

```
Key-Value:   Redis, DynamoDB         → кэш, сессии, счётчики
Document:    MongoDB, CouchDB        → каталог товаров, профили
Column:      Cassandra, ScyllaDB     → time-series, логи, аналитика
Graph:       Neo4j, DGraph           → социальные графы, рекомендации
```

## Индексы

```sql
-- B-Tree (default) — для = < > BETWEEN ORDER BY
CREATE INDEX idx_users_email ON users(email);

-- Hash — только для = (быстрее B-Tree для equality)
CREATE INDEX idx_users_id ON users USING hash(id);

-- Composite — для запросов с несколькими полями
CREATE INDEX idx_orders_user_date ON orders(user_id, created_at);
-- Работает для: WHERE user_id = X AND created_at > Y
-- НЕ работает для: WHERE created_at > Y (без user_id)
-- Правило: leftmost prefix

-- Covering index — все данные в индексе, не нужно читать таблицу
CREATE INDEX idx_covering ON orders(user_id, status, total);
-- SELECT status, total FROM orders WHERE user_id = X
-- Index-only scan — быстро!
```

### Когда индекс мешает

- Частые INSERT/UPDATE (индексы обновляются)
- Маленькие таблицы (full scan дешевле)
- Низкая selectivity (bool поле — 50% таблицы)

## Репликация

```
Master-Slave (Primary-Replica):
  [Master] ──write──→ [Replica 1] (read)
     │                [Replica 2] (read)
     │                [Replica 3] (read)
     └── все writes идут в master

  + Read масштабирование
  + Failover (replica → master)
  - Replication lag (eventual consistency)
  - Single point of write

Master-Master (Multi-Primary):
  [Master 1] ←───→ [Master 2]
  - Conflict resolution сложный
  - Используется для geo-distribution
```

### Replication lag

```
Sync replication:   Master ждёт подтверждения от replica → медленно, но consistent
Async replication:  Master не ждёт → быстро, но может потерять данные
Semi-sync:          Ждёт хотя бы 1 replica → компромисс
```

## Шардинг (Partitioning)

```
Horizontal sharding — разделение СТРОК:
  Shard 1: user_id 1-1M
  Shard 2: user_id 1M-2M
  Shard 3: user_id 2M-3M

Vertical sharding — разделение СТОЛБЦОВ/ТАБЛИЦ:
  DB1: users, auth       (user service)
  DB2: orders, payments  (order service)
  DB3: products, catalog (product service)
```

### Стратегии шардинга

```
Hash-based:
  shard = hash(user_id) % N
  + Равномерное распределение
  - Resharding при добавлении шардов (всё пересчитывается)

Range-based:
  shard 1: A-M, shard 2: N-Z
  + Простота, range queries
  - Неравномерность (hot spots)

Directory-based:
  Lookup table: user_id → shard
  + Гибкость
  - Single point of failure (directory)

Consistent Hashing:
  Хеш-кольцо, виртуальные узлы
  + Минимальное перераспределение при добавлении шарда
  + Равномерность с виртуальными узлами
  - Сложнее реализовать
```

### Проблемы шардинга

1. **Cross-shard joins** — дорого или невозможно
2. **Cross-shard transactions** — нужен 2PC или Saga
3. **Hotspots** — один шард получает больше нагрузки
4. **Resharding** — перебалансировка при росте
5. **Referential integrity** — foreign keys не работают между шардами

## Connection Pooling

```go
// Go: database/sql имеет встроенный connection pool
db, _ := sql.Open("postgres", dsn)
db.SetMaxOpenConns(25)       // макс открытых соединений
db.SetMaxIdleConns(10)       // макс idle соединений
db.SetConnMaxLifetime(5*time.Minute) // макс время жизни

// Формула: MaxOpenConns ≈ (CPU cores * 2) + effective_spindle_count
// Для SSD: 25-50 connections обычно достаточно
// Больше != лучше — too many connections = contention на БД
```

## Частые вопросы

**Q: Когда шардить?**
A: Когда вертикальное масштабирование + read replicas + кэш недостаточны. Обычно >1TB данных или >10K write QPS.

**Q: SQL или NoSQL?**
A: SQL по умолчанию. NoSQL когда: нет сложных отношений, нужна горизонтальная масштабируемость, flexible schema, high write throughput.

**Q: Как выбрать shard key?**
A: Высокая cardinality + равномерное распределение + покрывает основные запросы. Для user-facing: часто user_id.
