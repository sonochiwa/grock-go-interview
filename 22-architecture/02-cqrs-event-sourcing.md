# CQRS и Event Sourcing в Go

## CQRS Implementation

```go
// Command side (write)
type CreateOrderCommand struct {
    UserID string
    Items  []OrderItem
}

type CommandHandler interface {
    Handle(ctx context.Context, cmd any) error
}

type createOrderHandler struct {
    repo      WriteOrderRepository
    publisher EventPublisher
}

func (h *createOrderHandler) Handle(ctx context.Context, cmd any) error {
    c := cmd.(CreateOrderCommand)
    order := domain.NewOrder(c.UserID, c.Items)
    if err := h.repo.Save(ctx, order); err != nil {
        return err
    }
    return h.publisher.Publish(ctx, OrderCreatedEvent{
        OrderID: order.ID,
        UserID:  order.UserID,
        Total:   order.Total,
    })
}

// Query side (read)
type GetOrderQuery struct {
    OrderID string
}

type OrderReadModel struct {
    ID         string    `json:"id"`
    UserID     string    `json:"user_id"`
    UserName   string    `json:"user_name"`   // денормализовано!
    Items      []Item    `json:"items"`
    ItemCount  int       `json:"item_count"`  // pre-computed
    Total      int       `json:"total"`
    Status     string    `json:"status"`
    CreatedAt  time.Time `json:"created_at"`
}

type QueryHandler interface {
    Handle(ctx context.Context, query any) (any, error)
}

type getOrderHandler struct {
    readRepo ReadOrderRepository // отдельное хранилище для чтения
}

func (h *getOrderHandler) Handle(ctx context.Context, query any) (any, error) {
    q := query.(GetOrderQuery)
    return h.readRepo.GetByID(ctx, q.OrderID)
}

// Read model updater (projection)
type OrderProjection struct {
    readRepo ReadOrderRepository
}

func (p *OrderProjection) HandleEvent(ctx context.Context, event Event) error {
    switch e := event.(type) {
    case OrderCreatedEvent:
        return p.readRepo.Insert(ctx, &OrderReadModel{
            ID:     e.OrderID,
            UserID: e.UserID,
            Total:  e.Total,
            Status: "pending",
        })
    case OrderCancelledEvent:
        return p.readRepo.UpdateStatus(ctx, e.OrderID, "cancelled")
    }
    return nil
}
```

## Event Sourcing в Go

```go
// Event Store
type EventStore interface {
    // Append events для aggregate
    Append(ctx context.Context, aggregateID string, expectedVersion int, events []Event) error
    // Load все events для aggregate
    Load(ctx context.Context, aggregateID string) ([]Event, error)
    // Load с определённой версии (для snapshot)
    LoadFrom(ctx context.Context, aggregateID string, fromVersion int) ([]Event, error)
}

type Event struct {
    ID            string
    AggregateID   string
    AggregateType string
    Type          string
    Version       int
    Data          json.RawMessage
    Timestamp     time.Time
}

// PostgreSQL event store
type pgEventStore struct {
    db *sql.DB
}

func (s *pgEventStore) Append(ctx context.Context, aggID string, expectedVersion int, events []Event) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Optimistic concurrency control
    var currentVersion int
    err = tx.QueryRowContext(ctx,
        `SELECT COALESCE(MAX(version), 0) FROM events WHERE aggregate_id = $1`, aggID,
    ).Scan(&currentVersion)
    if err != nil {
        return err
    }
    if currentVersion != expectedVersion {
        return fmt.Errorf("concurrency conflict: expected version %d, got %d",
            expectedVersion, currentVersion)
    }

    for i, event := range events {
        event.Version = expectedVersion + i + 1
        _, err = tx.ExecContext(ctx,
            `INSERT INTO events (id, aggregate_id, aggregate_type, type, version, data, timestamp)
             VALUES ($1, $2, $3, $4, $5, $6, $7)`,
            event.ID, aggID, event.AggregateType, event.Type,
            event.Version, event.Data, event.Timestamp,
        )
        if err != nil {
            return err
        }
    }

    return tx.Commit()
}

// SQL schema:
// CREATE TABLE events (
//     id UUID PRIMARY KEY,
//     aggregate_id UUID NOT NULL,
//     aggregate_type VARCHAR(255) NOT NULL,
//     type VARCHAR(255) NOT NULL,
//     version INT NOT NULL,
//     data JSONB NOT NULL,
//     timestamp TIMESTAMPTZ NOT NULL,
//     UNIQUE(aggregate_id, version)
// );
```

### Aggregate с Event Sourcing

```go
type OrderAggregate struct {
    id      string
    status  OrderStatus
    total   int
    items   []OrderItem
    version int
    changes []Event // uncommitted events
}

func NewOrderFromEvents(events []Event) *OrderAggregate {
    o := &OrderAggregate{}
    for _, e := range events {
        o.apply(e)
    }
    return o
}

func (o *OrderAggregate) CreateOrder(userID string, items []OrderItem) {
    o.raise(OrderCreatedEvent{
        OrderID: o.id,
        UserID:  userID,
        Items:   items,
    })
}

func (o *OrderAggregate) Cancel() error {
    if o.status != OrderStatusPending {
        return errors.New("can only cancel pending orders")
    }
    o.raise(OrderCancelledEvent{OrderID: o.id})
    return nil
}

// apply — применить событие к состоянию (без side effects!)
func (o *OrderAggregate) apply(event Event) {
    switch e := event.Data.(type) {
    case OrderCreatedEvent:
        o.id = e.OrderID
        o.status = OrderStatusPending
        o.items = e.Items
        o.total = calculateTotal(e.Items)
    case OrderCancelledEvent:
        o.status = OrderStatusCancelled
    }
    o.version = event.Version
}

// raise — записать новое событие
func (o *OrderAggregate) raise(data any) {
    event := Event{
        ID:        uuid.NewString(),
        Type:      reflect.TypeOf(data).Name(),
        Data:      data,
        Timestamp: time.Now(),
    }
    o.apply(event)
    o.changes = append(o.changes, event)
}

func (o *OrderAggregate) UncommittedChanges() []Event { return o.changes }
func (o *OrderAggregate) ClearChanges()                { o.changes = nil }
```

## Snapshot

```go
type Snapshot struct {
    AggregateID string
    Version     int
    Data        json.RawMessage
    CreatedAt   time.Time
}

// Загрузка: snapshot + events после snapshot
func LoadOrder(ctx context.Context, store EventStore, snapStore SnapshotStore, id string) (*OrderAggregate, error) {
    snap, err := snapStore.Load(ctx, id)
    if err != nil && !errors.Is(err, ErrNotFound) {
        return nil, err
    }

    var order OrderAggregate
    fromVersion := 0

    if snap != nil {
        json.Unmarshal(snap.Data, &order)
        fromVersion = snap.Version
    }

    events, err := store.LoadFrom(ctx, id, fromVersion)
    if err != nil {
        return nil, err
    }
    for _, e := range events {
        order.apply(e)
    }

    return &order, nil
}

// Сохранять snapshot каждые N событий
const snapshotEvery = 100

func SaveOrder(ctx context.Context, store EventStore, snapStore SnapshotStore, order *OrderAggregate) error {
    if err := store.Append(ctx, order.id, order.version, order.UncommittedChanges()); err != nil {
        return err
    }
    if order.version%snapshotEvery == 0 {
        data, _ := json.Marshal(order)
        snapStore.Save(ctx, &Snapshot{AggregateID: order.id, Version: order.version, Data: data})
    }
    order.ClearChanges()
    return nil
}
```
