# Distributed Systems

## CAP теорема

```
Можно выбрать только 2 из 3:

  C — Consistency (все узлы видят одни данные одновременно)
  A — Availability (каждый запрос получает ответ)
  P — Partition Tolerance (система работает при разрыве сети)

P — всегда нужно (сеть БУДЕТ разрываться)
→ Реальный выбор: CP или AP

CP (Consistency + Partition Tolerance):
  - При partition: отказать в обслуживании (вернуть ошибку)
  - Примеры: ZooKeeper, etcd, HBase, MongoDB (default)
  - Когда: банковские транзакции, inventory, distributed locks

AP (Availability + Partition Tolerance):
  - При partition: вернуть данные (возможно устаревшие)
  - Примеры: Cassandra, DynamoDB, CouchDB
  - Когда: social feed, analytics, рекомендации
```

### PACELC (расширение CAP)

```
Если Partition → выбор A или C
Else (нормальная работа) → выбор Latency или Consistency

PA/EL — Cassandra, DynamoDB (availability + low latency)
PC/EC — ZooKeeper, etcd (consistency всегда)
PA/EC — MongoDB (available при partition, consistent обычно)
```

## Модели консистентности

```
Strong Consistency:
  - Чтение всегда возвращает последнюю запись
  - Linearizability — самая сильная гарантия
  - Медленно (нужен консенсус)
  - Пример: etcd, Spanner

Eventual Consistency:
  - Данные КОГДА-НИБУДЬ станут консистентными
  - Быстро, но читать можно stale данные
  - Пример: DNS, DynamoDB (default)

Causal Consistency:
  - Если A → B (причинно), то все увидят A перед B
  - Нет гарантий для несвязанных операций
  - Пример: MongoDB (causal sessions)

Read-your-writes:
  - Пользователь видит свои собственные записи
  - Остальные могут видеть stale
  - Реализация: read from leader / sticky sessions
```

## Консенсус

### Raft (etcd, Consul)

```
Роли: Leader, Follower, Candidate
Гарантия: majority (N/2 + 1) узлов согласны

Выбор лидера (Leader Election):
  1. Follower не получает heartbeat → становится Candidate
  2. Candidate голосует за себя, запрашивает голоса
  3. Получает majority → становится Leader
  4. Leader шлёт heartbeats всем Followers

Репликация лога:
  1. Client → Leader: запись
  2. Leader → Followers: AppendEntries RPC
  3. Majority подтвердило → commit
  4. Leader → Client: успех

Term (эпоха):
  - Монотонно растущий номер
  - Новые выборы → новый term
  - Stale leader (старый term) → отвергается

Пример: etcd в Kubernetes хранит состояние кластера
  - 3 или 5 узлов (нечётное!)
  - Выдерживает падение: (N-1)/2 узлов
  - 3 узла → 1 падение, 5 узлов → 2 падения
```

## Распределённые транзакции

### Two-Phase Commit (2PC)

```
Coordinator → Participants

Phase 1 (Prepare):
  Coordinator: "Можете закоммитить?"
  Participant A: "Да (PREPARED)"
  Participant B: "Да (PREPARED)"

Phase 2 (Commit):
  Coordinator: "Коммитьте!"
  Participant A: COMMITTED
  Participant B: COMMITTED

Проблемы:
  - Blocking: если Coordinator падает после Phase 1 → участники ждут вечно
  - Single point of failure: Coordinator
  - Медленно: 2 round-trips + disk writes
```

### Saga Pattern

```
Вместо распределённой транзакции — цепочка локальных транзакций
Каждый шаг имеет компенсирующее действие (откат)

Пример: Создание заказа
  1. Order Service: создать заказ (compensate: отменить заказ)
  2. Payment Service: списать деньги (compensate: вернуть деньги)
  3. Inventory Service: зарезервировать товар (compensate: освободить)
  4. Shipping Service: создать доставку (compensate: отменить)

  Если шаг 3 упал → компенсация: вернуть деньги → отменить заказ

Orchestration Saga:
  Центральный оркестратор управляет шагами
  + Простая логика, видна вся цепочка
  - Single point of failure

Choreography Saga:
  Каждый сервис слушает события и реагирует
  + Нет центральной точки
  - Сложно отслеживать, cyclic dependencies
```

