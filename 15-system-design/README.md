# 15. System Design

Самый весомый раздел на senior собеседовании. 30-40 минут, оценивается не "правильный ответ", а ход мысли, trade-offs, и умение задавать правильные вопросы.

## План действий на System Design секции

```
0-2 мин   → Уточнение требований (ОБЯЗАТЕЛЬНО!)
2-5 мин   → Оценки нагрузки (back-of-envelope)
5-10 мин  → High-level design (блок-схема)
10-25 мин → Deep dive в компоненты
25-35 мин → Масштабирование, bottlenecks, trade-offs
35-40 мин → Итоги, что бы улучшил с бОльшим временем
```

## Содержание

### Фреймворк
1. [Подход к System Design](01-approach.md) — пошаговый план, что говорить, чего избегать
2. [Back-of-Envelope](02-back-of-envelope.md) — расчёты нагрузки, IOPS, пропускная способность, порядки

### Фундамент
3. [Масштабирование](03-scalability.md) — горизонтальное/вертикальное, load balancing, auto-scaling
4. [Базы данных](04-databases.md) — SQL vs NoSQL, индексы, шардинг, репликация, партиционирование
5. [Кэширование](05-caching.md) — стратегии, Redis, invalidation, cache stampede
6. [CDN и Storage](06-cdn-storage.md) — S3, CDN, blob storage

### Коммуникация
7. [API Design](07-api-design.md) — REST, GraphQL, WebSocket, SSE, long polling
8. [Messaging](08-messaging.md) — очереди vs стримы, паттерны, выбор
9. [Distributed Systems](09-distributed-systems.md) — CAP, consistency, consensus, partitioning

### Паттерны
10. [Reliability Patterns](10-reliability.md) — circuit breaker, retry, bulkhead, timeout
11. [Data Patterns](11-data-patterns.md) — CQRS, Event Sourcing, Outbox, Saga

### Кейсы
12. [Case Studies](12-case-studies.md) — Rate Limiter, URL Shortener, Chat, Feed, Notification
