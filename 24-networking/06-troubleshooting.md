# Network Troubleshooting

## Инструменты

### curl

```bash
# Базовый HTTP
curl -v https://api.example.com/health   # verbose (headers, TLS info)
curl -I https://api.example.com/         # только headers (HEAD)
curl -X POST -H "Content-Type: application/json" -d '{"key":"val"}' url

# Timing
curl -w "@curl-format.txt" -o /dev/null -s https://example.com
# curl-format.txt:
#   dns_resolution:  %{time_namelookup}s
#   tcp_established: %{time_connect}s
#   tls_handshake:   %{time_appconnect}s
#   first_byte:      %{time_starttransfer}s
#   total:           %{time_total}s

# Типичный вывод:
#   dns_resolution:  0.012s
#   tcp_established: 0.045s    (RTT ≈ 33ms)
#   tls_handshake:   0.112s
#   first_byte:      0.156s    (server processing ≈ 44ms)
#   total:           0.189s

# Resolve override (тестирование без DNS)
curl --resolve api.example.com:443:10.0.1.5 https://api.example.com/

# Follow redirects
curl -L url

# С сертификатом
curl --cacert ca.pem --cert client.pem --key client-key.pem https://...
```

### dig / nslookup

```bash
# DNS lookup
dig example.com               # A запись
dig example.com AAAA           # IPv6
dig example.com MX             # Mail servers
dig example.com ANY            # Все записи
dig +trace example.com         # Полная цепочка resolution
dig +short example.com         # Только IP
dig @8.8.8.8 example.com      # Через конкретный DNS

# Reverse DNS
dig -x 93.184.216.34

# Время ответа
dig example.com | grep "Query time"
# ;; Query time: 12 msec
```

### ss / netstat

```bash
# ss (замена netstat, быстрее)
ss -tlnp              # TCP listening ports + PID
ss -tunap             # Все TCP/UDP соединения с процессами
ss -s                 # Статистика (сколько соединений по состояниям)
ss state time-wait    # Все соединения в TIME_WAIT
ss dst 10.0.1.5       # Соединения к конкретному IP

# Подсчёт соединений по состоянию
ss -tan | awk '{print $1}' | sort | uniq -c | sort -rn
#  1523 ESTAB
#   342 TIME-WAIT
#    12 CLOSE-WAIT    ← проблема! Не закрываются соединения
#     5 LISTEN
```

### tcpdump

```bash
# Захват трафика
tcpdump -i eth0 port 443                    # Весь HTTPS трафик
tcpdump -i eth0 host 10.0.1.5               # Трафик к/от хоста
tcpdump -i eth0 'tcp[tcpflags] & (tcp-syn) != 0'  # Только SYN пакеты
tcpdump -i eth0 -w capture.pcap             # Сохранить в файл
tcpdump -r capture.pcap                      # Прочитать из файла

# Фильтры
tcpdump -i any port 5432                     # PostgreSQL
tcpdump -i any 'port 80 and host 10.0.1.5'  # HTTP к конкретному хосту
tcpdump -i any -A port 80                    # ASCII output (видно HTTP)
tcpdump -i any -n                            # Без DNS resolution (быстрее)

# Примеры debug:
# "Почему сервис не отвечает?"
tcpdump -i any port 8080 -c 10  # Видим ли TCP SYN?
# Если SYN есть, но нет SYN-ACK → сервер не слушает / firewall

# "Почему соединение медленное?"
tcpdump -i any port 443 -ttt    # Показать delta time между пакетами
```

### traceroute / mtr

```bash
# Путь пакета через сеть
traceroute example.com
# 1  gateway (192.168.1.1)    1.2ms
# 2  isp-router (10.0.0.1)   5.3ms
# 3  * * *                    (filtered/timeout)
# 4  cdn-edge (93.184.216.34) 12.1ms

# mtr = traceroute + ping (realtime)
mtr example.com
# Показывает: loss%, avg latency, jitter для каждого hop
```

## Типичные проблемы и диагностика

```
Проблема: "Connection refused"
  Причина: порт не слушается
  Диагностика:
    ss -tlnp | grep 8080        # Слушает ли процесс?
    curl -v localhost:8080       # Локально работает?
    iptables -L -n              # Firewall блокирует?

Проблема: "Connection timeout"
  Причина: пакеты не доходят
  Диагностика:
    ping target_ip              # ICMP доходит?
    traceroute target_ip        # Где теряется?
    tcpdump -i any host target  # SYN уходит? SYN-ACK приходит?
    iptables -L -n              # Firewall?
    security groups (cloud)?

Проблема: "Connection reset"
  Причина: peer закрыл соединение
  Диагностика:
    tcpdump → видим RST пакет? От кого?
    Сервер OOM? (dmesg | grep -i oom)
    max connections exceeded?

Проблема: Медленные запросы
  Диагностика:
    curl -w timing              # Где задержка? DNS? TCP? TLS? Server?
    mtr target                  # Высокий latency на hop?
    ss -i dst target            # TCP window, retransmits?
    tcpdump -ttt                # Delta time между пакетами?

Проблема: "Too many open files"
  Причина: исчерпаны file descriptors
  Диагностика:
    ulimit -n                   # Текущий лимит fd
    ls /proc/<pid>/fd | wc -l   # Сколько fd у процесса
    ss -s                       # Сколько сокетов
    lsof -p <pid> | wc -l       # Все открытые файлы
  Решение:
    ulimit -n 65535 (или в systemd LimitNOFILE=65535)

Проблема: Много TIME_WAIT
  Причина: короткоживущие TCP соединения
  Диагностика:
    ss state time-wait | wc -l
  Решение:
    net.ipv4.tcp_tw_reuse = 1
    Connection pooling (Keep-Alive)
```

## MTU / MSS

```
MTU (Maximum Transmission Unit):
  Максимальный размер L2 frame payload
  Ethernet: 1500 bytes
  Jumbo frames: 9000 bytes (datacenter)

MSS (Maximum Segment Size):
  Максимальный размер TCP data в одном segment
  MSS = MTU - IP header (20) - TCP header (20)
  Ethernet: MSS = 1500 - 40 = 1460 bytes

Path MTU Discovery (PMTUD):
  Отправляет пакеты с DF (Don't Fragment) flag
  Если пакет > MTU роутера → ICMP "Fragmentation Needed"
  Sender уменьшает размер пакета

  Проблема: если ICMP заблокирован → "black hole"
  Пакеты молча теряются, соединение зависает
  → Всегда разрешать ICMP type 3 code 4!
```
