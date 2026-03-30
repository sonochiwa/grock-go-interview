# Sentinel Errors

## Обзор

Sentinel error — предопределённая ошибка-значение, объявленная как переменная пакета. Именуется `Err...`.

## Концепции

```go
// Стандартная библиотека
var (
    io.EOF              // конец потока
    sql.ErrNoRows       // запрос не вернул строк
    os.ErrNotExist      // файл не существует
    os.ErrPermission    // нет прав
    context.Canceled    // контекст отменён
    context.DeadlineExceeded // таймаут
)

// Использование
data, err := io.ReadAll(r)
if errors.Is(err, io.EOF) {
    // нормальное завершение чтения
}

// Определение своих sentinel errors
var (
    ErrNotFound     = errors.New("not found")
    ErrUnauthorized = errors.New("unauthorized")
    ErrConflict     = errors.New("conflict")
)
```

### Когда использовать sentinel errors

- Ошибка имеет **фиксированный смысл** без дополнительного контекста
- Вызывающий код проверяет **конкретную ошибку** через `errors.Is`
- Ошибка является частью **публичного API**

### Проблемы sentinel errors

```go
// Tight coupling: импортирующий пакет зависит от вашей переменной
if err == mypkg.ErrNotFound { ... }

// Нет контекста: "not found" ЧТО?
// Решение: оборачивай при возврате
return fmt.Errorf("user %d: %w", id, ErrNotFound)
```

## Частые вопросы на собеседованиях

**Q: Чем sentinel error отличается от custom error type?**
A: Sentinel — одно значение, проверяется через `errors.Is`. Custom type — структура с полями, проверяется через `errors.As`, содержит дополнительный контекст.

**Q: Почему используется errors.Is, а не ==?**
A: `errors.Is` проходит по всей цепочке обёрнутых ошибок. `==` проверяет только верхний уровень.

## Подводные камни

1. **Не используй `var Err... = fmt.Errorf(...)`** — создаёт новую ошибку при каждом вызове, сравнение не сработает. Используй `errors.New`.
2. **Не экспортируй ошибки** без необходимости — это часть API, которую придётся поддерживать.
