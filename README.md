# Go Interview Preparation

Структурированный конспект по Go для подготовки к собеседованиям.
От основ до внутренностей рантайма. Актуально для Go 1.26.

## Roadmap

Порядок изучения — от простого к сложному, каждая тема опирается на предыдущие:

```
01-fundamentals → 02-interfaces → 03-errors → 04-concurrency →
05-sync → 06-concurrency-patterns → 07-memory-model → 08-generics →
09-stdlib → 10-internals → 11-testing → 12-profiling →
13-design-patterns → 14-reflect → 15-codegen → 16-postgresql →
17-system-design → 18-production → 19-grpc → 20-kafka →
21-redis → 22-architecture → 23-performance → 24-security →
25-version-history → 26-networking → 27-infrastructure →
28-linux → 29-git → 30-mongodb
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
- [ ] [09. Стандартная библиотека](09-stdlib/) — net/http, encoding/json, io, bytes/strings, sort/slices, time, os, fmt, HTTP/REST паттерны | [задачи](09-stdlib/exercises/)
- [ ] [10. Internals](10-internals/) — слайсы, мапы (classic + Swiss Table), каналы, scheduler (GMP), GC, аллокатор | [задачи](10-internals/exercises/)

### Инструменты и практики

- [ ] [11. Advanced Testing](11-testing-advanced/) — integration tests, mocks/fakes, fuzzing, race detector, synctest | [задачи](11-testing-advanced/exercises/)
- [ ] [12. Profiling](12-profiling/) — benchmarks, pprof, trace, escape analysis, PGO | [задачи](12-profiling/exercises/)
- [ ] [13. Design Patterns](13-design-patterns/) — creational, structural, behavioral, Go-specific | [задачи](13-design-patterns/exercises/)
- [ ] [14. Reflect](14-reflect/) — type/value, struct tags, динамический доступ | [задачи](14-reflect/exercises/)
- [ ] [15. Codegen](15-codegen/) — go generate, stringer, AST, шаблоны

### Базы данных и технологии

- [ ] [16. PostgreSQL](16-postgresql/) — SQL fundamentals, драйверы, запросы, транзакции, миграции, индексы, паттерны | [задачи](16-postgresql/exercises/)
- [ ] [17. System Design](17-system-design/) — подход, расчёты, масштабирование, БД, кэширование, API, messaging, distributed systems, reliability, case studies | [задачи](17-system-design/exercises/)
- [ ] [18. Production Go](18-production/) — graceful shutdown, observability (slog, Prometheus, OpenTelemetry), configuration, resilience | [задачи](18-production/exercises/)
- [ ] [19. gRPC](19-grpc/) — protobuf, типы вызовов, interceptors, error handling, metadata, production | [задачи](19-grpc/exercises/)
- [ ] [20. Kafka](20-kafka/) — библиотеки, producer/consumer паттерны, DLQ, тестирование | [задачи](20-kafka/exercises/)
- [ ] [21. Redis](21-redis/) — типы данных, кэширование, distributed locks, pub/sub, Sentinel/Cluster, паттерны | [задачи](21-redis/exercises/)
- [ ] [22. Architecture](22-architecture/) — Clean Architecture, CQRS, Event Sourcing, DDD, microservices patterns | [задачи](22-architecture/exercises/)
- [ ] [23. Performance](23-performance/) — memory optimization, CPU, I/O, connection pooling, zero-copy | [задачи](23-performance/exercises/)
- [ ] [24. Security](24-security/) — SQL injection, XSS, JWT, OAuth2, session auth, crypto, RBAC/ABAC | [задачи](24-security/exercises/)

### Справочник

- [ ] [25. История версий](25-version-history/) — изменения Go 1.18–1.26

### Фундаментальные знания

- [ ] [26. Networking](26-networking/) — OSI, TCP/IP, HTTP/1.1-3, DNS, load balancing, troubleshooting | [задачи](26-networking/exercises/)
- [ ] [27. Infrastructure](27-infrastructure/) — Docker multi-stage, Kubernetes, CI/CD (GitLab), linting | [задачи](27-infrastructure/exercises/)
- [ ] [28. Linux](28-linux/) — processes, filesystem, memory, networking, I/O, containers, troubleshooting, shell | [задачи](28-linux/exercises/)
- [ ] [29. Git](29-git/) — internals (objects/DAG), branching (merge/rebase), workflows, bisect, hooks | [задачи](29-git/exercises/)
- [ ] [30. MongoDB](30-mongodb/) — драйвер, CRUD, моделирование данных, индексы, агрегации, транзакции | [задачи](30-mongodb/exercises/)

### Практика

Задачи находятся в папке `exercises/` внутри каждого модуля.

- [ ] [Interview Problems](interview-problems/) — классические задачи с собеседований (Two Sum, Merge Intervals, etc.)
