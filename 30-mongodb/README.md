# 29. MongoDB в Go

Работа с MongoDB из Go через официальный драйвер `go.mongodb.org/mongo-driver`.

## Содержание

| #  | Файл | Темы |
|----|------|------|
| 01 | [Драйвер и подключение](01-driver-and-connection.md) | mongo.Client, connection string, connection pool, context |
| 02 | [CRUD операции](02-crud.md) | InsertOne/Many, Find, FindOne, UpdateOne/Many, DeleteOne/Many, фильтры (bson.M/D) |
| 03 | [Моделирование данных](03-data-modeling.md) | embedding vs referencing, schema design patterns, bson struct tags |
| 04 | [Индексы](04-indexes.md) | single field, compound, multikey, text, TTL, unique, explain |
| 05 | [Агрегации](05-aggregation.md) | pipeline stages ($match, $group, $project, $lookup, $unwind), cursor |
| 06 | [Транзакции](06-transactions.md) | sessions, multi-document transactions, write concern, read concern |
| 07 | [Паттерны в Go](07-patterns.md) | repository pattern, soft delete, pagination, change streams, тестирование с testcontainers |

---

## Задачи

Практические задачи по этой теме: [exercises/](exercises/)
