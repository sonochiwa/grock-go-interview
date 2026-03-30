# Кастомные типы ошибок

## Обзор

Когда sentinel error недостаточно — нужен контекст: какой файл, какая операция, какой HTTP код. Для этого создаём свой тип, реализующий error.

## Концепции

```go
// Кастомный тип ошибки
type NotFoundError struct {
    Entity string
    ID     int64
}

func (e *NotFoundError) Error() string {
    return fmt.Sprintf("%s with id %d not found", e.Entity, e.ID)
}

// Использование
func GetUser(id int64) (*User, error) {
    user, err := db.FindUser(id)
    if err == sql.ErrNoRows {
        return nil, &NotFoundError{Entity: "user", ID: id}
    }
    return user, err
}

// Проверка
var target *NotFoundError
if errors.As(err, &target) {
    fmt.Printf("not found: %s %d\n", target.Entity, target.ID)
    // HTTP 404
}
```

### Реальный пример из стандартной библиотеки

```go
// os.PathError
type PathError struct {
    Op   string // "open", "read", "write"
    Path string // путь к файлу
    Err  error  // оригинальная ошибка
}

func (e *PathError) Error() string {
    return e.Op + " " + e.Path + ": " + e.Err.Error()
}

func (e *PathError) Unwrap() error { return e.Err }
// Unwrap позволяет errors.Is/As проходить через обёртку
```

### Кастомный Is/As

```go
type HTTPError struct {
    Code    int
    Message string
}

func (e *HTTPError) Error() string {
    return fmt.Sprintf("HTTP %d: %s", e.Code, e.Message)
}

// Кастомный Is: два HTTPError "равны" если совпадает код
func (e *HTTPError) Is(target error) bool {
    t, ok := target.(*HTTPError)
    if !ok {
        return false
    }
    return e.Code == t.Code
}

// Теперь:
err := &HTTPError{Code: 404, Message: "user not found"}
errors.Is(err, &HTTPError{Code: 404}) // true (коды совпадают)
```

## Частые вопросы на собеседованиях

**Q: Когда использовать sentinel error, а когда custom type?**
A: Sentinel — когда достаточно факта ошибки ("не найдено"). Custom type — когда нужен контекст (что не найдено, какой ID).

**Q: Зачем метод Unwrap()?**
A: Позволяет errors.Is/As проходить по цепочке обёрнутых ошибок. Без Unwrap — проверяется только верхний уровень.

## Подводные камни

1. **Используй pointer receiver** для Error() — иначе `errors.As` с указателем не сработает.
2. **Не забудь Unwrap()** если оборачиваешь другую ошибку — иначе errors.Is не дойдёт до оригинала.
