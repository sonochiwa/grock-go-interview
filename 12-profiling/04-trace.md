# runtime/trace

## Обзор

Trace показывает timeline выполнения горутин, GC, scheduling events. В отличие от pprof (sampling), trace записывает каждое событие.

## Сбор

```bash
# В тестах
go test -trace=trace.out

# HTTP
curl -o trace.out http://localhost:6060/debug/pprof/trace?seconds=5

# В коде
f, _ := os.Create("trace.out")
trace.Start(f)
defer trace.Stop()
```

## Анализ

```bash
go tool trace trace.out
# Откроется в браузере: timeline, goroutine analysis, network blocking, etc.
```

## Что видно в trace

- Горутины: создание, blocking, unblocking, scheduling
- GC: паузы, фазы, STW
- Syscalls: блокировки на I/O
- Network: polling events
- Scheduler: work stealing, P utilization

## Когда trace, а не pprof

| pprof | trace |
|---|---|
| Где тратится CPU? | Почему горутины ждут? |
| Где аллоцируется память? | Как горутины взаимодействуют? |
| Агрегированная статистика | Timeline событий |
| Низкий overhead (~5%) | Высокий overhead (значительный) |
