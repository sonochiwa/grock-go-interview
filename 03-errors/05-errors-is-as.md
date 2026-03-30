# errors.Is и errors.As

## Обзор

Две функции для проверки ошибок в цепочке. Is — сравнивает с конкретным значением. As — ищет конкретный тип.

## Концепции

### errors.Is

```go
// Рекурсивно проходит по цепочке Unwrap, сравнивая каждую ошибку
err := fmt.Errorf("read config: %w",
    fmt.Errorf("open file: %w", os.ErrNotExist))

errors.Is(err, os.ErrNotExist) // true — нашёл в цепочке!

// Эквивалент (но НЕ делай так):
// err == os.ErrNotExist // false — проверяет только верхний уровень
```

### errors.As

```go
// Ищет ошибку конкретного ТИПА в цепочке
var pathErr *os.PathError
if errors.As(err, &pathErr) {
    fmt.Println("Operation:", pathErr.Op)
    fmt.Println("Path:", pathErr.Path)
}

// target должен быть указателем на тип ошибки или интерфейс
// errors.As сам разыменовывает и заполняет target
```

### Полный пример

```go
type ValidationError struct {
    Field string
    Rule  string
}
func (e *ValidationError) Error() string { ... }

func validateUser(u User) error {
    if u.Name == "" {
        return &ValidationError{Field: "name", Rule: "required"}
    }
    return nil
}

func createUser(u User) error {
    if err := validateUser(u); err != nil {
        return fmt.Errorf("create user: %w", err)
    }
    // ...
}

// В HTTP handler:
err := createUser(user)

var ve *ValidationError
if errors.As(err, &ve) {
    // HTTP 400 — bad request
    http.Error(w, ve.Error(), http.StatusBadRequest)
    return
}
if err != nil {
    // HTTP 500 — internal error
    http.Error(w, "internal error", http.StatusInternalServerError)
}
```

## Частые вопросы на собеседованиях

**Q: Чем errors.Is отличается от ==?**
A: errors.Is проходит по всей цепочке Unwrap. == сравнивает только текущее значение.

**Q: Чем errors.As отличается от type assertion?**
A: errors.As проходит по цепочке и заполняет target. Type assertion работает только с текущим значением.

**Q: Можно ли реализовать кастомный Is/As?**
A: Да. Если тип ошибки имеет метод `Is(error) bool` или `As(any) bool`, errors.Is/As вызовет его.
