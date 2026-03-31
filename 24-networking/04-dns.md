# DNS (Domain Name System)

## Как работает DNS resolution

```
User types: example.com

1. Browser cache → нет
2. OS cache (/etc/hosts, systemd-resolved) → нет
3. Recursive resolver (провайдер / 8.8.8.8 / 1.1.1.1):
   │
   ├─→ Root nameserver (.)
   │   "Кто знает .com?" → "Спроси ns1.verisign.com"
   │
   ├─→ TLD nameserver (.com)
   │   "Кто знает example.com?" → "Спроси ns1.example.com"
   │
   └─→ Authoritative nameserver (example.com)
       "IP для example.com?" → "93.184.216.34"

4. Resolver кэширует ответ (TTL)
5. Возвращает IP клиенту

Весь процесс: ~10-100ms (без кэша)
С кэшем: <1ms
```

## Типы записей

```
│ Тип   │ Назначение               │ Пример                              │
├───────┼──────────────────────────┼─────────────────────────────────────┤
│ A     │ IPv4 адрес               │ example.com → 93.184.216.34         │
│ AAAA  │ IPv6 адрес               │ example.com → 2606:2800:...         │
│ CNAME │ Алиас (→ другой домен)   │ www.example.com → example.com       │
│ MX    │ Mail сервер              │ example.com → mail.example.com (10) │
│ TXT   │ Текст (SPF, DKIM, verify)│ example.com → "v=spf1 ..."         │
│ NS    │ Nameserver               │ example.com → ns1.example.com       │
│ SOA   │ Start of Authority       │ Инфо о зоне (serial, refresh, TTL)  │
│ SRV   │ Service (host:port)      │ _grpc._tcp.example.com → ...        │
│ PTR   │ Reverse DNS (IP → domain)│ 34.216.184.93 → example.com         │

CNAME важно:
  - НЕ может быть на apex (корневой) домен
  - Только один CNAME на запись (не может сосуществовать с A)
  - Используется для CDN: cdn.example.com → d123.cloudfront.net
```

## TTL (Time to Live)

```
TTL = сколько секунд кэшировать запись

Типичные значения:
  300 (5 мин)   — динамичный (failover, blue-green deploy)
  3600 (1 час)  — стандартный
  86400 (1 день) — стабильный

Проблема при миграции:
  TTL = 86400 → после смены IP клиенты идут на старый IP сутки
  Решение: заранее понизить TTL до 300 → сменить IP → вернуть TTL

Negative TTL:
  Если домен не найден → кэшировать NXDOMAIN на SOA minimum TTL
  Может мешать при добавлении нового домена
```

## Рекурсивный vs Итеративный

```
Рекурсивный (resolver делает всю работу):
  Client → Resolver: "Где example.com?"
  Resolver → Root → TLD → Auth → Resolver → Client
  Client получает готовый ответ

Итеративный (каждый сервер даёт ссылку):
  Client → Root: "Где example.com?" → "Спроси .com NS"
  Client → TLD: "Где example.com?" → "Спроси auth NS"
  Client → Auth: "Где example.com?" → "93.184.216.34"

На практике:
  Client → Resolver: рекурсивный
  Resolver → Root → TLD → Auth: итеративный
```

## DNS в production

```
DNS Load Balancing:
  Множество A-записей на один домен:
  api.example.com → 10.0.1.1, 10.0.1.2, 10.0.1.3
  DNS возвращает в random/round-robin порядке

  Проблемы:
  - Нет health check (мёртвый IP остаётся в DNS)
  - Кэширование → неравномерная нагрузка
  - Не учитывает нагрузку серверов

GeoDNS:
  Разные IP для разных регионов:
  api.example.com → 10.0.1.1 (US)
  api.example.com → 10.0.2.1 (EU)
  Используется в CDN (CloudFlare, AWS Route53)

DNS Failover:
  Health check + автоматическое удаление unhealthy IP
  Route53 / CloudFlare managed DNS

Service Discovery (internal):
  Consul DNS: myservice.service.consul → 10.0.3.5
  Kubernetes: myservice.default.svc.cluster.local → 10.96.0.15
  CoreDNS в k8s
```

## Частые вопросы

**Q: Почему DNS использует UDP?**
A: Запрос/ответ обычно < 512 bytes — один пакет. TCP overhead (handshake) не нужен. Для больших ответов (DNSSEC, множество записей) — TCP fallback.

**Q: Что такое DNS amplification attack?**
A: Attacker шлёт DNS запрос с поддельным source IP (жертвы). DNS сервер отвечает жертве большим ответом. Amplification: запрос 50 bytes → ответ 3000 bytes = 60×.

**Q: Как работает DNS в Docker/K8s?**
A: Container DNS → CoreDNS (кластерный) → upstream DNS. Резолвит service names: `my-service.my-namespace.svc.cluster.local`. Настраивается через `/etc/resolv.conf` (ndots:5 в k8s).
