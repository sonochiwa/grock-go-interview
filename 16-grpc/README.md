# 16. gRPC

gRPC — высокопроизводительный RPC фреймворк от Google. Использует Protocol Buffers (protobuf) для сериализации и HTTP/2 для транспорта. Стандарт для коммуникации между микросервисами.

## Содержание

1. [Основы и Protobuf](01-basics-protobuf.md) — proto файлы, типы данных, кодогенерация
2. [Типы вызовов](02-call-types.md) — Unary, Server/Client/Bidirectional streaming
3. [Interceptors и Middleware](03-interceptors.md) — Unary/Stream interceptors, chaining
4. [Error Handling](04-error-handling.md) — Status codes, error details, rich errors
5. [Метаданные и Deadline](05-metadata-deadline.md) — Headers, context propagation
6. [Production](06-production.md) — TLS, health check, reflection, load balancing


---

## Задачи

Практические задачи по этой теме: [exercises/](exercises/)
