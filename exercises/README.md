# Практические задачи

Задачи для закрепления теории. Каждая задача имеет: условие, стартовый код, тесты, эталонное решение.

## Структура

```
exercises/<topic>/<difficulty>/<task-name>/
├── task.md           # Условие задачи
├── main.go           # Стартовый код с TODO
├── main_test.go      # Тесты для проверки
└── solution/
    └── main.go       # Эталонное решение
```

## Уровни сложности

- **easy** — базовые концепции, 5-15 мин
- **medium** — комбинация концепций, 15-30 мин
- **hard** — продвинутые задачи, 30-60 мин

## Разделы

| Раздел | Easy | Medium | Hard | Темы |
|--------|------|--------|------|------|
| [01-fundamentals](01-fundamentals/) | 2 | 1 | 1 | Reverse slice, unique, sort, LRU cache |
| [02-interfaces](02-interfaces/) | 1 | 1 | — | Stringer, Shape calculator |
| [03-errors](03-errors/) | 1 | 1 | — | Custom error, error wrapping chain |
| [04-concurrency](04-concurrency/) | 1 | 1 | 1 | Ping-pong, parallel fetch, rate limiter |
| [05-sync](05-sync/) | 1 | 1 | — | Safe counter (mutex+atomic), TTL cache |
| [06-concurrency-patterns](06-concurrency-patterns/) | 1 | 1 | 1 | Pipeline, fan-out/fan-in, worker pool |
| [07-generics](07-generics/) | 1 | 1 | — | Min/Max, generic Set |
| [09-profiling](09-profiling/) | — | 1 | — | Optimize string concat + allocs |
| [10-design-patterns](10-design-patterns/) | 1 | — | — | Functional options |
| [11-reflect-codegen](11-reflect-codegen/) | — | 1 | — | Struct validator with tags |
| [interview-problems](interview-problems/) | 2 | 1 | 2 | Two sum, palindrome, merge intervals, sharded map, graceful server |
| **Итого** | **11** | **10** | **4** | **25 задач** |

## Как решать

1. Прочитай `task.md` — условие задачи
2. Открой `main.go` — найди `// TODO` комментарии
3. Реализуй решение
4. Запусти тесты: `go test ./...`
5. Сравни с `solution/main.go` если застрял

## Запуск тестов

```bash
# Все тесты раздела
go test ./exercises/04-concurrency/...

# Конкретная задача
go test ./exercises/04-concurrency/easy/01-ping-pong/

# С race detector
go test -race ./exercises/...
```

> Задачи будут добавляться по мере изучения тем. Попроси Claude сгенерировать задачи по конкретной теме!
