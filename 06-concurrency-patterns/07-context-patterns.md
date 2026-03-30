# Context Patterns

## Обзор

Продвинутые паттерны использования context: каскадная отмена, таймауты по слоям, graceful shutdown.

## Каскадная отмена

```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context() // отменяется при disconnect клиента

    // Дочерний контекст с таймаутом
    dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
    defer cancel()

    user, err := db.GetUser(dbCtx, userID) // таймаут 2с ИЛИ disconnect
    if err != nil {
        // ...
    }

    // Ещё один дочерний с другим таймаутом
    apiCtx, cancel2 := context.WithTimeout(ctx, 5*time.Second)
    defer cancel2()

    orders, err := api.GetOrders(apiCtx, user.ID)
    // ...
}
```

## Таймауты по слоям

```go
// HTTP handler: 30 секунд на весь запрос
// → Service: 10 секунд на бизнес-логику
//   → Repository: 2 секунды на каждый запрос к БД
//   → External API: 5 секунд

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
    defer cancel()
    h.service.Process(ctx)
}

func (s *Service) Process(ctx context.Context) error {
    ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()

    // Дочерние операции наследуют оба таймаута
    // Сработает МЕНЬШИЙ из 10с и оставшегося от 30с
    return s.repo.GetData(ctx)
}
```

## AfterFunc (Go 1.21+)

```go
// Выполнить функцию при отмене контекста
stop := context.AfterFunc(ctx, func() {
    conn.Close() // закрыть соединение при отмене
})
defer stop() // отменить AfterFunc если не нужен

// Полезно для cleanup ресурсов, привязанных к контексту
```

## Частые вопросы на собеседованиях

**Q: Что произойдёт если дочерний таймаут больше родительского?**
A: Сработает родительский. WithTimeout(parentWith5s, 30s) = реально 5 секунд.
