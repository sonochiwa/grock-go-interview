# API Design

## REST

```
GET    /users          → список пользователей
GET    /users/{id}     → один пользователь
POST   /users          → создать
PUT    /users/{id}     → полное обновление
PATCH  /users/{id}     → частичное обновление
DELETE /users/{id}     → удалить

Статус коды:
  200 OK             → успех
  201 Created        → создано
  204 No Content     → удалено (нет тела)
  400 Bad Request    → невалидный запрос
  401 Unauthorized   → не аутентифицирован
  403 Forbidden      → нет прав
  404 Not Found      → не найдено
  409 Conflict       → конфликт (duplicate)
  429 Too Many Req   → rate limited
  500 Internal Error → ошибка сервера
  503 Service Unavail → сервис недоступен
```

### Пагинация

```
Offset-based (простой, но медленный для больших offset):
  GET /users?offset=100&limit=20

Cursor-based (быстрый, но нельзя "перепрыгнуть"):
  GET /users?cursor=eyJpZCI6MTAwfQ==&limit=20
  Response: { data: [...], next_cursor: "eyJpZCI6MTIwfQ==" }

Keyset (page token):
  GET /users?after_id=100&limit=20
  WHERE id > 100 ORDER BY id LIMIT 20 — использует индекс!
```

### Versioning

```
URL path:     /api/v1/users  (рекомендуется)
Header:       Accept: application/vnd.api.v1+json
Query param:  /users?version=1
```

## REST vs gRPC vs GraphQL

| | REST | gRPC | GraphQL |
|---|---|---|---|
| Формат | JSON | Protobuf (binary) | JSON |
| Transport | HTTP/1.1, HTTP/2 | HTTP/2 | HTTP |
| Скорость | Средняя | Быстрая (5-10x) | Средняя |
| Schema | OpenAPI (optional) | .proto (обязательно) | SDL (обязательно) |
| Streaming | SSE, WebSocket | Native bidirectional | Subscriptions |
| Browser | Нативно | Через grpc-web | Нативно |
| Когда | Public API, CRUD | Microservices, high perf | Сложные клиенты, mobile |

## WebSocket

```
Когда: real-time bidirectional (chat, игры, live updates)
Когда НЕ: request-response, редкие обновления

Масштабирование:
  - WebSocket connection = stateful
  - Sticky sessions или shared state (Redis pub/sub)
  - Каждый сервер хранит свои connections
  - Broadcast: publish в Redis → все серверы полу��ают → рассылают своим clients
```

## SSE (Server-Sent Events)

```
Когда: server → client stream (уведомления, live feed)
  - Проще WebSocket (unidirectional)
  - Автоматический reconnect
  - Работает через обычный HTTP
  - Не подходит для client → server

Пример:
  Client: GET /events (Accept: text/event-stream)
  Server: data: {"type":"notification","msg":"hello"}\n\n
```

## Idempotency

```
Idempotent: повторный вызов даёт тот же результат
  GET    — всегда idempotent
  PUT    — idempotent (полная замена)
  DELETE — idempotent

НЕ idempotent:
  POST   — создаёт новый ресурс каждый раз

Решение для POST:
  Idempotency-Key: <UUID>
  Сервер хранит {key → response} в Redis (TTL 24h)
  Повторный POST с тем ж�� key → вернуть сохранённый response
```

## Частые вопросы

**Q: Как выбрать между REST и gRPC?**
A: Public API или browser → REST. Microservices, high throughput, streaming → gRPC. Mobile с разными нуждами → GraphQL.

**Q: Offset vs cursor pagination?**
A: Cursor для feed/timeline (беско��ечный скролл). Offset для admin панелей (нужен "страница 5").
