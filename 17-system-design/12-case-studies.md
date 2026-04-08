# System Design: Case Studies

## Шаблон ответа на SD интервью

```
1. Clarify Requirements (3-5 мин)
   - Functional requirements (что система делает)
   - Non-functional requirements (scale, latency, consistency)
   - Constraints (budget, team, timeline)

2. Back-of-Envelope (3-5 мин)
   - DAU, RPS, storage, bandwidth

3. High-Level Design (10-15 мин)
   - API design
   - Data model
   - Core components diagram

4. Deep Dive (15-20 мин)
   - Scaling bottlenecks
   - Data partitioning
   - Caching strategy
   - Failure scenarios
```

## URL Shortener (TinyURL)

```
Requirements:
  - Создание short URL → long URL mapping
  - Redirect short URL → long URL
  - Custom aliases (optional)
  - Analytics (click count)
  - 100M URLs/день создание, 10:1 read/write

Расчёты:
  - Write: 100M/86400 ≈ 1200 URL/s
  - Read: 12000 redirects/s
  - Storage: 100M × 365 × 5 years × 500 bytes ≈ 90 TB

API:
  POST /api/v1/shorten { long_url, custom_alias? } → { short_url }
  GET /{short_code} → 301/302 redirect

Генерация short code (7 символов, base62):
  1. Counter-based: auto-increment → base62 encode
     + Уникальный, короткий
     - Предсказуемый, single point
  2. Hash-based: MD5(long_url)[:7]
     + Одинаковый URL → одинаковый code
     - Коллизии
  3. Pre-generated: заранее сгенерировать коды в key range table
     + Быстро, нет коллизий
     - Нужна отдельная таблица
  → Snowflake ID → base62 (рекомендуется)

Data Model:
  urls:
    short_code  VARCHAR(7) PRIMARY KEY
    long_url    TEXT
    user_id     INT
    created_at  TIMESTAMP
    expires_at  TIMESTAMP
    click_count INT

Architecture:
  [Client] → [LB] → [API Servers] → [Cache (Redis)] → [DB (PostgreSQL)]
                                         ↑
                                    Read: cache-aside
                                    Write: write-through

Caching:
  - Hot URLs (20% URLs = 80% traffic) → Redis
  - Cache-aside: read cache first, miss → DB → cache
  - TTL: 24 hours

Scaling:
  - DB: sharding by short_code (hash)
  - Read replicas для analytics queries
  - CDN для redirect (301 с Cache-Control)
```

## Rate Limiter

```
Requirements:
  - Ограничить кол-во запросов per user/IP/API key
  - Distributed (несколько серверов)
  - Low latency (не замедлять запросы)
  - Configurable rules

Architecture:
  [Client] → [Rate Limiter Middleware] → [API Server]
                      ↓
              [Redis (counters)]

Алгоритм: Sliding Window Counter (Redis)
  Key: rate:{user_id}:{minute}
  INCR key → если > limit → reject (429)
  EXPIRE key 60

Rules хранятся в config:
  {
    "api": "/api/v1/messages",
    "limits": [
      {"window": "1m", "max": 60},
      {"window": "1h", "max": 1000},
      {"window": "1d", "max": 10000}
    ]
  }

Response headers:
  X-RateLimit-Limit: 60
  X-RateLimit-Remaining: 45
  X-RateLimit-Reset: 1234567890 (unix timestamp)
  Retry-After: 30 (при 429)

Distributed:
  - Centralized Redis → single source of truth
  - Race condition: INCR atomic в Redis → OK
  - Redis cluster → consistent hashing по user_id
```

## Chat System (WhatsApp/Telegram)

```
Requirements:
  - 1:1 и group messages
  - Online/offline status
  - Read receipts
  - Push notifications для offline users
  - 500M DAU, 40 msg/day average

Расчёты:
  - Messages: 500M × 40 = 20B msg/day ≈ 230K msg/s
  - Storage: 20B × 100 bytes = 2TB/day

Architecture:
  [Client] ←WebSocket→ [Chat Server] → [Message Queue (Kafka)]
                             ↓                    ↓
                     [Session Service]    [Message Store (Cassandra)]
                     [Presence Service]   [Push Notification Service]

Connection Management:
  - WebSocket для real-time
  - Каждый chat server хранит свои connections
  - Session Service (Redis): user_id → chat_server_id
  - Client reconnect → новый chat server → обновить session

Message Flow (1:1):
  1. User A → WebSocket → Chat Server 1
  2. Chat Server 1 → Kafka topic "messages"
  3. Chat Server 1 → Session Service: "где User B?"
  4. User B online → Chat Server 2 → WebSocket → User B
  5. User B offline → Push Notification Service

Message Flow (Group):
  1. User A → Chat Server → Kafka
  2. Fan-out: для каждого member группы
     - Online → через WebSocket
     - Offline → push notification
  3. Small groups (< 100): fan-out on write (отправить каждому)
  4. Large groups (> 100): fan-out on read (читать при открытии)

Data Model (Cassandra):
  messages_by_chat:
    chat_id     UUID    -- partition key
    message_id  TIMEUUID -- clustering key (sorted by time)
    sender_id   UUID
    content     TEXT
    created_at  TIMESTAMP

  → Все сообщения чата в одной partition → быстрое чтение

Presence (Online/Offline):
  - Heartbeat каждые 30s → Redis SET user:123:last_seen timestamp
  - Online: last_seen < 60s
  - При открытии чата: subscribe to presence updates
  - Для групп: poll, не push (слишком много updates)
```

