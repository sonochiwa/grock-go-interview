# DDD (Domain-Driven Design) в Go

## Building Blocks

```
Entity       — имеет identity (ID), мутабельный
Value Object — определяется значениями, иммутабельный
Aggregate    — кластер entities с root entity
Repository   — доступ к aggregates
Domain Event — что произошло в домене
Domain Service — логика, не принадлежащая одному entity
```

## Entity

```go
// Entity имеет уникальный ID
type Order struct {
    id        OrderID // private! изменять нельзя
    userID    UserID
    items     []OrderItem
    status    OrderStatus
    total     Money
    createdAt time.Time
    updatedAt time.Time
}

// Конструктор с валидацией
func NewOrder(userID UserID, items []OrderItem) (*Order, error) {
    if len(items) == 0 {
        return nil, ErrEmptyOrder
    }
    o := &Order{
        id:        NewOrderID(),
        userID:    userID,
        items:     items,
        status:    StatusPending,
        createdAt: time.Now(),
    }
    o.recalculateTotal()
    return o, nil
}

// Бизнес-методы (не сеттеры!)
func (o *Order) AddItem(item OrderItem) error {
    if o.status != StatusPending {
        return ErrOrderNotEditable
    }
    o.items = append(o.items, item)
    o.recalculateTotal()
    o.updatedAt = time.Now()
    return nil
}

// Getters (не сеттеры — защита инвариантов)
func (o *Order) ID() OrderID       { return o.id }
func (o *Order) Status() OrderStatus { return o.status }
func (o *Order) Total() Money       { return o.total }
```

## Value Object

```go
// Value Object — иммутабельный, сравнивается по значению
type Money struct {
    amount   int    // в минимальных единицах (копейки/центы)
    currency string
}

func NewMoney(amount int, currency string) (Money, error) {
    if amount < 0 {
        return Money{}, errors.New("amount cannot be negative")
    }
    if currency == "" {
        return Money{}, errors.New("currency is required")
    }
    return Money{amount: amount, currency: currency}, nil
}

func (m Money) Add(other Money) (Money, error) {
    if m.currency != other.currency {
        return Money{}, errors.New("cannot add different currencies")
    }
    return Money{amount: m.amount + other.amount, currency: m.currency}, nil
}

func (m Money) Amount() int    { return m.amount }
func (m Money) Currency() string { return m.currency }

// Email как Value Object
type Email struct {
    value string
}

func NewEmail(raw string) (Email, error) {
    // Валидация
    if !strings.Contains(raw, "@") {
        return Email{}, fmt.Errorf("invalid email: %s", raw)
    }
    return Email{value: strings.ToLower(strings.TrimSpace(raw))}, nil
}

func (e Email) String() string { return e.value }
```

## Aggregate

```go
// Aggregate Root = Order
// Aggregate содержит: Order + OrderItems
// Все изменения через Aggregate Root

type Order struct {
    id     OrderID
    items  []OrderItem
    status OrderStatus
    total  Money
    events []DomainEvent // domain events для публикации
}

// Инварианты проверяются в aggregate root
func (o *Order) Confirm() error {
    if o.status != StatusPending {
        return fmt.Errorf("cannot confirm order in status %s", o.status)
    }
    if len(o.items) == 0 {
        return ErrEmptyOrder
    }
    if o.total.Amount() <= 0 {
        return ErrInvalidTotal
    }

    o.status = StatusConfirmed
    o.addEvent(OrderConfirmedEvent{
        OrderID:   o.id,
        Total:     o.total,
        Timestamp: time.Now(),
    })
    return nil
}

func (o *Order) addEvent(event DomainEvent) {
    o.events = append(o.events, event)
}

func (o *Order) PopEvents() []DomainEvent {
    events := o.events
    o.events = nil
    return events
}
```

## Repository

```go
// Repository работает с Aggregate целиком
type OrderRepository interface {
    FindByID(ctx context.Context, id OrderID) (*Order, error)
    Save(ctx context.Context, order *Order) error
    // НЕ Update отдельных полей!
    // НЕ FindByStatus — это query, не domain operation
}

// Для queries — отдельный read-only repository (CQRS)
type OrderQueryRepository interface {
    ListByUser(ctx context.Context, userID UserID, page Pagination) ([]*OrderView, error)
    Search(ctx context.Context, filter OrderFilter) ([]*OrderView, error)
}
```

## Domain Service

```go
// Логика, которая не принадлежит одному aggregate
type PricingService struct {
    discountRepo DiscountRepository
}

func (s *PricingService) CalculatePrice(ctx context.Context, order *Order, user *User) (Money, error) {
    baseTotal := order.Total()

    discount, err := s.discountRepo.GetActiveForUser(ctx, user.ID())
    if err != nil {
        return Money{}, err
    }

    return discount.Apply(baseTotal), nil
}

// Transfer money — затрагивает 2 aggregate
type TransferService struct {
    accountRepo AccountRepository
}

func (s *TransferService) Transfer(ctx context.Context, from, to AccountID, amount Money) error {
    fromAcc, err := s.accountRepo.FindByID(ctx, from)
    if err != nil {
        return err
    }
    toAcc, err := s.accountRepo.FindByID(ctx, to)
    if err != nil {
        return err
    }

    if err := fromAcc.Withdraw(amount); err != nil {
        return err
    }
    toAcc.Deposit(amount)

    // Сохранить оба в одной транзакции
    return s.accountRepo.SaveBoth(ctx, fromAcc, toAcc)
}
```

## Domain Events

```go
type DomainEvent interface {
    EventName() string
    OccurredAt() time.Time
}

type OrderConfirmedEvent struct {
    OrderID   OrderID
    Total     Money
    Timestamp time.Time
}

func (e OrderConfirmedEvent) EventName() string     { return "order.confirmed" }
func (e OrderConfirmedEvent) OccurredAt() time.Time { return e.Timestamp }

// Публикация после сохранения aggregate
func (uc *ConfirmOrderUseCase) Execute(ctx context.Context, orderID OrderID) error {
    order, err := uc.repo.FindByID(ctx, orderID)
    if err != nil {
        return err
    }

    if err := order.Confirm(); err != nil {
        return err
    }

    if err := uc.repo.Save(ctx, order); err != nil {
        return err
    }

    // Публиковать events ПОСЛЕ успешного сохранения
    for _, event := range order.PopEvents() {
        uc.publisher.Publish(ctx, event)
    }

    return nil
}
```

## DDD в Go: практические советы

```
1. Go != Java. Не нужны геттеры/сеттеры для всего.
   Используй private fields + методы с бизнес-смыслом.

2. Не усложняй. DDD нужен для сложных доменов.
   Для CRUD — Clean Architecture достаточно.

3. Aggregate boundaries:
   - Один aggregate = одна транзакция
   - Между aggregates — eventual consistency
   - Маленькие aggregates → лучше concurrency

4. Value Objects через struct (не pointer):
   - Иммутабельность через private fields
   - Сравнение через == (если нет slice/map полей)

5. Ubiquitous Language:
   - Order.Confirm(), не Order.SetStatus("confirmed")
   - Money.Add(), не Money.SetAmount(a + b)
```
