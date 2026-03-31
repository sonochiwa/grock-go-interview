# TCP/IP

## TCP (Transmission Control Protocol)

### 3-Way Handshake (установка соединения)

```
Client                    Server
  │                         │
  │──── SYN (seq=x) ──────→│  1. Client отправляет SYN
  │                         │
  │←── SYN-ACK (seq=y,      │  2. Server отвечает SYN-ACK
  │     ack=x+1) ──────────│
  │                         │
  │──── ACK (seq=x+1,       │  3. Client подтверждает
  │     ack=y+1) ──────────→│
  │                         │
  │    Connection ESTABLISHED│
```

### 4-Way Termination (закрытие)

```
Client                    Server
  │                         │
  │──── FIN ───────────────→│  1. Client хочет закрыть
  │←── ACK ─────────────────│  2. Server подтверждает FIN
  │                         │     (может ещё слать данные)
  │←── FIN ─────────────────│  3. Server тоже закрывает
  │──── ACK ───────────────→│  4. Client подтверждает
  │                         │
  │   TIME_WAIT (2×MSL)     │  Client ждёт 2×MSL (60-120s)
```

### TCP State Machine

```
                    CLOSED
                      │
              ┌───────┴────────┐
          (listen)          (connect)
              │                │
           LISTEN          SYN_SENT ──→ ESTABLISHED
              │                              │
         SYN_RCVD ──→ ESTABLISHED         (close)
                          │                  │
                       (close)           FIN_WAIT_1
                          │                  │
                     CLOSE_WAIT          FIN_WAIT_2
                          │                  │
                      LAST_ACK           TIME_WAIT
                          │                  │
                       CLOSED             CLOSED

Важные состояния:
  ESTABLISHED — активное соединение
  TIME_WAIT — ждёт 2×MSL после закрытия (предотвращает путаницу с поздними пакетами)
  CLOSE_WAIT — получили FIN, но ещё не закрыли (утечка если не закрыть!)

Проблема TIME_WAIT:
  Много коротких соединений → тысячи сокетов в TIME_WAIT → исчерпание портов
  Решения:
    - net.ipv4.tcp_tw_reuse = 1
    - Connection pooling (Keep-Alive)
    - SO_REUSEADDR
```

### Flow Control (оконное управление)

```
Sliding Window:
  Receiver сообщает: "Моё окно = 64KB" (сколько данных могу принять)
  Sender не отправляет больше чем window size

  Window = 0 → receiver перегружен → sender ждёт (Zero Window Probe)

  [Sent & ACKed] [Sent, not ACKed] [Can send] [Cannot send yet]
                  ←──── Window ────→
```

### Congestion Control

```
Проблема: если все sender'ы отправляют на полную → сеть перегружена → packet loss

Алгоритмы (CUBIC — default в Linux):

1. Slow Start:
   cwnd (congestion window) начинается с 1 MSS
   Каждый ACK → cwnd × 2 (экспоненциальный рост)
   До ssthresh (slow start threshold)

   cwnd: 1 → 2 → 4 → 8 → 16 → ... → ssthresh

2. Congestion Avoidance:
   После ssthresh → линейный рост (+1 MSS per RTT)
   cwnd: 16 → 17 → 18 → 19 → ...

3. При потере пакета:
   Fast Retransmit: 3 duplicate ACKs → retransmit (без таймаута)
   Fast Recovery: ssthresh = cwnd/2, cwnd = ssthresh
   Timeout: ssthresh = cwnd/2, cwnd = 1 (начать заново)

   ┌─────────────────────────────────────────────┐
   │ cwnd                                        │
   │   ╱\                ╱──                      │
   │  ╱  \              ╱                         │
   │ ╱    \   ╱\       ╱                          │
   │╱      ╱   \     ╱  congestion avoidance      │
   │ slow ╱     ╱  ╱                              │
   │start╱     ╱ ╱     (линейный рост)            │
   │    ╱     ╱╱                                  │
   │   ╱   loss    loss                           │
   └─────────────────────────────────────────────┘

BBR (Google, 2016):
  Не реагирует на packet loss, а измеряет bandwidth и RTT
  Лучше для high-latency сетей (CDN, intercontinental)
  Используется в GCP, YouTube
```