## News Feed (Twitter/Instagram)

```
Requirements:
  - Post tweet/photo
  - View home feed (aggregated from followed users)
  - Follow/unfollow
  - 300M DAU, average 200 follows

Расчёты:
  - Write: 300M × 2 posts/day = 600M posts/day ≈ 7K/s
  - Read: 300M × 10 feed views/day = 3B/day ≈ 35K/s

Два подхода к feed:

Fan-out on Write (push):
  User posts → для каждого follower → добавить в их feed cache
  + Чтение быстрое (feed уже готов)
  - Celebrity problem: 10M followers × каждый пост = 10M записей
  - Задержка при публикации

Fan-out on Read (pull):
  User opens feed → собрать посты от всех followed → merge → sort
  + Быстрая публикация
  - Медленное чтение (N запросов к N followed users)

Гибридный подход (рекомендуется):
  - Обычные пользователи → fan-out on write
  - Celebrities (> 10K followers) → fan-out on read
  - Feed = pre-computed cache + merge celebrity posts on read

Architecture:
  [Post Service] → [Fan-out Service] → [Feed Cache (Redis)]
                        ↓
                  [Kafka] → для celebrities
                        ↓
  [Feed Service] ← merge → [Feed Cache] + [Post Service (celebrity)]

Feed Cache (Redis sorted set):
  ZADD feed:{user_id} {timestamp} {post_id}
  ZREVRANGE feed:{user_id} 0 19  → top 20 posts
  Хранить только последние 200-500 постов в cache
```

## Notification System

```
Requirements:
  - Push, SMS, Email
  - Prioritization (urgent vs marketing)
  - Rate limiting (не спамить пользователя)
  - Template management
  - 10M notifications/day

Architecture:
  [Services] → [Notification API] → [Kafka (by priority)]
                                          ↓
                                   [Notification Workers]
                                     ↓        ↓        ↓
                                  [Push]   [SMS]    [Email]
                                  (FCM)   (Twilio)  (SES)

Priority queues:
  - high: OTP, security alerts, order updates
  - medium: social notifications
  - low: marketing, recommendations
  → Разные Kafka topics или partition + consumer priority

Deduplication:
  - event_id в Redis (TTL 24h)
  - Тот же event_id → skip

Rate limiting per user:
  - Max 3 push/hour, 10 email/day
  - Aggregation: "5 people liked your post" вместо 5 отдельных

Template:
  "Hello {{.Name}}, your order {{.OrderID}} has been shipped"
  Хранение в DB, кэш в memory
```

## Distributed File Storage (Google Drive)

```
Requirements:
  - Upload/download файлов
  - Sync across devices
  - File versioning
  - Sharing and permissions
  - 50M users, 10M DAU

Architecture:
  [Client] → [API Gateway] → [Metadata Service] → [Metadata DB (PostgreSQL)]
                    ↓
              [Block Service] → [Object Storage (S3)]
                    ↓
              [Sync Service] → [Notification (WebSocket)]

Upload flow:
  1. Client splits file into blocks (4MB chunks)
  2. Client → Metadata Service: "uploading file X, N blocks"
  3. Client → Block Service: upload each block (deduplicated)
  4. Block Service → S3: store blocks
  5. Metadata Service: update file record

Block-level dedup:
  - Hash each block (SHA-256)
  - If hash exists → don't upload (same content)
  - 50%+ savings для похожих файлов

Sync:
  - Long polling или WebSocket для real-time sync
  - Client хранит local sync state (last_sync_timestamp)
  - Conflict resolution: last-writer-wins или keep both

Data Model:
  files:
    file_id, name, parent_id, user_id, is_folder,
    latest_version, created_at, updated_at

  file_versions:
    version_id, file_id, version_num, size,
    checksum, created_at

  blocks:
    block_id, hash, size, s3_key

  file_blocks:
    version_id, block_id, block_order
```

## Советы для SD интервью

```
1. НЕ прыгать сразу в детали — начни с requirements
2. Считай числа вслух (back-of-envelope)
3. Начни с simple design → добавляй complexity по необходимости
4. Называй trade-offs: "я выбрал X потому что Y, trade-off — Z"
5. Используй правильные термины (CAP, ACID, sharding, replication)
6. Рисуй диаграммы (даже если словесные в тексте)
7. Не пытайся решить ВСЁ — фокусируйся на core problem
8. Спрашивай интервьюера: "хотите deep dive в X или Y?"

Красные флаги (что НЕ делать):
  ❌ Single point of failure без объяснения
  ❌ Игнорировать масштаб (нужен sharding, а ты на одном сервере)
  ❌ Over-engineering (Kafka для 100 msg/day)
  ❌ Нет мониторинга/observability
  ❌ Нет обработки ошибок
  ❌ Говорить "я не знаю" без попытки рассуждать
```
