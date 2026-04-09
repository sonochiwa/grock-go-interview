# Гексагональная архитектура (Ports & Adapters)

## Идея

Автор — Alistair Cockburn (2005). Приложение — это ядро (домен + бизнес-логика), окружённое **портами** (интерфейсами) и **адаптерами** (реализациями). Внешний мир взаимодействует с ядром только через порты.

```
              Driving (входящие)                    Driven (исходящие)
          ┌─────────────────────┐              ┌─────────────────────┐
          │   HTTP Handler      │              │   PostgreSQL Repo   │
          │   gRPC Handler      │              │   Redis Cache       │
          │   CLI               │              │   Kafka Producer    │
          │   Cron Job          │              │   SMTP Sender       │
          └────────┬────────────┘              └────────▲────────────┘
                   │                                    │
              ═════▼════════════════════════════════════╪═══════
              ║    INPUT PORT                  OUTPUT PORT    ║
              ║    (interface)                 (interface)    ║
              ║         │                          ▲         ║
              ║         ▼                          │         ║
              ║    ┌──────────────────────────────────┐      ║
              ║    │           APPLICATION            │      ║
              ║    │     (use cases / services)       │      ║
              ║    │                                  │      ║
              ║    │    ┌────────────────────────┐    │      ║
              ║    │    │        DOMAIN           │    │      ║
              ║    │    │  entities, value objects │    │      ║
              ║    │    └────────────────────────┘    │      ║
              ║    └──────────────────────────────────┘      ║
              ═══════════════════════════════════════════════
```

## Ключевые концепции

### Порты

Порт — это **интерфейс**, определённый ядром приложения.

| Тип | Направление | Кто вызывает | Пример |
|-----|-------------|-------------|--------|
| **Input Port** (Driving) | Внешний мир → Ядро | HTTP handler вызывает use case | `OrderService` interface |
| **Output Port** (Driven) | Ядро → Внешний мир | Use case вызывает репозиторий | `OrderRepository` interface |

### Адаптеры

Адаптер — это **реализация** порта для конкретной технологии.

| Порт | Адаптер |
|------|---------|
| `OrderRepository` | `PostgresOrderRepo`, `InMemoryOrderRepo` |
| `NotificationSender` | `SMTPSender`, `SlackSender`, `MockSender` |
| `CacheStore` | `RedisCache`, `InMemoryCache` |
| `OrderService` (input) | `HTTPHandler`, `GRPCHandler`, `CLICommand` |

## Реализация в Go

### Структура проекта

```
order-service/
├── cmd/
│   └── server/
│       └── main.go                  ← DI, wiring адаптеров
├── internal/
│   ├── domain/                      ← ЯДРО (ноль зависимостей)
│   │   ├── order.go                 ← entity
│   │   └── errors.go                ← domain errors
│   ├── port/                        ← ПОРТЫ (интерфейсы)
│   │   ├── input.go                 ← input ports (use cases)
│   │   └── output.go                ← output ports (driven)
│   ├── app/                         ← APPLICATION (реализация input ports)
│   │   └── order_service.go         ← бизнес-логика
│   └── adapter/                     ← АДАПТЕРЫ (реализация output ports + driving)
│       ├── driving/                 ← входящие (HTTP, gRPC)
│       │   ├── http/
│       │   │   ├── handler.go
│       │   │   └── router.go
│       │   └── grpc/
│       │       └── handler.go
│       └── driven/                  ← исходящие (DB, cache, email)
│           ├── postgres/
│           │   └── order_repo.go
│           ├── redis/
│           │   └── cache.go
│           └── email/
│               └── smtp_sender.go
```

### Domain (ядро)

```go
// internal/domain/order.go
package domain

import "time"

type OrderStatus string

const (
    OrderPending   OrderStatus = "pending"
    OrderConfirmed OrderStatus = "confirmed"
    OrderCancelled OrderStatus = "cancelled"
    OrderShipped   OrderStatus = "shipped"
)

type Order struct {
    ID        string
    UserID    string
    Items     []OrderItem
    Status    OrderStatus
    Total     int64 // in cents
    CreatedAt time.Time
}

type OrderItem struct {
    ProductID string
    Quantity  int
    Price     int64
}

// Domain logic lives here
func (o *Order) Cancel() error {
    if o.Status == OrderShipped {
        return ErrCannotCancelShipped
    }
    o.Status = OrderCancelled
    return nil
}

func (o *Order) CalculateTotal() {
    var total int64
    for _, item := range o.Items {
        total += item.Price * int64(item.Quantity)
    }
    o.Total = total
}
```

### Ports (интерфейсы)

