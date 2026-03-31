# OSI Model

## 7 уровней

```
# │ Уровень          │ PDU        │ Протоколы              │ Устройства
──┼──────────────────┼────────────┼────────────────────────┼──────────────
7 │ Application      │ Data       │ HTTP, gRPC, DNS, SMTP  │ Proxy, WAF
6 │ Presentation     │ Data       │ TLS/SSL, MIME, JSON    │ —
5 │ Session          │ Data       │ WebSocket, RPC         │ —
4 │ Transport        │ Segment    │ TCP, UDP, QUIC         │ Firewall (L4)
3 │ Network          │ Packet     │ IP, ICMP, BGP, OSPF    │ Router
2 │ Data Link        │ Frame      │ Ethernet, ARP, VLAN    │ Switch
1 │ Physical         │ Bits       │ Ethernet PHY, Wi-Fi    │ Hub, Cable
```

> На практике используется модель TCP/IP (4 уровня), но OSI спрашивают на собесах.

## Инкапсуляция

```
Application data:  [HTTP Request]
                        ↓
Transport:         [TCP Header | HTTP Request]
                        ↓
Network:           [IP Header | TCP Header | HTTP Request]
                        ↓
Data Link:         [Eth Header | IP Header | TCP Header | HTTP Request | Eth Trailer]
                        ↓
Physical:          101010110001110101001...
```

Каждый уровень добавляет свой header (и иногда trailer). При получении — обратный процесс (деинкапсуляция).

## TCP/IP vs OSI

```
TCP/IP Model (4)     │ OSI Model (7)
─────────────────────┼──────────────────
Application          │ Application (7)
                     │ Presentation (6)
                     │ Session (5)
─────────────────────┼──────────────────
Transport            │ Transport (4)
─────────────────────┼──────────────────
Internet             │ Network (3)
─────────────────────┼──────────────────
Network Access       │ Data Link (2)
                     │ Physical (1)
```

## На каком уровне что происходит

```
L7 (Application):
  - HTTP запрос/ответ (GET /api/users)
  - DNS resolve (domain → IP)
  - gRPC (protobuf over HTTP/2)
  - WebSocket (after HTTP upgrade)

L6 (Presentation):
  - TLS шифрование/дешифрование
  - Сериализация (JSON, protobuf, gzip)
  - Конвертация кодировок (UTF-8)

L5 (Session):
  - Установка/поддержание сессий
  - WebSocket session management
  - TLS session resumption

L4 (Transport):
  - TCP: reliable, ordered, connection-oriented
  - UDP: unreliable, unordered, connectionless
  - Порты (src:49152 → dst:443)
  - Flow control (TCP window)
  - Congestion control

L3 (Network):
  - IP адресация (src IP → dst IP)
  - Routing между сетями
  - Фрагментация пакетов
  - TTL (hop limit)
  - ICMP (ping, traceroute)

L2 (Data Link):
  - MAC адресация (ARP: IP → MAC)
  - Ethernet frames
  - VLAN tagging
  - Error detection (CRC)

L1 (Physical):
  - Электрические сигналы
  - Оптоволокно, витая пара, Wi-Fi
  - Битрейт, модуляция
```

## Что спрашивают на собесах

**Q: На каком уровне работает HTTP?**
A: L7 (Application). Но использует TCP (L4), IP (L3), Ethernet (L2).

**Q: На каком уровне работает TLS?**
A: Между L4 и L7 — формально L6 (Presentation) в OSI, но в TCP/IP модели — часть Application layer.

**Q: Чем L4 load balancer отличается от L7?**
A: L4 маршрутизирует по IP:port (быстро, не видит HTTP). L7 маршрутизирует по HTTP path/headers/cookies (медленнее, но гибче).

**Q: Что такое MTU?**
A: Maximum Transmission Unit — максимальный размер пакета на L2. Ethernet MTU = 1500 bytes. Если IP пакет > MTU → фрагментация (или PMTUD с DF flag).

**Q: ARP — это какой уровень?**
A: L2 (Data Link). Преобразует IP → MAC адрес в локальной сети.
