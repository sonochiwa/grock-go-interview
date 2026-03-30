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

| Раздел | Темы |
|--------|------|
| [01-fundamentals](01-fundamentals/) | Слайсы, мапы, строки, указатели |
| [02-interfaces](02-interfaces/) | Реализация интерфейсов, type assertions |
| [03-errors](03-errors/) | Custom errors, wrapping, errors.Is/As |
| [04-concurrency](04-concurrency/) | Горутины, каналы, select |
| [05-sync](05-sync/) | Mutex, WaitGroup, atomic |
| [06-concurrency-patterns](06-concurrency-patterns/) | Pipeline, fan-out, worker pool |
| [07-generics](07-generics/) | Generic функции и типы |
| [08-internals](08-internals/) | Escape analysis, бенчмарки |
| [09-profiling](09-profiling/) | pprof, оптимизация |
| [10-design-patterns](10-design-patterns/) | Паттерны на Go |
| [11-reflect-codegen](11-reflect-codegen/) | Reflect, кодогенерация |
| [interview-problems](interview-problems/) | Классические задачи с собесов |

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
