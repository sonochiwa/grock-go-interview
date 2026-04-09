# Processes

## fork / exec / clone

```
fork():
  Создаёт копию текущего процесса
  Child = точная копия parent (memory, fd, env)
  Copy-on-Write: физическая память копируется только при записи
  Возвращает: 0 в child, PID child в parent

exec():
  Заменяет текущий процесс новой программой
  PID не меняется, но код/данные/стек — новые
  open fd наследуются (если нет CLOEXEC)

fork() + exec() = стандартный способ запуска программ
  1. fork() → child
  2. child: exec("./myapp") → заменить код на myapp
  3. parent: waitpid(child) → ждать завершения

clone():
  Обобщение fork — можно выбрать что разделять
  clone(CLONE_VM | CLONE_FS | ...) → потоки (threads)
  clone(CLONE_NEWPID | CLONE_NEWNS | ...) → контейнеры (namespaces)
  Go runtime: clone() для goroutine M (OS threads)
```

## PID, PPID, Process Tree

```
PID 1 = init (systemd)
  └── PID 100 = sshd
       └── PID 200 = bash
            └── PID 300 = ./myapp
                 ├── PID 301 (thread)
                 └── PID 302 (thread)

Полезные команды:
  ps aux               — все процессы
  ps -ef --forest       — дерево процессов
  pstree -p             — красивое дерево с PID
  /proc/<pid>/status    — подробная инфо о процессе
  /proc/<pid>/cmdline   — команда запуска
  /proc/<pid>/fd/       — открытые файловые дескрипторы
```

## Zombie и Orphan процессы

```
Zombie (defunct):
  Child завершился, но parent не вызвал wait()
  Запись в таблице процессов остаётся (PID, exit code)
  Не занимает память/CPU, но занимает PID

  $ ps aux | grep Z
  user  12345  0.0  0.0  0  0 ?  Z  10:00  0:00 [myapp] <defunct>

  Решение:
  - Parent должен вызвать waitpid()
  - В Go: cmd.Wait() или cmd.Process.Wait()
  - Если parent игнорирует → убить parent → init подберёт zombie

  В Docker:
  PID 1 = ваше приложение (не init!)
  Если child создаёт subprocess → zombie
  Решение: tini (--init flag) или dumb-init

Orphan:
  Parent умер раньше child
  Child усыновляется PID 1 (init/systemd)
  init автоматически вызывает wait() → не проблема

  Go: горутина с exec.Command запускает child process
  Если Go процесс убит SIGKILL → child = orphan
```

## Signals

```
│ Signal  │ Номер │ Default     │ Назначение                    │
├─────────┼───────┼─────────────┼───────────────────────────────┤
│ SIGHUP  │ 1     │ Terminate   │ Terminal closed (reload конфиг)│
│ SIGINT  │ 2     │ Terminate   │ Ctrl+C                        │
│ SIGQUIT │ 3     │ Core dump   │ Ctrl+\ (с дампом)             │
│ SIGKILL │ 9     │ Terminate   │ Немедленное убийство (нельзя  │
│         │       │             │ перехватить!)                  │
│ SIGSEGV │ 11    │ Core dump   │ Segfault                      │
│ SIGPIPE │ 13    │ Terminate   │ Write to closed pipe/socket   │
│ SIGTERM │ 15    │ Terminate   │ Graceful shutdown (default kill)│
│ SIGCHLD │ 17    │ Ignore      │ Child process завершился      │
│ SIGSTOP │ 19    │ Stop        │ Заморозить (нельзя перехватить)│
│ SIGCONT │ 18    │ Continue    │ Продолжить после STOP         │
│ SIGUSR1 │ 10    │ Terminate   │ User-defined                  │
│ SIGUSR2 │ 12    │ Terminate   │ User-defined                  │

В Go:
  // Graceful shutdown
  ctx, stop := signal.NotifyContext(context.Background(),
      syscall.SIGINT, syscall.SIGTERM)
  defer stop()

  // SIGPIPE: Go runtime игнорирует по умолчанию (не crash на broken pipe)
  // SIGQUIT: Go печатает goroutine dump (полезно для debug)

kill -0 <pid>   — проверить существование процесса (без сигнала)
kill -15 <pid>  — SIGTERM (graceful)
kill -9 <pid>   — SIGKILL (force, последнее средство!)
```

## Daemon / systemd

```
Systemd unit file (/etc/systemd/system/myapp.service):

[Unit]
Description=My Go Application
After=network.target postgresql.service
Wants=postgresql.service

[Service]
Type=simple
User=myapp
Group=myapp
WorkingDirectory=/opt/myapp
ExecStart=/opt/myapp/server
ExecReload=/bin/kill -HUP $MAINPID
Restart=on-failure
RestartSec=5
LimitNOFILE=65535
Environment=GOMAXPROCS=4
Environment=GOMEMLIMIT=900MiB

# Security
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/var/lib/myapp

[Install]
WantedBy=multi-user.target

Команды:
  systemctl start myapp
  systemctl stop myapp       # SIGTERM → ждёт → SIGKILL
  systemctl restart myapp
  systemctl reload myapp     # SIGHUP
  systemctl status myapp
  systemctl enable myapp     # autostart
  journalctl -u myapp -f     # логи
  journalctl -u myapp --since "1 hour ago"
```

## Process States

```
R — Running / Runnable (в run queue)
S — Sleeping (interruptible, ждёт I/O)
D — Disk sleep (uninterruptible, ждёт disk I/O)
    НЕ убивается SIGKILL! Нужно ждать завершения I/O
Z — Zombie (завершился, parent не wait())
T — Stopped (SIGSTOP / debugging)
I — Idle (kernel thread)

Проблема: много процессов в D state
  → Диск перегружен (iostat)
  → NFS зависла (mount -o soft vs hard)
  → Kernel bug
```

## Частые вопросы

**Q: Что происходит при kill -9?**
A: SIGKILL доставляется ядром напрямую, процесс НЕ МОЖЕТ его перехватить. Немедленное завершение без cleanup (нет defer, нет graceful shutdown, нет flush). Используй SIGTERM первым.

**Q: Чем процесс отличается от потока?**
A: Процесс — изолированное адресное пространство (свои memory, fd). Поток — разделяет address space с другими потоками процесса. В Linux оба = task_struct, отличаются флагами clone().

**Q: Как Go создаёт OS threads?**
A: runtime использует clone() с CLONE_VM | CLONE_FS | CLONE_FILES. M (machine) = OS thread. Горутины мультиплексируются на M. GOMAXPROCS определяет сколько M работают одновременно.
