# Wrapping и Unwrapping ошибок

## Обзор

Оборачивание ошибок добавляет контекст, сохраняя оригинальную ошибку в цепочке. Ключевая фича с Go 1.13 (%w).

## Концепции

### fmt.Errorf с %w

```go
// Оборачиваем ошибку с контекстом
func ReadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("reading config %s: %w", path, err)
    }
    // ...
}

// Цепочка ошибок:
// "reading config app.yaml: open app.yaml: no such file or directory"
// Внутри: ReadConfig error → os.ReadFile error → syscall error
```

### %w vs %v

```go
// %w — ОБОРАЧИВАЕТ (сохраняет цепочку)
fmt.Errorf("query failed: %w", err)
// errors.Is(result, err) == true ← можно проверить оригинал

// %v — НЕ оборачивает (теряет цепочку)
fmt.Errorf("query failed: %v", err)
// errors.Is(result, err) == false ← оригинал потерян
```

### Множественный %w (Go 1.20+)

```go
// С Go 1.20 можно оборачивать НЕСКОЛЬКО ошибок
err := fmt.Errorf("operation failed: %w and %w", err1, err2)

errors.Is(err, err1) // true
errors.Is(err, err2) // true

// errors.Unwrap возвращает только первую
// Для доступа ко всем: errors.Join или метод Unwrap() []error
```

### errors.Join (Go 1.20+)

```go
// Объединение нескольких ошибок
var errs []error
for _, item := range items {
    if err := process(item); err != nil {
        errs = append(errs, err)
    }
}
if err := errors.Join(errs...); err != nil {
    return err // содержит все ошибки
}
```

### Когда оборачивать, а когда нет

```go
// ОБОРАЧИВАЙ: когда вызывающему коду нужно проверить оригинальную ошибку
return fmt.Errorf("get user: %w", err) // errors.Is(err, sql.ErrNoRows) сработает

// НЕ ОБОРАЧИВАЙ: когда это деталь реализации
return fmt.Errorf("get user: %v", err) // скрываем внутреннюю ошибку
// Например: не оборачивай ошибки БД в HTTP-слое — это утечка абстракции
```

## Частые вопросы на собеседованиях

**Q: В чём разница между %w и %v в fmt.Errorf?**
A: %w оборачивает ошибку, сохраняя цепочку для errors.Is/As. %v форматирует как строку, теряя связь с оригиналом.

**Q: Когда НЕ нужно оборачивать ошибку?**
A: Когда оригинальная ошибка — деталь реализации. Оборачивание экспортирует зависимость от внутренней ошибки через API.

**Q: Что делает errors.Join?**
A: Объединяет несколько ошибок в одну. errors.Is/As проверяет каждую.

## Подводные камни

1. **Слишком длинные цепочки** — каждый слой добавляет контекст, сообщение становится нечитаемым. Оборачивай только когда добавляешь ПОЛЕЗНЫЙ контекст.
2. **Не оборачивай дважды одну ошибку** — дублирование контекста.
