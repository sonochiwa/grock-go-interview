# Clean Architecture в Go

## Принцип

```
Зависимости направлены ВНУТРЬ (от внешних слоёв к внутренним)
Внутренние слои НЕ знают о внешних

  ┌──────────────────────────────────────────┐
  │  Frameworks & Drivers (HTTP, gRPC, DB)   │  ← внешний слой
  │  ┌──────────────────────────────────────┐│
  │  │  Adapters (handlers, repositories)   ││
  │  │  ┌──────────────────────────────────┐││
  │  │  │  Use Cases (бизнес-логика)       │││
  │  │  │  ┌──────────────────────────────┐│││
  │  │  │  │  Entities (доменные модели)  ││││  ← ядро
  │  │  │  └──────────────────────────────┘│││
  │  │  └──────────────────────────────────┘││
  │  └──────────────────────────────────────┘│
  └──────────────────────────────────────────┘
```

## Структура проекта

```
order-service/
├── cmd/
│   └── server/
│       └── main.go              ← точка входа, DI, запуск
├── internal/
│   ├── domain/                  ← ЯДРО (нет зависимостей!)
│   │   ├── order.go             ← entity + бизнес-правила
│   │   ├── errors.go            ← domain errors
│   │   └── repository.go        ← interface repository
│   ├── usecase/                 ← USE CASES (зависит только от domain)
│   │   ├── create_order.go
│   │   ├── get_order.go
│   │   └── cancel_order.go
│   ├── adapter/                 ← АДАПТЕРЫ (реализации интерфейсов)
│   │   ├── postgres/
│   │   │   └── order_repo.go    ← реализация domain.OrderRepository
│   │   ├── redis/
│   │   │   └── cache.go
│   │   └── kafka/
│   │       └── publisher.go
│   └── port/                    ← ПОРТЫ (входные адаптеры)
│       ├── http/
│       │   ├── handler.go
│       │   ├── middleware.go
│       │   └── router.go
│       └── grpc/
│           └── server.go
├── pkg/                         ← переиспользуемые пакеты
├── migrations/
└── proto/
```

## Domain Layer (ядро)

```go
// internal/domain/order.go
package domain

type OrderStatus string

const (
    OrderStatusPending   OrderStatus = "pending"
    OrderStatusConfirmed OrderStatus = "confirmed"
    OrderStatusCancelled OrderStatus = "cancelled"
)

type Order struct {
    ID        string
    UserID    string
    Items     []OrderItem
    Status    OrderStatus
    Total     int // в копейках
    CreatedAt time.Time
}

type OrderItem struct {
    ProductID string
    Quantity  int
    Price     int
}

// Бизнес-логика В entity, не в usecase
func (o *Order) Cancel() error {
    if o.Status == OrderStatusCancelled {
        return ErrAlreadyCancelled
    }
    if o.Status == OrderStatusConfirmed {
        return ErrCannotCancelConfirmed
    }
    o.Status = OrderStatusCancelled
    return nil
}

func (o *Order) CalculateTotal() {
    total := 0
    for _, item := range o.Items {
        total += item.Price * item.Quantity
    }
    o.Total = total
}

// internal/domain/repository.go
type OrderRepository interface {
    Get(ctx context.Context, id string) (*Order, error)
    Create(ctx context.Context, order *Order) error
    Update(ctx context.Context, order *Order) error
    List(ctx context.Context, userID string, limit, offset int) ([]*Order, error)
}

// internal/domain/errors.go
var (
    ErrOrderNotFound        = errors.New("order not found")
    ErrAlreadyCancelled     = errors.New("order already cancelled")
    ErrCannotCancelConfirmed = errors.New("cannot cancel confirmed order")
)
```

## Use Case Layer

