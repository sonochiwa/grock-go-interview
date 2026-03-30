# Semaphore

## Обзор

Семафор ограничивает количество одновременных операций. Простейшая реализация — буферизированный канал.

## Концепции

### Через канал

```go
sem := make(chan struct{}, 10) // макс 10 одновременно

for _, url := range urls {
    sem <- struct{}{} // занимаем слот (блокируется если 10 заняты)
    go func(url string) {
        defer func() { <-sem }() // освобождаем слот
        fetch(url)
    }(url)
}

// Дождаться завершения
for i := 0; i < cap(sem); i++ {
    sem <- struct{}{} // ждём пока все слоты освободятся
}
```

### Через x/sync/semaphore (weighted)

```go
import "golang.org/x/sync/semaphore"

sem := semaphore.NewWeighted(10) // макс вес 10

for _, task := range tasks {
    // Acquire блокирует если нет свободного веса
    if err := sem.Acquire(ctx, 1); err != nil {
        return err // context cancelled
    }
    go func(t Task) {
        defer sem.Release(1)
        process(t)
    }(task)
}

// Ждём все
if err := sem.Acquire(ctx, 10); err != nil {
    return err
}
```

### Weighted semaphore — разный "вес" задач

```go
sem := semaphore.NewWeighted(100) // 100 единиц ресурса

// Лёгкая задача: 1 единица
sem.Acquire(ctx, 1)
go lightTask()

// Тяжёлая задача: 10 единиц
sem.Acquire(ctx, 10)
go heavyTask()
```

## Частые вопросы на собеседованиях

**Q: Как реализовать семафор в Go?**
A: `make(chan struct{}, N)`. Send — acquire, receive — release. Или `x/sync/semaphore` для weighted.

**Q: Чем семафор отличается от worker pool?**
A: Семафор ограничивает параллелизм, но создаёт горутину на задачу. Worker pool — фиксированные горутины.
