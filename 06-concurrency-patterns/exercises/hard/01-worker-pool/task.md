# Worker Pool

Реализуй generic worker pool:

- `NewPool[T, R any](workers int, handler func(T) (R, error)) *Pool[T, R]`
- `Submit(task T) <-chan Result[R]` — отправляет задачу, возвращает канал с результатом
- `Close()` — graceful shutdown, ждёт завершения текущих задач

`Result[R]` содержит `Value R` и `Err error`.