```go
// internal/usecase/create_order.go
package usecase

type CreateOrderInput struct {
    UserID string
    Items  []domain.OrderItem
}

type CreateOrderUseCase struct {
    repo      domain.OrderRepository
    publisher EventPublisher
}

func NewCreateOrderUseCase(repo domain.OrderRepository, pub EventPublisher) *CreateOrderUseCase {
    return &CreateOrderUseCase{repo: repo, publisher: pub}
}

func (uc *CreateOrderUseCase) Execute(ctx context.Context, input CreateOrderInput) (*domain.Order, error) {
    order := &domain.Order{
        ID:        uuid.NewString(),
        UserID:    input.UserID,
        Items:     input.Items,
        Status:    domain.OrderStatusPending,
        CreatedAt: time.Now(),
    }
    order.CalculateTotal()

    if err := uc.repo.Create(ctx, order); err != nil {
        return nil, fmt.Errorf("create order: %w", err)
    }

    // Публикация события (через интерфейс)
    uc.publisher.Publish(ctx, Event{
        Type:    "OrderCreated",
        Payload: order,
    })

    return order, nil
}
```

## Adapter Layer

```go
// internal/adapter/postgres/order_repo.go
package postgres

type orderRepo struct {
    db *sql.DB
}

func NewOrderRepository(db *sql.DB) domain.OrderRepository {
    return &orderRepo{db: db}
}

func (r *orderRepo) Get(ctx context.Context, id string) (*domain.Order, error) {
    row := r.db.QueryRowContext(ctx,
        `SELECT id, user_id, status, total, created_at FROM orders WHERE id = $1`, id)

    var order domain.Order
    err := row.Scan(&order.ID, &order.UserID, &order.Status, &order.Total, &order.CreatedAt)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, domain.ErrOrderNotFound
    }
    return &order, err
}

func (r *orderRepo) Create(ctx context.Context, order *domain.Order) error {
    _, err := r.db.ExecContext(ctx,
        `INSERT INTO orders (id, user_id, status, total, created_at)
         VALUES ($1, $2, $3, $4, $5)`,
        order.ID, order.UserID, order.Status, order.Total, order.CreatedAt,
    )
    return err
}
```

## Port Layer (входной адаптер)

```go
// internal/port/http/handler.go
type OrderHandler struct {
    createOrder *usecase.CreateOrderUseCase
    getOrder    *usecase.GetOrderUseCase
}

func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req CreateOrderRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, http.StatusBadRequest, "invalid request")
        return
    }

    order, err := h.createOrder.Execute(r.Context(), usecase.CreateOrderInput{
        UserID: req.UserID,
        Items:  toDomainItems(req.Items),
    })
    if err != nil {
        // Map domain errors to HTTP
        switch {
        case errors.Is(err, domain.ErrOrderNotFound):
            respondError(w, http.StatusNotFound, err.Error())
        default:
            respondError(w, http.StatusInternalServerError, "internal error")
        }
        return
    }

    respondJSON(w, http.StatusCreated, toResponse(order))
}
```

## DI (Dependency Injection) в main

```go
// cmd/server/main.go
func main() {
    cfg := config.Load()

    // Infrastructure
    db, _ := sql.Open("postgres", cfg.Database.DSN)
    redisClient := redis.NewClient(cfg.Redis)
    kafkaProducer, _ := sarama.NewSyncProducer(cfg.Kafka.Brokers, kafkaConfig)

    // Adapters (реализации интерфейсов)
    orderRepo := postgres.NewOrderRepository(db)
    eventPub := kafkaadapter.NewPublisher(kafkaProducer)
    cache := redisadapter.NewCache(redisClient)

    // Use Cases
    createOrder := usecase.NewCreateOrderUseCase(orderRepo, eventPub)
    getOrder := usecase.NewGetOrderUseCase(orderRepo, cache)

    // Ports (HTTP handlers)
    handler := httpport.NewOrderHandler(createOrder, getOrder)
    router := httpport.NewRouter(handler)

    // Start server
    srv := &http.Server{Addr: ":8080", Handler: router}
    srv.ListenAndServe()
}
```

## Правила

```
1. domain/ — НОЛЬ внешних зависимостей (только stdlib)
2. usecase/ — зависит только от domain/ (через interfaces)
3. adapter/ — реализует interfaces из domain/
4. port/ — вызывает usecase/, маппит DTO ↔ domain

Тестирование:
  - domain: unit тесты (чистая логика)
  - usecase: unit тесты с fakes/mocks для repository
  - adapter: integration тесты (testcontainers)
  - port: HTTP тесты (httptest)
```
