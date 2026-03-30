# Масштабирование

## Вертикальное vs Горизонтальное

```
Вертикальное (Scale Up):     Горизонтальное (Scale Out):
  ┌──────────┐                ┌────┐ ┌────┐ ┌────┐
  │          │                │ S1 │ │ S2 │ │ S3 │
  │ BIGGER   │                └────┘ └────┘ └────┘
  │ SERVER   │                    ↑
  │          │                ┌────────┐
  └──────────┘                │   LB   │
                              └────────┘
```

| | Вертикальное | Горизонтальное |
|---|---|---|
| Что | Мощнее железо | Больше серверов |
| Предел | Физический (max CPU/RAM) | Практически нет |
| Сложность | Простое | Distributed systems проблемы |
| Downtime | При апгрейде | Zero downtime |
| Стоимость | Экспоненциально растёт | Линейно растёт |
| Когда | Начальный этап, БД | Stateless сервисы, web |

## Load Balancing

### Алгоритмы

```
Round Robin:        A → B → C → A → B → C
Weighted RR:        A(3) → A → A → B(1) → C(2) → C → ...
Least Connections:  Кто менее загружен → ему
IP Hash:            hash(client_IP) % N → sticky sessions
Random:             Случайный сервер
```

### L4 vs L7 Load Balancer

```
L4 (Transport):
  - Работает на TCP/UDP уровне
  - Не видит HTTP headers
  - Быстрый, дешёвый
  - Пример: AWS NLB, HAProxy (TCP mode)

L7 (Application):
  - Видит HTTP: URL, headers, cookies
  - Маршрутизация по path, host
  - SSL termination
  - Пример: Nginx, AWS ALB, Envoy
```

### Стратегии деплоя

```
Blue-Green:
  [Blue (v1)] ← LB → [Green (v2)]
  Переключаем LB на Green

Canary:
  [v1] [v1] [v1] [v2]    ← 25% трафика на v2
  Если ок → [v2] [v2] [v2] [v2]

Rolling:
  [v1→v2] [v1] [v1] [v1]
  [v2] [v1→v2] [v1] [v1]
  [v2] [v2] [v1→v2] [v1]
  [v2] [v2] [v2] [v1→v2]
```

## Auto-scaling

```yaml
# Типичные метрики для auto-scaling:
- CPU utilization > 70%
- Memory > 80%
- RPS per instance > threshold
- Queue depth > threshold
- Custom metrics (p99 latency)
```

## Stateless vs Stateful

```
Stateless (легко масштабировать):
  - Сессии в Redis/JWT
  - Нет локального состояния
  - Любой сервер может обработать любой запрос

Stateful (сложно масштабировать):
  - Сессии в памяти
  - WebSocket connections
  - In-memory cache
  → Нужен sticky sessions или shared state
```

## DNS Load Balancing

```
myapp.com → [DNS]
             ├→ 1.2.3.4 (US-East)
             ├→ 5.6.7.8 (EU-West)
             └→ 9.10.11.12 (Asia)

Geo-based routing: пользователь из EU → EU-West DC
```

## Частые вопросы

**Q: С чего начать масштабирование?**
A: 1) Мониторинг → найти bottleneck. 2) Кэширование. 3) Read replicas. 4) Горизонтальное масштабирование stateless сервисов. 5) Шардинг.

**Q: Когда нужен Load Balancer?**
A: Когда больше одного сервера. Даже с одним — для health checks и zero-downtime deploys.
