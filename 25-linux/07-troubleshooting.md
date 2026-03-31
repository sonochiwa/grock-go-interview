# Linux Troubleshooting

## Шпаргалка: что смотреть

```
Проблема          │ Инструменты
──────────────────┼────────────────────────────
CPU высокий       │ top, htop, pidstat, perf top
Memory высокий    │ free -h, vmstat, /proc/meminfo, smem
Disk I/O          │ iostat, iotop, blktrace
Network           │ ss, tcpdump, iftop, nethogs
Process issues    │ ps, strace, lsof, /proc/<pid>/
Kernel/Boot       │ dmesg, journalctl -k
Logs              │ journalctl -u <service>, /var/log/
```

## top / htop

```
top:
  load average: 1.50, 2.00, 1.80   ← 1min, 5min, 15min
  # Load = runnable + uninterruptible processes
  # Load < num_cores → OK
  # Load > num_cores → перегрузка

  %Cpu(s): 25.0 us,  5.0 sy,  0.0 ni, 68.0 id,  2.0 wa
  # us = user space
  # sy = kernel space
  # ni = nice (low priority)
  # id = idle
  # wa = I/O wait (!!!)  ← высокий = disk bottleneck
  # hi/si = hardware/software interrupts

  MiB Mem:  16384.0 total,  2048.0 free,  8192.0 used,  6144.0 buff/cache
  # buff/cache = можно освободить при необходимости
  # "available" = free + reclaimable cache

Полезные hotkeys в top:
  1     — показать per-CPU
  M     — сортировать по памяти
  P     — сортировать по CPU
  H     — показать потоки
  c     — полная команда

htop:
  Визуальнее, поддержка mouse, tree view, filter
  F5 — tree view (parent/child)
  F6 — сортировка
  F4 — filter по имени
```

## strace

```bash
# Системные вызовы процесса (самый мощный debug tool!)

strace -p <pid>                    # attach к процессу
strace -p <pid> -e trace=network   # только network syscalls
strace -p <pid> -e trace=file      # только file syscalls
strace -p <pid> -e trace=read,write # только read/write
strace -p <pid> -c                 # статистика syscalls
strace -p <pid> -T                 # время каждого syscall
strace -f ./myapp                  # follow child processes

# Примеры debug:

# "Почему процесс зависает?"
strace -p <pid>
# → futex(... FUTEX_WAIT ...) → deadlock!
# → read(5, ... → ждёт I/O на fd 5 → lsof -p <pid> → fd 5 = socket

# "Почему файл не открывается?"
strace -e trace=open,openat ./myapp
# openat(AT_FDCWD, "/etc/config.yaml", O_RDONLY) = -1 ENOENT
# → Файл не найден!

# "Где bottleneck?"
strace -c -p <pid>  # через 10 секунд Ctrl+C
# % time     seconds  usecs/call     calls    errors syscall
# 85.71      1.234567     123         10000           epoll_wait
# 10.00      0.144000      14         10000           write
#  4.29      0.061714       6         10000           read
```

## vmstat / iostat

```bash
# vmstat — memory, swap, I/O, CPU
vmstat 1 5    # каждую секунду, 5 раз
# procs -------memory------ ---swap-- -----io---- -system-- ------cpu-----
#  r  b   swpd   free   buff  cache   si   so    bi    bo   in   cs us sy id wa
#  1  0      0 2048000  128   6144     0    0     5   100  500  800 25  5 68  2

# r  = runnable processes (waiting for CPU)
# b  = blocked processes (waiting for I/O)  ← b > 0 = I/O проблема
# si/so = swap in/out (должны быть 0!)
# wa = I/O wait %

# iostat — disk I/O
iostat -x 1
# Device    r/s    w/s   rkB/s   wkB/s  await  %util
# sda       50     200   400     1600    5.2    78.5
#
# %util > 90% → диск перегружен
# await > 10ms (SSD) или > 20ms (HDD) → медленно
# r/s + w/s = IOPS

# iotop — I/O по процессам (аналог top для disk)
iotop -o    # только процессы с активным I/O
```

## dmesg / journalctl

```bash
# dmesg — kernel ring buffer
dmesg | tail -20                  # последние сообщения ядра
dmesg | grep -i error             # ошибки
dmesg | grep -i "oom\|killed"     # OOM kills
dmesg | grep -i "segfault"        # segmentation faults
dmesg -T                          # human-readable timestamps

# journalctl — systemd logs
journalctl -u myapp -f            # follow logs сервиса
journalctl -u myapp --since "1h ago"
journalctl -u myapp -p err        # только ошибки
journalctl -k                     # kernel messages
journalctl --disk-usage           # сколько места занимают логи
journalctl --vacuum-size=500M     # очистить до 500MB
```

## perf

```bash
# CPU profiling (sampling)
perf top                          # real-time hot functions
perf record -g -p <pid> -- sleep 30
perf report                       # interactive report

# Flame graph
perf script | stackcollapse-perf.pl | flamegraph.pl > flame.svg

# Go: лучше использовать pprof (нативный)
# Но perf полезен для system-level (syscalls, kernel)
```

## Комплексная диагностика

```
"Сервис тормозит" — план действий:

1. top/htop: CPU, memory, load average
   → CPU 100%? → perf top / pprof
   → Memory 100%? → OOM? dmesg | grep oom
   → Load > cores? → много процессов в queue

2. vmstat 1: swap, I/O wait
   → wa > 20%? → disk проблема → iostat
   → si/so > 0? → swapping → нужно больше RAM

3. iostat -x 1: disk utilization
   → %util > 90%? → disk saturated
   → await > 50ms? → slow disk

4. ss -s: сетевые соединения
   → Много TIME_WAIT? → connection pooling
   → Много CLOSE_WAIT? → не закрываем соединения

5. strace -c -p <pid>: где время?
   → Какой syscall занимает больше всего?

6. journalctl -u myapp: ошибки в логах?
```
