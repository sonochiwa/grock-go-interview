# Containers Under the Hood

## Контейнер = namespaces + cgroups + rootfs

```
Docker container НЕ является виртуальной машиной!
Это обычный Linux процесс с изоляцией через:

1. Namespaces → ЧТО видит процесс (изоляция)
2. Cgroups    → СКОЛЬКО ресурсов может использовать (лимиты)
3. rootfs     → файловая система (overlay FS)
```

## Namespaces (изоляция)

```
│ Namespace │ Изолирует                      │ Flag           │
├───────────┼────────────────────────────────┼────────────────┤
│ PID       │ Process IDs (свой PID 1)       │ CLONE_NEWPID   │
│ NET       │ Network stack (interfaces, IP)  │ CLONE_NEWNET   │
│ MNT       │ Mount points (filesystem view)  │ CLONE_NEWNS    │
│ UTS       │ Hostname, domain name           │ CLONE_NEWUTS   │
│ IPC       │ Shared memory, semaphores       │ CLONE_NEWIPC   │
│ USER      │ UID/GID mappings (rootless)     │ CLONE_NEWUSER  │
│ CGROUP    │ Cgroup root directory            │ CLONE_NEWCGROUP│

Как создать "контейнер" вручную:
  # unshare = запустить процесс в новых namespaces
  unshare --pid --net --mount --uts --ipc --fork bash

  # В новом namespace:
  hostname container-1        # UTS: свой hostname
  ps aux                       # PID: только свои процессы
  ip addr                      # NET: пусто (нет interfaces)

Docker делает:
  clone(CLONE_NEWPID | CLONE_NEWNET | CLONE_NEWNS | ...)
  → процесс в изолированных namespaces
```

## Cgroups v2 (лимиты ресурсов)

```
Cgroups = Control Groups

Контролирует:
  cpu      — CPU time (shares, quota)
  memory   — RAM usage + swap
  io       — Disk I/O bandwidth
  pids     — Максимум процессов
  cpuset   — Привязка к конкретным CPU cores

Cgroups v2 (unified hierarchy):
  /sys/fs/cgroup/
  ├── system.slice/
  │   └── docker-<id>.scope/
  │       ├── cpu.max         → "100000 100000" (quota/period)
  │       ├── memory.max      → 536870912 (512MB)
  │       ├── memory.current  → текущее использование
  │       ├── pids.max        → 100
  │       └── io.max          → disk I/O лимиты

CPU лимит:
  cpu.max = "50000 100000"
  → 50ms из каждых 100ms = 50% CPU = 0.5 cores
  Docker: --cpus=0.5

Memory лимит:
  memory.max = 536870912  (512 MB)
  Превышение → OOM kill внутри cgroup
  Docker: --memory=512m

  memory.high = soft limit (замедление GC pressure)
  memory.max = hard limit (OOM kill)

Kubernetes:
  resources.requests → scheduling (гарантия)
  resources.limits → cgroup limits (hard cap)
```

## Overlay Filesystem

```
Docker images = слои (layers)

Base image:     [alpine]            layer 1 (read-only)
Install app:    [+ go binary]       layer 2 (read-only)
Config:         [+ config.yaml]     layer 3 (read-only)
Container:      [container layer]   layer 4 (read-write)

OverlayFS:
  Lower dirs: readonly layers (image)
  Upper dir:  writable layer (container)
  Merged dir: unified view

  mount -t overlay overlay \
    -o lowerdir=/layer3:/layer2:/layer1,upperdir=/container,workdir=/work \
    /merged

Copy-on-Write:
  Чтение: файл берётся из нижнего слоя (нет копирования)
  Запись: файл копируется в upper layer, потом модифицируется
  Удаление: "whiteout" file в upper layer

Практические следствия:
  - docker build → каждый RUN = новый layer
  - Порядок COPY важен: часто меняющиеся файлы — в конце
  - go mod download отдельно от COPY . → кэшируется
```

## Как Docker запускает контейнер

```
docker run -d --name myapp -p 8080:8080 --memory=512m myimage:latest

1. Pull image (если нет локально)
   → Скачать layers из registry

2. Подготовить rootfs
   → OverlayFS: lower=image layers, upper=container layer

3. Создать namespaces
   → clone(CLONE_NEWPID | CLONE_NEWNET | CLONE_NEWNS | ...)

4. Настроить cgroups
   → memory.max=512MB, cpu.max=...

5. Настроить networking
   → veth pair → bridge (docker0) → iptables NAT
   → Port mapping: iptables -t nat PREROUTING -p tcp --dport 8080 → container:8080

6. Настроить rootfs
   → pivot_root → container видит только свою FS
   → Mount /proc, /sys, /dev

7. Запустить entrypoint/cmd
   → exec в контексте нового namespace
   → PID 1 внутри контейнера
```

## Частые вопросы

**Q: Контейнер vs VM?**
A: VM = отдельное ядро + hypervisor (полная изоляция, ~GB overhead). Контейнер = общее ядро + namespaces (лёгкая изоляция, ~MB overhead). Контейнер быстрее стартует, меньше ресурсов, но слабее изоляция.

**Q: Rootless containers?**
A: Контейнер без привилегий root на хосте. User namespace: UID 0 в контейнере = UID 100000 на хосте. Podman: rootless по умолчанию. Docker: можно настроить (userns-remap).

**Q: Почему PID 1 важен в контейнере?**
A: PID 1 получает осиротевшие процессы и должен вызывать wait(). Go binary как PID 1 не делает этого → zombie процессы. Решение: tini / dumb-init как PID 1 (docker run --init).
