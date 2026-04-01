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

- [ ] [01. Основы](01-fundamentals/) — типы, указатели, структуры, слайсы, мапы, строки, функции | [задачи](01-fundamentals/exercises/)
- [ ] [02. Интерфейсы](02-interfaces/) — неявная реализация, type assertions, iface/eface под капотом | [задачи](02-interfaces/exercises/)
- [ ] [03. Ошибки](03-errors/) — sentinel errors, wrapping, errors.Is/As, стратегии обработки | [задачи](03-errors/exercises/)

### Конкурентность

- [ ] [04. Concurrency](04-concurrency/) — горутины, каналы, select, context | [задачи](04-concurrency/exercises/)
- [ ] [05. Sync](05-sync/) — mutex, waitgroup, once, pool, atomic, errgroup, singleflight | [задачи](05-sync/exercises/)
- [ ] [06. Паттерны конкурентности](06-concurrency-patterns/) — pipeline, fan-out/fan-in, worker pool, semaphore, or-channel | [задачи](06-concurrency-patterns/exercises/)

### Продвинутые темы

- [ ] [07. Memory Model](07-memory-model/) — happens-before, visibility, data races
- [ ] [08. Generics](08-generics/) — type parameters, constraints, паттерны, ограничения | [задачи](08-generics/exercises/)
- [ ] [09. Internals](09-internals/) — слайсы, мапы (classic + Swiss Table), каналы, scheduler (GMP), GC, аллокатор | [задачи](09-internals/exercises/)

### Инструменты и практики

- [ ] [10. Profiling](10-profiling/) — benchmarks, pprof, trace, escape analysis, PGO | [задачи](10-profiling/exercises/)
- [ ] [11. Design Patterns](11-design-patterns/) — creational, structural, behavioral, Go-specific | [задачи](11-design-patterns/exercises/)
- [ ] [12. Reflect](12-reflect/) — type/value, struct tags, динамический доступ | [задачи](12-reflect/exercises/)
- [ ] [13. Codegen](13-codegen/) — go generate, stringer, AST, шаблоны

### Справочник

- [ ] [14. История версий](14-version-history/) — изменения Go 1.18–1.26

### Senior-Level Topics

- [ ] [15. System Design](15-system-design/) — подход, расчёты, масштабирование, БД, кэширование, API, messaging, distributed systems, reliability, case studies | [задачи](15-system-design/exercises/)
- [ ] [16. gRPC](16-grpc/) — protobuf, типы вызовов, interceptors, error handling, metadata, production | [задачи](16-grpc/exercises/)
- [ ] [17. Kafka](17-kafka/) — библиотеки, producer/consumer паттерны, DLQ, тестирование | [задачи](17-kafka/exercises/)
- [ ] [18. Advanced Testing](18-testing-advanced/) — integration tests, mocks/fakes, fuzzing, race detector, synctest | [задачи](18-testing-advanced/exercises/)
- [ ] [19. Production Go](19-production/) — graceful shutdown, observability (slog, Prometheus, OpenTelemetry), configuration, resilience | [задачи](19-production/exercises/)
- [ ] [20. Architecture](20-architecture/) — Clean Architecture, CQRS, Event Sourcing, DDD, microservices patterns | [задачи](20-architecture/exercises/)
- [ ] [21. Security](21-security/) — SQL injection, XSS, JWT, OAuth2, crypto, RBAC/ABAC | [задачи](21-security/exercises/)
- [ ] [22. Performance](22-performance/) — memory optimization, CPU, I/O, connection pooling, zero-copy | [задачи](22-performance/exercises/)
- [ ] [23. Infrastructure](23-infrastructure/) — Docker multi-stage, Kubernetes, CI/CD, linting | [задачи](23-infrastructure/exercises/)

### Фундаментальные знания

- [ ] [24. Networking](24-networking/) — OSI, TCP/IP, HTTP/1.1-3, DNS, load balancing, troubleshooting | [задачи](24-networking/exercises/)
- [ ] [25. Linux](25-linux/) — processes, filesystem, memory, networking, I/O, containers, troubleshooting, shell | [задачи](25-linux/exercises/)
- [ ] [26. Git](26-git/) — internals (objects/DAG), branching (merge/rebase), workflows, bisect, hooks | [задачи](26-git/exercises/)

### Практика

Задачи находятся в папке `exercises/` внутри каждого модуля.

- [ ] [Interview Problems](interview-problems/) — классические задачи с собеседований (Two Sum, Merge Intervals, etc.)
