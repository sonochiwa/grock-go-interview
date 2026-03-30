# 09. Внутреннее устройство Go

Как всё работает под капотом. Именно эти знания отличают middle от senior на собеседованиях.

## Содержание

1. [Слайсы](01-slice-internals.md) — SliceHeader, алгоритм роста
2. [Map (classic)](02-map-internals-classic.md) — bucket-based до Go 1.24
3. [Map (Swiss Table)](03-map-internals-swiss.md) — Swiss Table с Go 1.24
4. [Строки](04-string-internals.md) — StringHeader, иммутабельность
5. [Каналы](05-channel-internals.md) — hchan, sendq/recvq
6. [Интерфейсы](06-interface-internals.md) — iface/eface (см. также 02-interfaces)
7. [Scheduler](07-scheduler.md) — GMP модель, work stealing
8. [GC (classic)](08-gc-classic.md) — tri-color mark-sweep
9. [GC (Green Tea)](09-gc-green-tea.md) — Go 1.26
10. [Memory Allocator](10-memory-allocator.md) — mcache, mcentral, mheap
