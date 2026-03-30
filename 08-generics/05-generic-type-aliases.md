# Generic Type Aliases (Go 1.24)

## Обзор

С Go 1.24 type aliases могут иметь type parameters. Позволяет создавать алиасы для generic типов.

## Концепции

```go
// До Go 1.24: нельзя
// type MySlice[T any] = []T // ОШИБКА

// С Go 1.24: можно
type MySlice[T any] = []T
type Pair[A, B any] = struct{ First A; Second B }

// Практическое применение: миграция типов между пакетами
// old пакет:
type OldResult[T any] = newpkg.Result[T] // алиас на новый тип
// Позволяет плавную миграцию без breaking changes
```

## Когда использовать

- Плавная миграция типов между пакетами
- Сокращение длинных generic типов
- Backwards compatibility при рефакторинге
