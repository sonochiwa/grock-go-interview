# pprof: CPU профилирование

## Обзор

pprof — встроенный профайлер Go. CPU profile показывает, где программа тратит время.

## Способы сбора

### 1. В тестах/бенчмарках

```bash
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof
```

### 2. В коде (runtime/pprof)

```go
import "runtime/pprof"

f, _ := os.Create("cpu.prof")
pprof.StartCPUProfile(f)
defer pprof.StopCPUProfile()
// ... код ...
```

### 3. HTTP endpoint (net/http/pprof)

```go
import _ "net/http/pprof"

go func() {
    http.ListenAndServe("localhost:6060", nil)
}()
```

```bash
# Собрать 30-секундный профиль
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
```

## Анализ

```bash
go tool pprof cpu.prof

# Команды внутри pprof:
(pprof) top 20           # топ 20 функций по CPU
(pprof) list funcName    # исходный код с аннотациями
(pprof) web              # flame graph в браузере
(pprof) png > cpu.png    # сохранить граф

# Веб-интерфейс (рекомендуется)
go tool pprof -http=:8080 cpu.prof
```

### Чтение flame graph

```
Ширина = время CPU
  Чем шире блок — тем больше CPU он потребляет
  Вложенность = стек вызовов
  Снизу вверх = от main до самой глубокой функции
```

### flat vs cum

- **flat**: время только в этой функции (без вызовов)
- **cum** (cumulative): время включая все вызовы из неё

```
flat    flat%   cum     cum%   function
2.50s   25.00%  5.00s   50.00% pkg.HotFunction
// HotFunction сама занимает 25% CPU,
// но вместе с вызовами из неё — 50%
```

## Частые вопросы на собеседованиях

**Q: Как профилировать CPU в продакшене?**
A: `net/http/pprof` endpoint + `go tool pprof`. CPU profiling имеет ~5% overhead.

**Q: Что показывает flame graph?**
A: Стек вызовов, где ширина = время CPU. Позволяет быстро найти hotspots.
