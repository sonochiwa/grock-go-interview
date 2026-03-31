# Linux Networking

## iptables / nftables

```
iptables — packet filter (firewall)

Chains:
  INPUT    — входящие пакеты (к этому хосту)
  OUTPUT   — исходящие пакеты (от этого хоста)
  FORWARD  — транзитные пакеты (routing)
  PREROUTING  — до routing decision (DNAT)
  POSTROUTING — после routing decision (SNAT/masquerade)

Tables:
  filter — разрешить/запретить (default)
  nat    — трансляция адресов
  mangle — модификация пакетов
  raw    — до connection tracking

Примеры:
  # Разрешить SSH
  iptables -A INPUT -p tcp --dport 22 -j ACCEPT

  # Запретить всё входящее кроме established
  iptables -P INPUT DROP
  iptables -A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT

  # Port forwarding (DNAT)
  iptables -t nat -A PREROUTING -p tcp --dport 80 -j DNAT --to 10.0.1.5:8080

  # SNAT (masquerade) для выхода в интернет
  iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE

  # Просмотр
  iptables -L -n -v          # filter table
  iptables -t nat -L -n -v   # NAT table

nftables — замена iptables (современнее):
  nft add rule inet filter input tcp dport 22 accept
  nft list ruleset
```

## Network Namespaces

```
Изоляция network stack: свои interfaces, routing table, iptables

# Создать namespace
ip netns add myns

# Запустить команду в namespace
ip netns exec myns ip addr    # свой network stack
ip netns exec myns ping 8.8.8.8  # не работает (нет интерфейсов)

# Создать veth pair (виртуальный кабель)
ip link add veth0 type veth peer name veth1
ip link set veth1 netns myns

# Назначить IP
ip addr add 10.0.0.1/24 dev veth0
ip netns exec myns ip addr add 10.0.0.2/24 dev veth1

# Поднять интерфейсы
ip link set veth0 up
ip netns exec myns ip link set veth1 up

# Теперь ping работает
ping 10.0.0.2  ✅

Docker делает это автоматически для каждого контейнера:
  Container → veth pair → bridge (docker0) → iptables NAT → host network
```

## Network Stack (пакет изнутри)

```
Путь пакета в Linux:

Incoming:
  NIC → DMA → Ring Buffer → NAPI poll → Driver →
  → netfilter/iptables (PREROUTING) →
  → Routing decision →
  → netfilter (INPUT) → Socket buffer → Application

Outgoing:
  Application → Socket buffer →
  → Routing decision →
  → netfilter (OUTPUT) →
  → netfilter (POSTROUTING) →
  → Driver → NIC → Wire

Tuning:
  # Ring buffer size (сетевая карта)
  ethtool -g eth0
  ethtool -G eth0 rx 4096

  # Socket buffer size
  net.core.rmem_max = 16777216
  net.core.wmem_max = 16777216

  # TCP buffer
  net.ipv4.tcp_rmem = 4096 87380 16777216
  net.ipv4.tcp_wmem = 4096 65536 16777216

  # Connection backlog
  net.core.somaxconn = 65535
  net.ipv4.tcp_max_syn_backlog = 65535

  # Port range
  net.ipv4.ip_local_port_range = 1024 65535
```

## Bridge

```
Bridge = virtual switch (L2)

# Docker default: docker0 bridge
brctl show
# bridge name     bridge id           STP enabled     interfaces
# docker0         8000.0242ac110002   no              veth1234
#                                                     veth5678

# Каждый container подключен через veth pair к bridge
# Bridge → iptables NAT → host interface → internet
```

## Частые вопросы

**Q: Как Docker обеспечивает сетевую изоляцию?**
A: Network namespace (отдельный stack) + veth pair (виртуальный кабель к bridge) + iptables (NAT для выхода в интернет, port forwarding для -p 8080:80).

**Q: Как контейнеры общаются друг с другом?**
A: Через bridge (docker0). Оба подключены к bridge → L2 communication. По имени — через встроенный DNS (user-defined networks).

**Q: Что такое conntrack?**
A: Connection tracking — ядро отслеживает все TCP/UDP соединения. Позволяет: stateful firewall (ESTABLISHED,RELATED), NAT. conntrack -L — список. Проблема: таблица переполняется при высокой нагрузке → nf_conntrack_max.