### TCP Header

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
├─────────────────────────┼─────────────────────────┤
│       Source Port        │    Destination Port     │ 4 bytes
├─────────────────────────────────────────────────────┤
│                  Sequence Number                    │ 4 bytes
├─────────────────────────────────────────────────────┤
│               Acknowledgment Number                 │ 4 bytes
├──────┼──────┼─┼─┼─┼─┼─┼─┼─────────────────────────┤
│Offset│Reserv│C│E│U│A│P│R│S│F│     Window Size      │ 4 bytes
├──────┴──────┴─┴─┴─┴─┴─┴─┴─┼─────────────────────────┤
│        Checksum            │    Urgent Pointer      │ 4 bytes
├────────────────────────────┴─────────────────────────┤

Flags:
  SYN — инициация соединения
  ACK — подтверждение
  FIN — завершение
  RST — сброс (ошибка)
  PSH — push (отправить сразу)
  URG — urgent data

Min header: 20 bytes, Max: 60 bytes (with options)
```

### Keepalive

```
TCP Keepalive (L4):
  net.ipv4.tcp_keepalive_time = 7200  (2 часа до первого probe)
  net.ipv4.tcp_keepalive_intvl = 75   (интервал между probes)
  net.ipv4.tcp_keepalive_probes = 9   (сколько проб до drop)

HTTP Keep-Alive (L7):
  Connection: keep-alive
  Keep-Alive: timeout=5, max=100
  Переиспользование TCP соединения для нескольких HTTP запросов

В Go:
  http.Transport{
      IdleConnTimeout: 90 * time.Second,
      MaxIdleConns:    100,
  }
```

## UDP (User Datagram Protocol)

```
Свойства:
  - Connectionless (без handshake)
  - Unreliable (нет гарантии доставки)
  - No ordering (пакеты могут приходить в другом порядке)
  - No flow/congestion control
  - Маленький header (8 bytes vs 20+ у TCP)
  - Быстрее TCP (нет overhead)

Header:
  Source Port (2) | Dest Port (2) | Length (2) | Checksum (2) = 8 bytes

Используется:
  - DNS (запрос/ответ < 512 bytes)
  - Video/Audio streaming (потеря кадра OK)
  - Gaming (low latency важнее reliability)
  - QUIC (HTTP/3) — reliability поверх UDP
  - DHCP, NTP, SNMP
```

## TCP vs UDP

```
│ Свойство          │ TCP                │ UDP              │
├───────────────────┼────────────────────┼──────────────────┤
│ Connection        │ Connection-oriented│ Connectionless   │
│ Reliability       │ Гарантирует        │ Нет              │
│ Ordering          │ Гарантирует        │ Нет              │
│ Flow control      │ Да (window)        │ Нет              │
│ Congestion ctrl   │ Да (CUBIC/BBR)     │ Нет              │
│ Header size       │ 20-60 bytes        │ 8 bytes          │
│ Speed             │ Медленнее          │ Быстрее          │
│ Broadcast         │ Нет                │ Да               │
│ Use case          │ HTTP, DB, email    │ DNS, video, game │
│ HOL blocking      │ Да                 │ Нет              │
```

## Порты

```
Well-known ports (0-1023):
  20, 21  — FTP
  22      — SSH
  25      — SMTP
  53      — DNS
  80      — HTTP
  443     — HTTPS
  5432    — PostgreSQL
  6379    — Redis
  9092    — Kafka

Ephemeral ports (49152-65535):
  Назначаются ОС для client-side соединений
  Лимит: ~16K портов на IP
  Если исчерпаны → "cannot assign requested address"
```

## Частые вопросы

**Q: Зачем TIME_WAIT?**
A: 1) Гарантировать что последний ACK дошёл (если потерялся — server повторит FIN, а мы ещё слушаем). 2) Предотвратить путаницу с поздними пакетами от старого соединения.

**Q: Что такое Nagle's algorithm?**
A: Буферизирует мелкие TCP пакеты и отправляет один большой. Уменьшает overhead, но добавляет latency. Отключается через TCP_NODELAY (нужно для real-time).

**Q: Что такое HOL blocking?**
A: Head-of-line blocking: потеря одного пакета в TCP блокирует ВСЕ последующие (даже из других потоков). HTTP/2 решает на L7, но не на L4. HTTP/3 (QUIC/UDP) решает полностью.

**Q: SYN flood атака?**
A: Атакующий шлёт тысячи SYN без завершения handshake → server заполняет backlog полуоткрытыми соединениями. Защита: SYN cookies (net.ipv4.tcp_syncookies=1).
