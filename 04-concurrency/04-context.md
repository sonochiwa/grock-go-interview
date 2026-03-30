# Context

## Обзор

context.Context управляет жизненным циклом операций: отмена, таймауты, передача request-scoped данных. Первый аргумент каждой функции, которая может быть отменена.

## Концепции

### Создание

```go
// Корневые контексты
ctx := context.Background() // для main(), init(), тестов
ctx := context.TODO()       // заглушка — "потом добавлю нормальный context"

// Производные контексты
ctx, cancel := context.WithCancel(parent)
ctx, cancel := context.WithTimeout(parent, 5*time.Second)
ctx, cancel := context.WithDeadline(parent, time.Now().Add(5*time.Second))
ctx = context.WithValue(parent, key, value)
// ВСЕГДА вызывай cancel() когда операция завершена!
```

### WithCancel

```go
func longOperation(ctx context.Context) error {
    ctx, cancel := context.WithCancel(ctx)
    defer cancel() // освободить ресурсы

    for i := 0; i < 1000; i++ {
        select {
        case <-ctx.Done():
            return ctx.Err() // context.Canceled или context.DeadlineExceeded
        default:
            doStep(i)
        }
    }
    return nil
}
```

### WithTimeout

```go
func fetchUser(ctx context.Context, id int64) (*User, error) {
    ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
    defer cancel()

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        // err содержит context.DeadlineExceeded если таймаут
        return nil, fmt.Errorf("fetch user %d: %w", id, err)
    }
    defer resp.Body.Close()
    // ...
}
```

### WithValue (осторожно!)

```go
// Для request-scoped данных (trace ID, user ID, auth token)
type contextKey string
const userIDKey contextKey = "userID"

func WithUserID(ctx context.Context, id int64) context.Context {
    return context.WithValue(ctx, userIDKey, id)
}

func UserIDFromContext(ctx context.Context) (int64, bool) {
    id, ok := ctx.Value(userIDKey).(int64)
    return id, ok
}

// НЕ ИСПОЛЬЗУЙ WithValue для:
// - Передачи зависимостей (используй DI)
// - Опциональных параметров функции
// - Данных, которые нужны только одной функции
// - Строковых ключей (коллизии!)
```

### Каскадная отмена

```go
// Отмена родителя отменяет всех детей
parent, parentCancel := context.WithCancel(context.Background())

child1, cancel1 := context.WithTimeout(parent, 5*time.Second)
child2, cancel2 := context.WithCancel(parent)
defer cancel1()
defer cancel2()

parentCancel() // отменяет parent, child1 и child2!
```

### Context в HTTP

```go
// Сервер: request context отменяется при disconnect клиента
func handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context() // отменится при disconnect

    result, err := longOperation(ctx)
    if err != nil {
        if ctx.Err() != nil {
            return // клиент отключился, нет смысла отвечать
        }
        http.Error(w, err.Error(), 500)
        return
    }
    json.NewEncoder(w).Encode(result)
}
```

### Best practices

```go
// 1. context — ПЕРВЫЙ аргумент, ВСЕГДА
func GetUser(ctx context.Context, id int64) (*User, error)

// 2. НИКОГДА не храни context в структуре
type Server struct {
    // ctx context.Context ← НЕ ДЕЛАЙ ТАК
}

// 3. ВСЕГДА вызывай cancel()
ctx, cancel := context.WithTimeout(ctx, time.Second)
defer cancel() // даже если операция завершилась раньше

// 4. Не передавай nil context
// Если не знаешь какой — context.TODO()

// 5. WithoutCancel (Go 1.21+) — для фоновых задач после ответа
bgCtx := context.WithoutCancel(r.Context())
go logAnalytics(bgCtx, data) // не отменится при disconnect клиента
```

## Под капотом

Context — дерево. Каждый WithCancel/WithTimeout создаёт узел, подписывающийся на отмену родителя. cancel() рекурсивно отменяет всех детей.

```go
// Упрощённая структура cancelCtx
type cancelCtx struct {
    Context    // родитель
    mu       sync.Mutex
    done     chan struct{} // закрывается при отмене
    children map[canceler]struct{} // дочерние контексты
    err      error
}
```

## Частые вопросы на собеседованиях

**Q: Зачем вызывать cancel() если операция завершилась успешно?**
A: Освобождает ресурсы (горутина таймера, запись в parent.children). Без defer cancel() — утечка.

**Q: Чем WithTimeout отличается от WithDeadline?**
A: WithTimeout — относительное время (через N секунд). WithDeadline — абсолютное (до конкретного момента). Внутри WithTimeout вызывает WithDeadline.

**Q: Почему нельзя хранить context в структуре?**
A: Context привязан к одной операции/запросу. Хранение в структуре разделяет один context между разными операциями — нарушает семантику.

**Q: Что делает context.WithoutCancel (Go 1.21)?**
A: Создаёт context, который наследует values, но не наследует отмену. Для фоновых задач, которые должны продолжиться после отмены родителя.

## Подводные камни

1. **Забытый defer cancel()** — утечка ресурсов.
2. **WithValue со строковым ключом** — коллизии между пакетами. Используй unexported type.
3. **Слишком большой таймаут** — 30 секунд на HTTP запрос к БД — по сути нет таймаута.
4. **Игнорирование ctx.Done()** в долгих операциях — context бесполезен если не проверяешь.
