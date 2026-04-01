# Graceful Server

Реализуй HTTP сервер с graceful shutdown:

- `NewGracefulServer(handler http.Handler) *GracefulServer`
- `Start(addr string) error` — запускает сервер
- `Shutdown(ctx context.Context) error` — graceful stop:
  1. Перестать принимать новые запросы
  2. Дождаться завершения текущих (или timeout из ctx)
  3. Вернуть nil если все завершились, ctx.Err() если timeout

Добавь `ActiveRequests() int64` для отслеживания.
