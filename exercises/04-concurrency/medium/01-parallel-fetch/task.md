# Parallel Fetch

Напиши функцию `FetchAll(ctx context.Context, urls []string, fetch func(ctx context.Context, url string) (string, error)) ([]Result, error)`. Fetches all URLs concurrently.

## Требования

- `Result{URL string, Body string, Err error}` — результат для каждого URL
- Все URL обрабатываются конкурентно
- Если `ctx` отменён — прекратить работу и вернуть ошибку контекста
- Используй `errgroup` с `SetLimit(10)` для ограничения параллелизма
- Результаты возвращаются в том же порядке, что и входные URL
- Ошибки отдельных fetch сохраняются в `Result.Err`, не прерывают остальные

## Пример

```go
results, err := FetchAll(ctx, []string{"https://a.com", "https://b.com"}, myFetch)
// results[0].URL == "https://a.com"
// results[0].Body == "..." или results[0].Err != nil
```
