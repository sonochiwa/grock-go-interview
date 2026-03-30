# pprof: Memory профилирование

## Обзор

Memory profile показывает, где программа аллоцирует память. Два вида: inuse (текущее использование) и alloc (все аллокации).

## Сбор

```bash
# В тестах
go test -memprofile=mem.prof -bench=.

# HTTP
go tool pprof http://localhost:6060/debug/pprof/heap

# В коде
runtime.GC() // для точности
pprof.WriteHeapProfile(f)
```

## Анализ

```bash
# Текущее использование (default)
go tool pprof -inuse_space mem.prof

# Все аллокации (для GC pressure)
go tool pprof -alloc_objects mem.prof
go tool pprof -alloc_space mem.prof

# Команды
(pprof) top 20 -cum     # топ по cumulative
(pprof) list funcName   # исходный код
```

### Типы memory profile

| Тип | Что показывает | Когда использовать |
|---|---|---|
| inuse_space | Байты в куче сейчас | Утечки памяти |
| inuse_objects | Объекты в куче сейчас | Количество объектов |
| alloc_space | Всего аллоцировано байт | GC pressure |
| alloc_objects | Всего аллокаций | GC pressure |

## Типичные оптимизации

```go
// 1. sync.Pool для переиспользования буферов
// 2. strings.Builder вместо конкатенации
// 3. Предаллокация слайсов: make([]T, 0, expectedSize)
// 4. Избегать boxing (interface{}) для hot paths
// 5. Pointer receiver для больших структур
```

## Частые вопросы на собеседованиях

**Q: Чем alloc_objects отличается от inuse_objects?**
A: alloc — все аллокации за время профилирования. inuse — текущий heap. alloc показывает давление на GC, inuse — утечки.
