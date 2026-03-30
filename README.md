# Go Interview Preparation

Структурированный конспект по Go для подготовки к собеседованиям.
От основ до внутренностей рантайма. Актуально для Go 1.26.

## Roadmap

Порядок изучения — от простого к сложному, каждая тема опирается на предыдущие:

```
01-fundamentals → 02-interfaces → 03-errors → 04-concurrency →
05-sync → 06-concurrency-patterns → 07-memory-model → 08-generics →
09-internals → 10-profiling → 11-design-patterns → 12-reflect →
13-codegen → 14-version-history
```

## Содержание

### Основы языка

- [ ] [01. Основы](01-fundamentals/) — типы, указатели, структуры, слайсы, мапы, строки, функции
- [ ] [02. Интерфейсы](02-interfaces/) — неявная реализация, type assertions, iface/eface под капотом
- [ ] [03. Ошибки](03-errors/) — sentinel errors, wrapping, errors.Is/As, стратегии обработки

### Конкурентность

- [ ] [04. Concurrency](04-concurrency/) — горутины, каналы, select, context
- [ ] [05. Sync](05-sync/) — mutex, waitgroup, once, pool, atomic, errgroup, singleflight
- [ ] [06. Паттерны конкурентности](06-concurrency-patterns/) — pipeline, fan-out/fan-in, worker pool, semaphore, or-channel

### Продвинутые темы

- [ ] [07. Memory Model](07-memory-model/) — happens-before, visibility, data races
- [ ] [08. Generics](08-generics/) — type parameters, constraints, паттерны, ограничения
- [ ] [09. Internals](09-internals/) — слайсы, мапы (classic + Swiss Table), каналы, scheduler (GMP), GC, аллокатор

### Инструменты и практики

- [ ] [10. Profiling](10-profiling/) — benchmarks, pprof, trace, escape analysis, PGO
- [ ] [11. Design Patterns](11-design-patterns/) — creational, structural, behavioral, Go-specific
- [ ] [12. Reflect](12-reflect/) — type/value, struct tags, динамический доступ
- [ ] [13. Codegen](13-codegen/) — go generate, stringer, AST, шаблоны

### Справочник

- [ ] [14. История версий](14-version-history/) — изменения Go 1.18–1.26

### Практика

- [ ] [Задачи](exercises/) — 1000+ задач по всем темам (easy/medium/hard)
