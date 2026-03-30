# Стратегии обработки ошибок

## Обзор

Как выстроить обработку ошибок в многослойном приложении. panic/recover, антипаттерны, лучшие практики.

## panic/recover

```go
// panic — аварийная остановка (раскрутка стека)
func mustParseURL(raw string) *url.URL {
    u, err := url.Parse(raw)
    if err != nil {
        panic(fmt.Sprintf("invalid URL: %s", raw))
    }
    return u
}

// recover — перехват паники (только в defer)
func safeHandler(w http.ResponseWriter, r *http.Request) {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("panic recovered: %v\n%s", r, debug.Stack())
            http.Error(w, "internal error", 500)
        }
    }()
    // ... обработка запроса
}
```

### Когда panic допустим

1. **Инициализация** — `regexp.MustCompile`, невалидный конфиг при старте
2. **Программная ошибка** — индекс за пределами, nil pointer (Go сам паникует)
3. **Must-функции** — когда ошибка означает баг в коде, не в данных
4. **Никогда** — в библиотечном коде (кроме Must-вариантов)

## Обработка ошибок по слоям

```
HTTP Handler → Service → Repository → Database
     ↑              ↑          ↑
   HTTP 4xx/5xx   бизнес    sql ошибки
                  ошибки
```

```go
// Repository: оборачивает DB ошибки
func (r *UserRepo) GetByID(id int64) (*User, error) {
    user, err := r.db.QueryRow(...)
    if err == sql.ErrNoRows {
        return nil, ErrNotFound // свой sentinel, НЕ sql.ErrNoRows
    }
    if err != nil {
        return nil, fmt.Errorf("query user %d: %w", id, err)
    }
    return user, nil
}

// Service: бизнес-логика
func (s *UserService) GetUser(id int64) (*User, error) {
    user, err := s.repo.GetByID(id)
    if err != nil {
        return nil, fmt.Errorf("get user: %w", err)
    }
    return user, nil
}

// Handler: маппинг на HTTP
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
    user, err := h.service.GetUser(id)
    if errors.Is(err, ErrNotFound) {
        http.Error(w, "user not found", 404)
        return
    }
    if err != nil {
        log.Printf("error: %v", err) // логируем ОДИН раз, на верхнем уровне
        http.Error(w, "internal error", 500)
        return
    }
    json.NewEncoder(w).Encode(user)
}
```

## Антипаттерны

```go
// 1. Логирование И возврат — ошибка залогируется дважды
if err != nil {
    log.Printf("error: %v", err) // ❌
    return err                     // вызывающий тоже залогирует
}

// 2. Глотание ошибки
result, _ := doSomething() // ❌ куда делась ошибка?

// 3. Пустой контекст
return fmt.Errorf("failed: %w", err) // ❌ ЧТО failed?
return fmt.Errorf("loading user %d: %w", id, err) // ✅

// 4. Проверка строки ошибки
if err.Error() == "not found" { ... } // ❌ хрупко
if errors.Is(err, ErrNotFound) { ... } // ✅
```

## Частые вопросы на собеседованиях

**Q: Когда использовать panic, а когда error?**
A: panic — для программных ошибок и инициализации. error — для ожидаемых ситуаций (файл не найден, таймаут, невалидный ввод).

**Q: Где логировать ошибку?**
A: На верхнем уровне (handler, main). Промежуточные слои оборачивают и пробрасывают.

**Q: Почему "логирование и возврат" — антипаттерн?**
A: Ошибка логируется на каждом уровне стека, засоряя логи дубликатами.

## Подводные камни

1. **recover() работает только в defer** и только в той же горутине. Паника в дочерней горутине не перехватывается родительской.
2. **panic(nil)** — recover() вернёт nil, но паника произошла. С Go 1.21 panic(nil) оборачивается в *runtime.PanicNilError.
