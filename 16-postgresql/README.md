# 16. PostgreSQL в Go

Практическое руководство по работе с PostgreSQL из Go. Общая теория баз данных (SQL vs NoSQL, индексы, репликация, шардирование) описана в [System Design: Databases](../17-system-design/04-databases.md). Настройка пула соединений — в [Performance: I/O](../22-performance/03-io-networking.md).

## Содержание

0. [SQL Fundamentals](00-sql-fundamentals.md) — типы данных, SELECT, JOIN'ы, агрегации, оконные функции, нормализация, ACID
1. [Драйверы и database/sql](01-drivers-and-stdlib.md) — database/sql vs pgx vs lib/pq, pgxpool
2. [Запросы и сканирование](02-queries-and-scanning.md) — Query/QueryRow/Exec, Scan, sql.Null*, Batch
3. [Транзакции](03-transactions.md) — уровни изоляции, retry, SELECT FOR UPDATE, savepoints
4. [Миграции](04-migrations.md) — goose vs golang-migrate, embed.FS, zero-downtime
5. [Инструменты запросов](05-query-tools.md) — sqlc vs sqlx vs GORM vs squirrel
6. [Индексы и EXPLAIN](06-indexes-and-explain.md) — EXPLAIN ANALYZE, B-tree/GIN/GiST/BRIN
7. [Продвинутый SQL](07-advanced-sql.md) — CTE, оконные функции, JSONB, UPSERT, COPY
8. [Паттерны](08-patterns.md) — soft delete, optimistic locking, cursor pagination

---

## Задачи

Практические задачи по этой теме: [exercises/](exercises/)
