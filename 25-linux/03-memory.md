# Memory

## Virtual Memory

```
Каждый процесс видит свое изолированное адресное пространство (virtual addresses)
MMU (Memory Management Unit) транслирует virtual → physical

Virtual Address Space (64-bit):
  ┌──────────────────┐ 0xFFFFFFFFFFFFFFFF
  │   Kernel Space   │
  ├──────────────────┤ 0x7FFFFFFFFFFF
  │      Stack       │ ↓ растёт вниз
  │                  │
  │      ↕ gap       │
  │                  │
  │      Heap        │ ↑ растёт вверх (malloc/mmap)
  ├──────────────────┤
  │   mmap region    │ shared libraries, файлы
  ├──────────────────┤
  │   BSS (uninit)   │ глобальные переменные (zeroed)
  │   Data (init)    │ инициализированные глобальные
  │   Text (code)    │ исполняемый код (read-only)
  └──────────────────┘ 0x0000000000000000

Преимущества virtual memory:
  - Изоляция процессов (один не видит память другого)
  - Каждый процесс думает что у него вся память
  - Lazy allocation: virtual memory > physical memory
  - Memory-mapped files
  - Shared libraries: одна копия libc в физической памяти
```

## Page Table

```
Virtual → Physical mapping:
  Page = 4 KB (default)
  Huge Pages = 2 MB или 1 GB

  Virtual addr → Page Table → Physical addr (или Page Fault)

  4-level page table (x86-64):
  PGD → PUD → PMD → PTE → Physical Page

TLB (Translation Lookaside Buffer):
  Кэш page table в CPU
  TLB hit: ~1 cycle
  TLB miss: ~100 cycles (walk page table)
  Huge pages → меньше записей в TLB → меньше TLB misses

Page Fault:
  Minor: страница в памяти но нет mapping → обновить PTE (дёшево)
  Major: страница на диске (swap) → загрузить с диска (дорого!)

  Go: mmap для аллокации heap → minor page faults при первом доступе
```

## OOM Killer

```
Когда физическая + swap память исчерпана:
  Ядро вызывает OOM Killer → убивает процесс

Выбор жертвы — oom_score:
  /proc/<pid>/oom_score       — текущий score (0-1000)
  /proc/<pid>/oom_score_adj   — ручная настройка (-1000..1000)
  /proc/<pid>/oom_adj         — deprecated

  oom_score_adj = -1000 → никогда не убивать (для критичных сервисов)
  oom_score_adj = 1000  → убить первым

В Kubernetes:
  QoS Guaranteed (limits == requests) → oom_score_adj = -998
  QoS Burstable → oom_score_adj = 2..999
  QoS BestEffort (нет limits) → oom_score_adj = 1000

Диагностика:
  dmesg | grep -i "oom\|killed"
  journalctl -k | grep -i oom
  # "Out of memory: Kill process 12345 (myapp) score 900"

Предотвращение для Go:
  GOMEMLIMIT=900MiB   → GC не даст превысить
  Container limits: 1Gi
  GOMEMLIMIT ≈ 0.9 × container limit
```

## Swap

```
Swap = "виртуальный RAM" на диске
  Когда RAM заполнена → неиспользуемые pages выгружаются на swap

vm.swappiness (0-100):
  0   — swap только при крайней необходимости
  60  — default
  100 — агрессивный swap

Для серверов с Go:
  swappiness = 1 (или swap disabled)
  Go GC лучше работает с памятью в RAM
  Swap + GC = плохая комбинация (GC сканирует pages → page fault → disk I/O)

  # Отключить swap:
  swapoff -a
  # Или в /etc/sysctl.conf:
  vm.swappiness = 1
```

## mmap

```
mmap = отображение файла/памяти в адресное пространство
  Файл → memory region (read/write через указатель, без read()/write())

Типы:
  MAP_PRIVATE: copy-on-write (изменения не пишутся в файл)
  MAP_SHARED:  изменения видны другим процессам и пишутся в файл
  MAP_ANONYMOUS: память без файла (аналог malloc для больших аллокаций)

Go runtime использует mmap:
  - Heap allocation: mmap(MAP_ANONYMOUS) для mheap arena
  - Файлы: os.ReadFile() vs mmap → mmap быстрее для больших файлов

  import "golang.org/x/exp/mmap"
  // или syscall.Mmap() напрямую
```

## Shared Memory

```
IPC (Inter-Process Communication):

  1. Shared memory (shmget/mmap MAP_SHARED):
     Самый быстрый IPC — нет копирования
     Нужна синхронизация (semaphore, futex)

  2. Pipe (|):
     Однонаправленный поток данных
     ls | grep "txt" — stdout ls → stdin grep

  3. Unix Domain Socket:
     Как TCP но через файл
     Быстрее TCP (нет network stack)
     Go: net.Listen("unix", "/tmp/myapp.sock")

  4. Signals: для простых уведомлений (SIGUSR1/2)

Производительность IPC:
  Shared memory > Unix socket > TCP loopback > pipe
```

## Частые вопросы

**Q: Что такое overcommit?**
A: Linux может выделить больше virtual memory чем есть physical+swap. malloc() вернёт OK, но при попытке использовать → OOM kill. Контролируется: vm.overcommit_memory (0=heuristic, 1=always, 2=never).

**Q: Как узнать реальное использование памяти процессом?**
A: RSS (Resident Set Size) = physical memory. Но включает shared pages! PSS (Proportional Set Size) = RSS / количество процессов sharing. USS (Unique Set Size) = только private pages. Смотреть: `/proc/<pid>/smaps_rollup`.

**Q: Copy-on-Write — как работает?**
A: При fork() child получает те же physical pages что и parent (только page table копируется). При записи — page копируется. Экономит память: fork 1GB процесса → мгновенно, без 1GB копирования.
