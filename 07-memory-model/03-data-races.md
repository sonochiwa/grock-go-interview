# Data Races

## Обзор

Data race — одновременный доступ к переменной из нескольких горутин, где хотя бы один доступ — запись, без синхронизации.

## Концепции

```go
// DATA RACE: две горутины читают/пишут counter без синхронизации
var counter int
go func() { counter++ }()
go func() { counter++ }()
// counter может быть 0, 1 или 2

// UNDEFINED BEHAVIOR: Go memory model не определяет результат
```

### Обнаружение: -race

```bash
go test -race ./...
go run -race main.go
go build -race -o myapp

# Race detector:
# - Замедляет программу в 5-10x
# - Увеличивает потребление памяти в 5-10x
# - НЕ ловит все race conditions — только те, что произошли во время запуска
```

### Последствия data race в Go

**В Go data race = undefined behavior!** Не "неправильный результат", а:
- Чтение "рваного" значения (torn read)
- Бесконечные циклы
- Segfault
- Повреждение памяти

```go
// Пример torn read: interface
var x any
go func() { x = "hello" }()
go func() { x = 42 }()
// x может содержать {type: string, data: 42} — torn read!
// Panic: невозможная комбинация типа и данных
```

### Типичные data races

```go
// 1. Неподелённый map
m := make(map[string]int)
go func() { m["a"] = 1 }()
go func() { _ = m["a"] }()
// fatal error: concurrent map read and map write

// 2. Slice append
var s []int
go func() { s = append(s, 1) }()
go func() { s = append(s, 2) }()

// 3. Структура без лока
type Config struct { Debug bool; Workers int }
var cfg Config
go func() { cfg.Debug = true }()
go func() { _ = cfg.Workers }()
```

## Частые вопросы на собеседованиях

**Q: Что такое data race?**
A: Одновременный доступ к памяти из нескольких горутин, хотя бы одна пишет, без синхронизации.

**Q: Чем data race отличается от race condition?**
A: Data race — concurrent unsynchronized access. Race condition — логическая ошибка из-за порядка операций. Можно иметь race condition без data race (все обращения через mutex, но логика неверна).

**Q: Как обнаружить data race?**
A: `go test -race`, `go run -race`. В CI всегда запускай тесты с -race.
