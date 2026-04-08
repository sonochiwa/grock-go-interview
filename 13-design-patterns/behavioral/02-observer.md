# Observer

## В Go

Observer (Pub/Sub) — один объект уведомляет множество подписчиков о событиях. В Go реализуется через каналы (см. 06-concurrency-patterns/08-pub-sub.md).

```go
type EventBus struct {
    mu          sync.RWMutex
    subscribers map[string][]chan Event
}

func (eb *EventBus) Subscribe(topic string) <-chan Event {
    ch := make(chan Event, 10)
    eb.mu.Lock()
    eb.subscribers[topic] = append(eb.subscribers[topic], ch)
    eb.mu.Unlock()
    return ch
}

func (eb *EventBus) Publish(topic string, event Event) {
    eb.mu.RLock()
    defer eb.mu.RUnlock()
    for _, ch := range eb.subscribers[topic] {
        select {
        case ch <- event:
        default: // drop if subscriber is slow
        }
    }
}
```

В Go каналы — естественная реализация Observer. Не нужны callback'и как в других языках.
