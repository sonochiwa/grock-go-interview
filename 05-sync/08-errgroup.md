# errgroup

## Обзор

`golang.org/x/sync/errgroup` — WaitGroup + обработка ошибок + контекст. Стандартный способ запуска параллельных задач с ошибками.

## Концепции

```go
import "golang.org/x/sync/errgroup"

// Базовый пример
g, ctx := errgroup.WithContext(context.Background())

g.Go(func() error {
    return fetchUsers(ctx)
})
g.Go(func() error {
    return fetchOrders(ctx)
})

if err := g.Wait(); err != nil {
    // err — ПЕРВАЯ ошибка из любой горутины
    // ctx отменён после первой ошибки
}
```

### SetLimit — ограничение параллелизма

```go
g, ctx := errgroup.WithContext(ctx)
g.SetLimit(10) // максимум 10 горутин одновременно

for _, url := range urls {
    url := url
    g.Go(func() error {
        return fetch(ctx, url)
    })
}
if err := g.Wait(); err != nil { ... }
```

### Практический пример: параллельная обработка

```go
func processItems(ctx context.Context, items []Item) ([]Result, error) {
    results := make([]Result, len(items))
    g, ctx := errgroup.WithContext(ctx)
    g.SetLimit(20)

    for i, item := range items {
        i, item := i, item
        g.Go(func() error {
            res, err := process(ctx, item)
            if err != nil {
                return fmt.Errorf("item %d: %w", i, err)
            }
            results[i] = res // безопасно: каждая горутина пишет в свой индекс
            return nil
        })
    }

    if err := g.Wait(); err != nil {
        return nil, err
    }
    return results, nil
}
```

## Частые вопросы на собеседованиях

**Q: Чем errgroup лучше WaitGroup?**
A: Обработка ошибок (возвращает первую), автоматическая отмена контекста, SetLimit для ограничения параллелизма.

**Q: Какую ошибку вернёт Wait()?**
A: Первую ненулевую ошибку. Остальные ошибки теряются (если нужны все — собирай вручную через mutex + slice).
