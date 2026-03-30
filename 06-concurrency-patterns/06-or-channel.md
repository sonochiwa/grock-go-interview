# Or-Channel

## Обзор

Паттерн "первый результат" — берём результат от первого завершившегося источника, остальные отменяем.

## Концепции

### Or-Done: обёртка для чистой отмены

```go
// orDone оборачивает канал, добавляя проверку done
func orDone(ctx context.Context, c <-chan int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for {
            select {
            case <-ctx.Done():
                return
            case v, ok := <-c:
                if !ok {
                    return
                }
                select {
                case out <- v:
                case <-ctx.Done():
                    return
                }
            }
        }
    }()
    return out
}

// Без orDone:
for val := range ch {
    select {
    case <-ctx.Done():
        return
    default:
    }
    process(val)
}

// С orDone (чище):
for val := range orDone(ctx, ch) {
    process(val)
}
```

### First-Result: гонка нескольких источников

```go
// Запрашиваем из нескольких источников, берём первый ответ
func firstResult(ctx context.Context, fns ...func(context.Context) (string, error)) (string, error) {
    ctx, cancel := context.WithCancel(ctx)
    defer cancel() // отменяем остальных после получения первого результата

    type result struct {
        val string
        err error
    }

    ch := make(chan result, len(fns)) // буферизированный!

    for _, fn := range fns {
        fn := fn
        go func() {
            val, err := fn(ctx)
            ch <- result{val, err}
        }()
    }

    // Берём первый успешный
    for range fns {
        r := <-ch
        if r.err == nil {
            return r.val, nil // cancel() отменит остальных
        }
    }
    return "", errors.New("all sources failed")
}

// Использование: запрос к нескольким зеркалам
result, err := firstResult(ctx,
    func(ctx context.Context) (string, error) { return fetch(ctx, "mirror1.com") },
    func(ctx context.Context) (string, error) { return fetch(ctx, "mirror2.com") },
    func(ctx context.Context) (string, error) { return fetch(ctx, "mirror3.com") },
)
```

### Рекурсивный or-channel

```go
// or объединяет N каналов: закрывается когда любой из них закрывается
func or(channels ...<-chan struct{}) <-chan struct{} {
    switch len(channels) {
    case 0:
        return nil
    case 1:
        return channels[0]
    }

    orDone := make(chan struct{})
    go func() {
        defer close(orDone)
        switch len(channels) {
        case 2:
            select {
            case <-channels[0]:
            case <-channels[1]:
            }
        default:
            // Делим пополам и рекурсивно
            mid := len(channels) / 2
            select {
            case <-or(channels[:mid]...):
            case <-or(channels[mid:]...):
            }
        }
    }()
    return orDone
}
```

## Частые вопросы на собеседованиях

**Q: Зачем буферизированный канал в first-result?**
A: Чтобы горутины-"проигравшие" не заблокировались при отправке результата. Без буфера — утечка горутин.

**Q: Как отменить "проигравших"?**
A: context.WithCancel + defer cancel(). Все горутины используют один контекст.