```go
// internal/port/input.go — input ports (что приложение УМЕЕТ делать)
package port

import (
    "context"
    "myapp/internal/domain"
)

// Input port — driving side calls this
type OrderService interface {
    CreateOrder(ctx context.Context, req CreateOrderRequest) (*domain.Order, error)
    GetOrder(ctx context.Context, id string) (*domain.Order, error)
    CancelOrder(ctx context.Context, id string) error
    ListUserOrders(ctx context.Context, userID string) ([]*domain.Order, error)
}

type CreateOrderRequest struct {
    UserID string
    Items  []OrderItemRequest
}

type OrderItemRequest struct {
    ProductID string
    Quantity  int
}
```

```go
// internal/port/output.go — output ports (что приложение ТРЕБУЕТ от внешнего мира)
package port

import (
    "context"
    "myapp/internal/domain"
)

// Output port — application calls this, adapters implement
type OrderRepository interface {
    Save(ctx context.Context, order *domain.Order) error
    FindByID(ctx context.Context, id string) (*domain.Order, error)
    FindByUserID(ctx context.Context, userID string) ([]*domain.Order, error)
}

type ProductCatalog interface {
    GetPrice(ctx context.Context, productID string) (int64, error)
}

type NotificationSender interface {
    SendOrderConfirmation(ctx context.Context, order *domain.Order) error
}

type CacheStore interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
}
```

### Application (реализация input port)

```go
// internal/app/order_service.go
package app

import (
    "context"
    "fmt"
    "time"

    "myapp/internal/domain"
    "myapp/internal/port"

    "github.com/google/uuid"
)

// Implements port.OrderService (input port)
type OrderService struct {
    repo     port.OrderRepository     // output port
    catalog  port.ProductCatalog      // output port
    notifier port.NotificationSender  // output port
}

func NewOrderService(
    repo port.OrderRepository,
    catalog port.ProductCatalog,
    notifier port.NotificationSender,
) *OrderService {
    return &OrderService{
        repo:     repo,
        catalog:  catalog,
        notifier: notifier,
    }
}

func (s *OrderService) CreateOrder(ctx context.Context, req port.CreateOrderRequest) (*domain.Order, error) {
    order := &domain.Order{
        ID:        uuid.New().String(),
        UserID:    req.UserID,
        Status:    domain.OrderPending,
        CreatedAt: time.Now(),
    }

    // Fetch prices from catalog (through output port)
    for _, item := range req.Items {
        price, err := s.catalog.GetPrice(ctx, item.ProductID)
        if err != nil {
            return nil, fmt.Errorf("get price for %s: %w", item.ProductID, err)
        }
        order.Items = append(order.Items, domain.OrderItem{
            ProductID: item.ProductID,
            Quantity:  item.Quantity,
            Price:     price,
        })
    }

    // Domain logic
    order.CalculateTotal()

    // Persist (through output port)
    if err := s.repo.Save(ctx, order); err != nil {
        return nil, fmt.Errorf("save order: %w", err)
    }

    // Notify (through output port, fire-and-forget)
    go s.notifier.SendOrderConfirmation(context.Background(), order)

    return order, nil
}

func (s *OrderService) CancelOrder(ctx context.Context, id string) error {
    order, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return err
    }

    // Domain logic — order decides if it can be cancelled
    if err := order.Cancel(); err != nil {
        return err
    }

    return s.repo.Save(ctx, order)
}
```

### Driving Adapter (HTTP)

```go
// internal/adapter/driving/http/handler.go
package http

import (
    "encoding/json"
    "net/http"

    "myapp/internal/port"
)

// Driving adapter — calls input port
type OrderHandler struct {
    service port.OrderService // depends on INPUT PORT, not implementation
}

func NewOrderHandler(service port.OrderService) *OrderHandler {
    return &OrderHandler{service: service}
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
    var req port.CreateOrderRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }

    order, err := h.service.CreateOrder(r.Context(), req)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(order)
}

func (h *OrderHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")

    if err := h.service.CancelOrder(r.Context(), id); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    w.WriteHeader(http.StatusNoContent)
}
```

### Driven Adapter (PostgreSQL)

```go
// internal/adapter/driven/postgres/order_repo.go
package postgres

import (
    "context"

    "myapp/internal/domain"

    "github.com/jackc/pgx/v5/pgxpool"
)

// Driven adapter — implements output port
type OrderRepo struct {
    pool *pgxpool.Pool
}

func NewOrderRepo(pool *pgxpool.Pool) *OrderRepo {
    return &OrderRepo{pool: pool}
}

func (r *OrderRepo) Save(ctx context.Context, order *domain.Order) error {
    _, err := r.pool.Exec(ctx,
        `INSERT INTO orders (id, user_id, status, total, created_at)
         VALUES ($1, $2, $3, $4, $5)
         ON CONFLICT (id) DO UPDATE SET status = $3, total = $4`,
        order.ID, order.UserID, order.Status, order.Total, order.CreatedAt,
    )
    return err
}

func (r *OrderRepo) FindByID(ctx context.Context, id string) (*domain.Order, error) {
    row := r.pool.QueryRow(ctx,
        `SELECT id, user_id, status, total, created_at FROM orders WHERE id = $1`, id)

    var o domain.Order
    err := row.Scan(&o.ID, &o.UserID, &o.Status, &o.Total, &o.CreatedAt)
    if err != nil {
        return nil, err
    }
    return &o, nil
}
```

