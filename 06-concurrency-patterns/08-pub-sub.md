# Pub/Sub

## Обзор

Publish-Subscribe — издатель отправляет сообщения, подписчики получают копии. Развязывает компоненты.

## Концепции

```go
type Broker[T any] struct {
    mu          sync.RWMutex
    subscribers map[string][]chan T
}

func NewBroker[T any]() *Broker[T] {
    return &Broker[T]{subscribers: make(map[string][]chan T)}
}

func (b *Broker[T]) Subscribe(topic string, bufSize int) <-chan T {
    ch := make(chan T, bufSize)
    b.mu.Lock()
    b.subscribers[topic] = append(b.subscribers[topic], ch)
    b.mu.Unlock()
    return ch
}

func (b *Broker[T]) Publish(topic string, msg T) {
    b.mu.RLock()
    defer b.mu.RUnlock()
    for _, ch := range b.subscribers[topic] {
        select {
        case ch <- msg:
        default:
            // подписчик не успевает — пропускаем (backpressure)
        }
    }
}

func (b *Broker[T]) Close(topic string) {
    b.mu.Lock()
    defer b.mu.Unlock()
    for _, ch := range b.subscribers[topic] {
        close(ch)
    }
    delete(b.subscribers, topic)
}

// Использование
broker := NewBroker[Event]()
sub1 := broker.Subscribe("orders", 100)
sub2 := broker.Subscribe("orders", 100)

go func() {
    for event := range sub1 { handleEvent(event) }
}()

broker.Publish("orders", Event{Type: "created", ID: 42})
```

## Частые вопросы на собеседованиях

**Q: Как обработать медленного подписчика?**
A: Буферизированный канал + select с default (drop) или отдельная горутина с таймаутом.
