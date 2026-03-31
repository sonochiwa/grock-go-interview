# Load Balancing

## L4 vs L7

```
L4 (Transport):
  Работает с TCP/UDP: IP + Port
  Не видит HTTP content (path, headers, cookies)
  Быстрее (меньше обработки)
  NAT: переписывает dst IP
  Примеры: AWS NLB, LVS, IPVS, HAProxy (TCP mode)

  Client → [L4 LB] → Backend1:8080
                    → Backend2:8080

L7 (Application):
  Работает с HTTP: path, headers, cookies, body
  Может маршрутизировать по content
  Медленнее (парсинг HTTP)
  TLS termination
  Примеры: Nginx, HAProxy (HTTP mode), AWS ALB, Envoy, Traefik

  Client → [L7 LB]
            /api/* → API servers
            /static/* → CDN
            Host: admin.* → Admin servers
            Cookie: canary=true → Canary servers
```

## Алгоритмы

```
Round Robin:
  → A → B → C → A → B → C
  Простой, не учитывает нагрузку

Weighted Round Robin:
  A (weight=3), B (weight=1)
  → A → A → A → B → A → A → A → B
  Для серверов разной мощности

Least Connections:
  Запрос → серверу с наименьшим числом active connections
  Лучше для long-lived connections (WebSocket, gRPC streams)

Weighted Least Connections:
  Least connections + вес

IP Hash:
  hash(client_ip) % num_servers
  Один клиент → один сервер (sticky)
  Проблема: перебалансировка при добавлении/удалении серверов

Consistent Hashing:
  Хеш-кольцо с virtual nodes
  Добавление/удаление сервера → перемещается ~1/N ключей
  Используется в: Nginx upstream, Envoy, кэши

Random:
  Удивительно эффективен при большом кол-ве серверов
  "Power of Two Choices": выбрать 2 случайных → отправить менее загруженному

Least Response Time:
  Запрос → серверу с минимальным response time
  Адаптивен к реальной нагрузке
```

## Health Checks

```
Passive (наблюдение за трафиком):
  Если сервер вернул 5xx N раз → пометить unhealthy
  Если timeout N раз → пометить unhealthy
  Не требует доп. трафика

Active (периодические проверки):
  GET /healthz каждые 10s
  200 OK → healthy
  Любой другой ответ или timeout → unhealthy
  Требует endpoint на сервере

Nginx:
  upstream backend {
      server 10.0.0.1:8080 max_fails=3 fail_timeout=30s;
      server 10.0.0.2:8080 max_fails=3 fail_timeout=30s;
  }

Envoy / ALB:
  health_check:
    path: /healthz
    interval: 10s
    timeout: 5s
    healthy_threshold: 2
    unhealthy_threshold: 3
```

## Sticky Sessions

```
Проблема: stateful sessions (login state в памяти сервера)

Решения:
  1. Cookie-based (L7):
     LB добавляет cookie: SERVERID=server-1
     Следующие запросы → тот же сервер
     Проблема: если сервер умер → потеря сессии

  2. IP-based (L4):
     hash(client_ip) → всегда тот же сервер
     Проблема: за NAT один IP = много пользователей

  3. Лучше: stateless sessions
     Session в Redis/DB → любой сервер может обработать
     JWT → state в самом токене
```

## TLS Termination

```
Варианты:
  1. TLS на LB (termination):
     Client →[HTTPS]→ LB →[HTTP]→ Backend
     ✅ Централизованное управление сертификатами
     ✅ Backend проще (не нужен TLS)
     ❌ HTTP между LB и backend (если не доверяем сети)

  2. TLS passthrough:
     Client →[HTTPS]→ LB →[HTTPS]→ Backend
     LB не видит HTTP content (только L4)
     ✅ End-to-end encryption
     ❌ Нет L7 routing

  3. TLS re-encryption:
     Client →[HTTPS]→ LB →[HTTPS]→ Backend
     LB расшифровывает, маршрутизирует, шифрует заново
     ✅ L7 routing + encryption
     ❌ Двойное шифрование = CPU overhead
```

## Частые вопросы

**Q: Когда L4, когда L7?**
A: L4 — high throughput, простой round-robin, TCP/UDP. L7 — content-based routing, TLS termination, WebSocket, canary deploys.

**Q: Как LB обрабатывает gRPC?**
A: gRPC = HTTP/2 с long-lived connections. L4 LB видит одно соединение → все запросы на один backend. Нужен L7 LB (Envoy) с per-RPC балансировкой. Или client-side LB.

**Q: Что если LB — single point of failure?**
A: Active-passive LB pair с shared virtual IP (VRRP/keepalived). Или DNS-based (multiple A records). Облако: managed LB (ALB/NLB) с built-in HA.
