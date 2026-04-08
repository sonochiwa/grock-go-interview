# Back-of-Envelope расчёты

## Порядки величин (ВЫУЧИ НАИЗУСТЬ)

### Степени двойки

| Степень | Точное | Приблизительное | Название |
|---------|--------|-----------------|----------|
| 2^10 | 1024 | ~1 тысяча | 1 KB |
| 2^20 | 1,048,576 | ~1 миллион | 1 MB |
| 2^30 | 1,073,741,824 | ~1 миллиард | 1 GB |
| 2^40 | | ~1 триллион | 1 TB |

### Время

| Операция | Latency | Примечание |
|----------|---------|------------|
| L1 cache reference | 0.5 ns | |
| L2 cache reference | 7 ns | |
| Main memory (RAM) | 100 ns | |
| SSD random read | 16 μs | ~100x RAM |
| SSD sequential read 1MB | 1 ms | |
| HDD random read | 2-10 ms | ~1000x RAM |
| HDD sequential read 1MB | 20 ms | |
| Network round trip (same DC) | 0.5 ms | |
| Network round trip (cross-DC) | 30-100 ms | |
| Network round trip (cross-continent) | 100-200 ms | |
| Read 1MB from memory | 0.25 ms | |
| Read 1MB from SSD | 1 ms | |
| Read 1MB from network (1 Gbps) | 10 ms | |
| Disk seek | 10 ms | |

### IOPS (Input/Output Operations Per Second)

| Устройство | Random IOPS | Sequential MB/s |
|------------|-------------|-----------------|
| HDD (7200 RPM) | 100-200 | 100-200 MB/s |
| SATA SSD | 10,000-100,000 | 500 MB/s |
| NVMe SSD | 100,000-1,000,000 | 3-7 GB/s |
| RAM | ∞ (ограничено CPU) | 50+ GB/s |

### Пропускная способность

| Канал | Bandwidth |
|-------|-----------|
| 1 Gbps network | ~125 MB/s |
| 10 Gbps network | ~1.25 GB/s |
| SSD NVMe | ~3-7 GB/s |
| DDR4 RAM | ~50 GB/s |

### Размеры данных

| Что | Примерный размер |
|-----|-----------------|
| UUID | 16 байт (binary) / 36 байт (string) |
| Timestamp (Unix) | 8 байт |
| IPv4 | 4 байта |
| IPv6 | 16 байт |
| Email | ~50 байт |
| URL | ~100 байт |
| Tweet/короткое сообщение | ~500 байт |
| Аватар (compressed) | ~10 KB |
| Фото (compressed) | ~200 KB - 2 MB |
| Минута видео (compressed) | ~5-50 MB |

## Формулы

### RPS (Requests Per Second)

```
RPS_avg = DAU × requests_per_user / 86400
RPS_peak = RPS_avg × peak_factor (обычно 2-5x)

Пример:
  10M DAU × 20 req/user / 86400 = ~2300 RPS avg
  Peak: 2300 × 3 = ~7000 RPS
```

### Storage

```
Storage = users × data_per_user × retention_period

Пример (chat):
  100M users × 40 messages/day × 500 bytes × 365 days
  = 100M × 40 × 500 × 365
  = 730 TB / year
  ≈ 2 TB / day
```

### Bandwidth

```
Bandwidth = RPS × avg_response_size

Пример:
  7000 RPS × 10 KB = 70 MB/s
  В год: 70 × 86400 × 365 ≈ 2 PB
```

### Количество серверов

```
Servers = RPS_peak / RPS_per_server

Типичный Go сервер:
  CPU-bound: 1000-5000 RPS на ядро (зависит от логики)
  IO-bound: 10,000-50,000 RPS (Go отлично справляется с IO)

Пример:
  7000 RPS / 5000 per server = 2 сервера + запас = 4-6 серверов
```

### QPS для базы данных

```
Один PostgreSQL сервер:
  Simple queries: 10,000-50,000 QPS
  Complex queries (JOIN): 100-1000 QPS
  Write-heavy: 5000-20,000 TPS (transactions/sec)

Один Redis:
  100,000-200,000 ops/sec (single thread)
  1,000,000+ ops/sec (cluster)

Один Kafka broker:
  500,000-1,000,000 messages/sec (depends on message size)
```

### Быстрые приблизительные вычисления

```
Секунд в дне: 86400 ≈ 10^5 (для quick math)
Секунд в году: ~3 × 10^7

Если 1M RPS → за день: 86 billion requests
Если 1KB на запрос, 1000 RPS → 1 MB/s → 86 GB/day → 30 TB/year
```

## Шаблон расчёта на собесе

```
"Давайте прикинем нагрузку:

Пользователи:
- DAU: 10M
- Каждый делает ~20 запросов в день

Нагрузка:
- AVG RPS: 10M × 20 / 100K = 2000 RPS
- Peak RPS: 2000 × 3 = 6000 RPS

Storage (за год):
- 10M users × 10KB profile = 100 GB (metadata)
- 10M × 5 posts/day × 1KB × 365 = 18 TB (content)

Bandwidth:
- 6000 RPS × 5KB avg response = 30 MB/s

Серверы:
- Go сервис: 6000 / 5000 per server ≈ 2 + reserve = 4 сервера
- БД: 6000 QPS — один Postgres master + replicas

Выводы:
- Read-heavy → кэш перед БД
- 18TB/год → нужна стратегия архивации
- 6K RPS — один Go сервис справится, но нужен HA"
```

## Частые ошибки

1. **Не считать вообще** — "ну, много данных" (плохо)
2. **Считать слишком точно** — 86400 не 100000, но на собесе это ок
3. **Забыть про peak** — average ≠ peak, дизайн на average = падение на пике
4. **Не учитывать рост** — "через год данных будет в 3 раза больше"
5. **Путать MB и Mb** — 1 MB = 8 Mb (bit vs byte)
