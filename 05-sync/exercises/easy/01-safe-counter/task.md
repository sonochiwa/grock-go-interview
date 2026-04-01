# Safe Counter

Реализуй два goroutine-safe счётчика:

1. `Counter` — на `sync.Mutex`: `Inc()`, `Dec()`, `Value() int64`
2. `AtomicCounter` — на `atomic.Int64`: те же методы

Оба должны корректно работать при конкурентном доступе.