### Wiring (main.go)

```go
// cmd/server/main.go
package main

import (
    "log"
    "net/http"

    "myapp/internal/app"
    httpAdapter "myapp/internal/adapter/driving/http"
    "myapp/internal/adapter/driven/postgres"
    "myapp/internal/adapter/driven/email"
    "myapp/internal/adapter/driven/catalog"

    "github.com/jackc/pgx/v5/pgxpool"
)

func main() {
    pool, _ := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))

    // Driven adapters (output ports)
    orderRepo := postgres.NewOrderRepo(pool)
    productCatalog := catalog.NewHTTPCatalog("http://catalog-service:8080")
    notifier := email.NewSMTPSender("smtp://...")

    // Application (implements input port, uses output ports)
    orderService := app.NewOrderService(orderRepo, productCatalog, notifier)

    // Driving adapter (calls input port)
    handler := httpAdapter.NewOrderHandler(orderService)

    mux := http.NewServeMux()
    mux.HandleFunc("POST /orders", handler.CreateOrder)
    mux.HandleFunc("DELETE /orders/{id}", handler.CancelOrder)

    log.Fatal(http.ListenAndServe(":8080", mux))
}
```

## Hexagonal vs Clean Architecture

| | Гексагональная | Clean Architecture |
|---|---|---|
| **Автор** | Alistair Cockburn (2005) | Robert Martin (2012) |
| **Фокус** | Порты и адаптеры (снаружи внутрь) | Слои и правило зависимостей (круги) |
| **Слои** | Domain, Application, Adapters | Entities, Use Cases, Adapters, Frameworks |
| **Порты** | Явные input/output порты | Интерфейсы на границах слоёв |
| **Адаптеры** | Driving (входящие) + Driven (исходящие) | Presenters, Controllers, Gateways |
| **Тестируемость** | Одинаковая — подставляй mock-адаптеры |
| **На практике** | Фактически одно и то же с разной терминологией |

> На собеседовании: это **одна идея** с разных ракурсов. Hexagonal акцентирует порты/адаптеры и симметрию (driving = driven). Clean Architecture — строгие круги и правило зависимостей. В Go-проекте реализация практически идентична.

### Главное отличие в структуре

```
Clean Architecture:            Hexagonal:
internal/                      internal/
├── domain/                    ├── domain/
├── usecase/                   ├── port/          ← явные порты
├── adapter/                   ├── app/           ← реализация input ports
│   ├── postgres/              └── adapter/
│   └── http/                      ├── driving/   ← входящие (HTTP, gRPC)
                                   └── driven/    ← исходящие (DB, cache)
```

## Тестирование

Главное преимущество — замена адаптеров на mock/fake без изменения бизнес-логики.

```go
func TestCreateOrder(t *testing.T) {
    // Fake driven adapters
    repo := &fakeOrderRepo{}
    catalog := &fakeCatalog{prices: map[string]int64{"prod-1": 1000}}
    notifier := &fakeNotifier{}

    // Real application logic
    svc := app.NewOrderService(repo, catalog, notifier)

    order, err := svc.CreateOrder(context.Background(), port.CreateOrderRequest{
        UserID: "user-1",
        Items:  []port.OrderItemRequest{{ProductID: "prod-1", Quantity: 2}},
    })

    require.NoError(t, err)
    assert.Equal(t, int64(2000), order.Total)
    assert.Equal(t, domain.OrderPending, order.Status)
    assert.Len(t, repo.saved, 1)
}
```

## Частые вопросы на собеседовании

1. **Чем Hexagonal отличается от Clean Architecture?** — Терминологией. Суть одна: домен в центре, зависимости направлены внутрь, внешний мир через интерфейсы. Hexagonal явно разделяет driving/driven адаптеры.

2. **Что такое порт?** — Интерфейс в Go. Input port — что приложение умеет (use case). Output port — что приложению нужно от внешнего мира (repo, cache).

3. **Зачем отделять port/ от domain/?** — `domain/` — чистые бизнес-сущности и правила. `port/` — контракты взаимодействия (DTO, интерфейсы). Домен не должен знать о портах, но порты знают о домене.

4. **Не слишком ли много абстракций для Go?** — Для маленького CRUD — да, overkill. Для сервиса с 3+ адаптерами (HTTP + gRPC, Postgres + Redis, email + Slack) — окупается тестируемостью и заменяемостью.

5. **Когда НЕ нужна гексагональная архитектура?** — Простой CRUD, прототипы, утилиты, скрипты. Если единственный input — HTTP и единственный output — одна БД, flat-структура проще.
