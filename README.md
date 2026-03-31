# Go Interview Preparation

Структурированный конспект по Go для подготовки к собеседованиям.
От основ до внутренностей рантайма. Актуально для Go 1.26.

## Roadmap

Порядок изучения — от простого к сложному, каждая тема опирается на предыдущие:

```
01-fundamentals → 02-interfaces → 03-errors → 04-concurrency →
05-sync → 06-concurrency-patterns → 07-memory-model → 08-generics →
09-internals → 10-profiling → 11-design-patterns → 12-reflect →
13-codegen → 14-version-history → 15-system-design → 16-grpc →
17-kafka → 18-testing → 19-production → 20-architecture →
21-security → 22-performance → 23-infrastructure →
24-networking → 25-linux → 26-git
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

### Senior-Level Topics

- [ ] [15. System Design](15-system-design/) — подход, расчёты, масштабирование, БД, кэширование, API, messaging, distributed systems, reliability, case studies
- [ ] [16. gRPC](16-grpc/) — protobuf, типы вызовов, interceptors, error handling, metadata, production
- [ ] [17. Kafka](17-kafka/) — библиотеки, producer/consumer паттерны, DLQ, тестирование
- [ ] [18. Advanced Testing](18-testing-advanced/) — integration tests, mocks/fakes, fuzzing, race detector, synctest
- [ ] [19. Production Go](19-production/) — graceful shutdown, observability (slog, Prometheus, OpenTelemetry), configuration, resilience
- [ ] [20. Architecture](20-architecture/) — Clean Architecture, CQRS, Event Sourcing, DDD, microservices patterns
- [ ] [21. Security](21-security/) — SQL injection, XSS, JWT, OAuth2, crypto, RBAC/ABAC
- [ ] [22. Performance](22-performance/) — memory optimization, CPU, I/O, connection pooling, zero-copy
- [ ] [23. Infrastructure](23-infrastructure/) — Docker multi-stage, Kubernetes, CI/CD, linting

### Фундаментальные знания

- [ ] [24. Networking](24-networking/) — OSI, TCP/IP, HTTP/1.1-3, DNS, load balancing, troubleshooting
- [ ] [25. Linux](25-linux/) — processes, filesystem, memory, networking, I/O, containers, troubleshooting, shell
- [ ] [26. Git](26-git/) — internals (objects/DAG), branching (merge/rebase), workflows, bisect, hooks

### Практика

- [ ] [Задачи](exercises/) — 1000+ задач по всем темам (easy/medium/hard)