## Распределённые ID

```
UUID v4:
  + Генерируется локально, нет координации
  - 128 бит, не сортируемый, плохо для B-Tree индексов

UUID v7 (Go 1.24+ в crypto/rand?):
  + Timestamp-sorted, лучше для БД

Snowflake (Twitter):
  [1 бит unused][41 бит timestamp][10 бит machine ID][12 бит sequence]
  + 64 бита, сортируемый по времени
  + ~4096 ID/ms на машину
  - Нужна координация machine ID

ULID:
  [48 бит timestamp][80 бит random]
  + 128 бит, сортируемый, совместим с UUID

Для SD интервью:
  - Малый масштаб → UUID v4/v7
  - Twitter/Instagram scale → Snowflake
  - Auto-increment → только single DB
```

## Service Discovery

```
Client-side (Consul, etcd):
  Client → Service Registry: "где Payment Service?"
  Registry → Client: ["10.0.1.5:8080", "10.0.1.6:8080"]
  Client → Payment Service (с load balancing)

Server-side (Kubernetes, AWS ALB):
  Client → Load Balancer: запрос
  LB → Service Registry: "где Payment Service?"
  LB → Payment Service

DNS-based (простейший):
  payment.service.internal → A records с IP адресами
  + Простота
  - DNS caching, медленное обновление

Kubernetes:
  Service → ClusterIP → Pod endpoints
  payment-service.default.svc.cluster.local
```

## Distributed Locking

```go
// Redis (Redlock algorithm)
// Простой lock:
SET resource_name my_random_value NX PX 30000
// NX = set if not exists, PX = expire in 30s

// Unlock (Lua script для атомарности):
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("del", KEYS[1])
end

// Redlock (для Redis cluster):
// 1. Получить timestamp
// 2. Попытаться получить lock на N/2+1 узлах
// 3. Lock валиден если: получен на majority + время не истекло
// 4. Если не получилось → unlock на всех узлах
```

```go
// etcd (рекомендуется для production)
import clientv3 "go.etcd.io/etcd/client/v3"
import "go.etcd.io/etcd/client/v3/concurrency"

cli, _ := clientv3.New(clientv3.Config{Endpoints: endpoints})
session, _ := concurrency.NewSession(cli, concurrency.WithTTL(10))
defer session.Close()

mutex := concurrency.NewMutex(session, "/locks/my-resource")
if err := mutex.Lock(ctx); err != nil {
    log.Fatal(err)
}
defer mutex.Unlock(ctx)
// критическая секция
```

## Часы и время

```
Проблема: часы на разных серверах не синхронизированы
NTP точность: 10-100ms (ненадёжно для ordering)

Lamport Clocks:
  - Логические часы (counter)
  - Send: counter++, attach counter
  - Receive: counter = max(local, received) + 1
  - Partial order (не определяет concurrent events)

Vector Clocks:
  - Каждый узел хранит вектор [N1:3, N2:5, N3:1]
  - Определяет causality и concurrent events
  - Размер растёт с кол-вом узлов

Hybrid Logical Clocks (HLC):
  - Physical timestamp + logical counter
  - Используется в CockroachDB, YugabyteDB
```

## Частые вопросы

**Q: Чем CAP теорема полезна на практике?**
A: Помогает понять trade-offs при выборе БД и дизайне системы. На SD интервью — объяснить, почему выбрал CP или AP систему для конкретного случая.

**Q: Почему Raft использует нечётное число узлов?**
A: Для majority. 3 узла: majority=2, выдерживает 1 падение. 4 узла: majority=3, тоже выдерживает 1. Четвёртый узел не даёт преимущества, но добавляет latency.

**Q: Saga vs 2PC?**
A: 2PC — строгая консистентность, но blocking и медленно. Saga — eventual consistency, но resilient и масштабируемо. Microservices → почти всегда Saga.

**Q: Как выбрать между strong и eventual consistency?**
A: Деньги, inventory, авторизация → strong. Feed, рекомендации, счётчики просмотров → eventual.
