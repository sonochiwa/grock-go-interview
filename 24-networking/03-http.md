# HTTP

## HTTP/1.1

```
Особенности:
  - Текстовый протокол
  - Keep-Alive по умолчанию (переиспользование TCP)
  - Pipelining (несколько запросов без ожидания ответа) — но мало кто поддерживает
  - HOL blocking: запросы обрабатываются последовательно
  - Chunked transfer encoding (streaming)

Проблема: 1 TCP connection = 1 запрос за раз (на практике)
  Решение: браузеры открывают 6 TCP connections на домен
  → 6 параллельных запросов
  → 6 × TCP handshake = overhead
```

## HTTP/2

```
Ключевые фичи:
  1. Binary protocol (не текстовый)
  2. Multiplexing: множество запросов на ОДНОМ TCP соединении
  3. Header compression (HPACK)
  4. Server Push (сервер отправляет ресурсы без запроса)
  5. Stream prioritization

Multiplexing:
  HTTP/1.1: [req1]───[res1]───[req2]───[res2]  (последовательно)
  HTTP/2:   [req1][req2] ──── [res2][res1]      (параллельно)

  Каждый запрос = stream с уникальным ID
  Streams мультиплексируются в одном TCP

Frames:
  HEADERS frame → запрос/ответ headers
  DATA frame → body
  SETTINGS frame → конфигурация
  GOAWAY frame → graceful shutdown

Проблема:
  TCP HOL blocking остаётся!
  Потеря 1 TCP пакета → блокировка ВСЕХ streams
  → HTTP/3 решает это через QUIC/UDP
```

## HTTP/3 (QUIC)

```
QUIC = UDP + reliability + encryption (built-in TLS 1.3)

Преимущества:
  1. Нет TCP HOL blocking — потеря в одном stream не блокирует другие
  2. 0-RTT connection (при повторном подключении)
  3. Connection migration (смена IP/сети без разрыва)
  4. Встроенное шифрование (TLS 1.3)

Latency сравнение:
  HTTP/1.1: TCP handshake (1 RTT) + TLS handshake (2 RTT) = 3 RTT
  HTTP/2:   TCP handshake (1 RTT) + TLS handshake (1 RTT TLS 1.3) = 2 RTT
  HTTP/3:   QUIC handshake (1 RTT, includes TLS) = 1 RTT
  HTTP/3:   0-RTT resumption = 0 RTT (!)

  1 RTT at 100ms = 100ms saved per connection
```

## HTTPS / TLS

### TLS 1.3 Handshake (1-RTT)

```
Client                           Server
  │                                │
  │── ClientHello ────────────────→│
  │   (supported ciphers,          │
  │    key share,                  │
  │    SNI: example.com)           │
  │                                │
  │←── ServerHello ────────────────│
  │    (chosen cipher,             │
  │     key share,                 │
  │     certificate,               │
  │     certificate verify,        │
  │     finished)                  │
  │                                │
  │── Finished ───────────────────→│
  │                                │
  │←─── Application Data ─────────│
  │                                │
  1 RTT total (vs 2 RTT in TLS 1.2)

TLS 1.3 improvements:
  - Убраны устаревшие cipher suites (RC4, 3DES, SHA-1)
  - Только AEAD ciphers (AES-GCM, ChaCha20-Poly1305)
  - Forward secrecy обязателен (ECDHE)
  - 0-RTT resumption (PSK)
  - Encrypted handshake (certificate hidden from observer)
```

### Сертификаты

```
Certificate chain:
  Root CA → Intermediate CA → Server Certificate

  Root CA: предустановлен в ОС/браузере (trust store)
  Intermediate: подписан Root, подписывает server cert
  Server cert: содержит public key сервера + domain name

Проверка:
  1. Получить server cert
  2. Проверить подпись через intermediate cert
  3. Проверить подпись intermediate через root CA
  4. Проверить domain name (SAN/CN)
  5. Проверить expiration
  6. Проверить revocation (CRL/OCSP)

SNI (Server Name Indication):
  В ClientHello → домен (незашифрованный в TLS 1.2!)
  Позволяет одному IP обслуживать несколько HTTPS доменов
  ECH (Encrypted Client Hello) — скрывает SNI (draft)
```

## HTTP Headers

```
Важные request headers:
  Host: example.com              — обязательный в HTTP/1.1
  Authorization: Bearer <token>  — авторизация
  Content-Type: application/json — тип тела запроса
  Accept: application/json       — ожидаемый тип ответа
  Accept-Encoding: gzip, br      — сжатие
  User-Agent: ...                — клиент
  X-Request-ID: uuid             — трассировка
  If-None-Match: "etag"          — условный запрос

Важные response headers:
  Content-Type: application/json
  Cache-Control: max-age=3600, public
  ETag: "abc123"                 — версия ресурса
  Set-Cookie: sid=xxx; HttpOnly; Secure; SameSite=Strict
  Strict-Transport-Security: max-age=31536000  — HSTS
  X-Content-Type-Options: nosniff
  Content-Encoding: gzip
```

## Кэширование

```
Cache-Control:
  no-store       — не кэшировать вообще
  no-cache       — кэшировать, но всегда revalidate
  public         — кэшировать на CDN и клиенте
  private        — только на клиенте (не CDN)
  max-age=3600   — кэш валиден 1 час
  s-maxage=3600  — TTL для shared caches (CDN)
  immutable      — никогда не revalidate (для hashed URLs)

Revalidation:
  ETag + If-None-Match:
    Server: ETag: "abc"
    Client: If-None-Match: "abc"
    Server: 304 Not Modified (без body!)

  Last-Modified + If-Modified-Since:
    Server: Last-Modified: Tue, 01 Jan 2025 00:00:00 GMT
    Client: If-Modified-Since: Tue, 01 Jan 2025 00:00:00 GMT
    Server: 304 Not Modified

Стратегия для SPA:
  index.html      → Cache-Control: no-cache (всегда проверять)
  app.a1b2c3.js   → Cache-Control: max-age=31536000, immutable
  style.d4e5f6.css → Cache-Control: max-age=31536000, immutable
```

## HTTP/1.1 vs 2 vs 3

```
│ Feature            │ HTTP/1.1    │ HTTP/2      │ HTTP/3        │
├────────────────────┼─────────────┼─────────────┼───────────────┤
│ Transport          │ TCP         │ TCP         │ QUIC (UDP)    │
│ Encoding           │ Text        │ Binary      │ Binary        │
│ Multiplexing       │ Нет         │ Да          │ Да            │
│ Header compression │ Нет         │ HPACK       │ QPACK         │
│ Server push        │ Нет         │ Да          │ Да            │
│ TCP HOL blocking   │ Да          │ Да          │ Нет           │
│ TLS                │ Optional    │ Де-факто    │ Встроен       │
│ Conn setup         │ 2-3 RTT    │ 2 RTT       │ 1 RTT / 0-RTT│
│ Conn migration     │ Нет         │ Нет         │ Да            │
```
