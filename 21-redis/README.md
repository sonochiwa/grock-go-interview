# Redis

Работа с Redis из Go с использованием библиотеки `go-redis/v9`.

## Содержание

| # | Тема | Описание |
|---|------|----------|
| 1 | [Типы данных](01-data-types.md) | Strings, Lists, Sets, Sorted Sets, Hashes, Streams |
| 2 | [Паттерны кэширования](02-caching-patterns.md) | Cache-Aside, Write-Through, Write-Behind, TTL, stampede protection |
| 3 | [Распределённые блокировки](03-distributed-lock.md) | SETNX, Redlock, redsync, сравнение с etcd/ZooKeeper |
| 4 | [Pub/Sub](04-pub-sub.md) | PUBLISH/SUBSCRIBE, паттерны, ограничения, сравнение с Kafka |
| 5 | [Конфигурация и эксплуатация](05-configuration.md) | Eviction policies, persistence, Sentinel, Cluster, пул соединений |
| 6 | [Паттерны использования](06-patterns.md) | Rate limiter, сессии, очереди, pipeline, транзакции, Lua-скрипты |
