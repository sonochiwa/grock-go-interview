# Атомарные операции

## Обзор

Atomic операции — lock-free альтернатива mutex для простых операций (счётчики, флаги, указатели). Быстрее mutex, но ограничены.

## Концепции

### Старый стиль (функции)

```go
import "sync/atomic"

var counter int64

atomic.AddInt64(&counter, 1)           // атомарный инкремент
val := atomic.LoadInt64(&counter)      // атомарное чтение
atomic.StoreInt64(&counter, 100)       // атомарная запись
swapped := atomic.CompareAndSwapInt64(&counter, 100, 200) // CAS
```

### Новый стиль (типы, Go 1.19+)

```go
var counter atomic.Int64
counter.Add(1)              // инкремент
val := counter.Load()       // чтение
counter.Store(100)          // запись
swapped := counter.CompareAndSwap(100, 200) // CAS

var flag atomic.Bool
flag.Store(true)
if flag.Load() { ... }

// atomic.Pointer[T] — типобезопасный атомарный указатель
var configPtr atomic.Pointer[Config]
configPtr.Store(&Config{Debug: true})
cfg := configPtr.Load()
```

### atomic.Value (legacy)

```go
var config atomic.Value

// Запись (тип фиксируется после первого Store)
config.Store(&Config{Debug: true})

// Чтение
cfg := config.Load().(*Config)

// Предпочитай atomic.Pointer[T] в новом коде — типобезопасно
```

### CAS (Compare-And-Swap)

```go
// Основа lock-free алгоритмов
// "Если значение == old, замени на new; верни успешность"
var state atomic.Int32

// Спин-лок (пример, не для продакшена)
for !state.CompareAndSwap(0, 1) {
    runtime.Gosched() // уступить процессор
}
// критическая секция
state.Store(0) // освободить
```

## Когда atomic vs Mutex

| Atomic | Mutex |
|---|---|
| Один счётчик/флаг | Несколько связанных полей |
| Простые операции (inc, swap) | Сложная логика |
| Максимальная производительность | Читаемость важнее |
| Lock-free | Может блокировать |

## Частые вопросы на собеседованиях

**Q: Чем atomic.Int64 лучше atomic.AddInt64?**
A: Типобезопасность, нет работы с указателями, метод вместо функции, нельзя случайно использовать не-atomic доступ.

**Q: Что такое CAS?**
A: Compare-And-Swap. Атомарно: если текущее значение == expected, заменить на new. Основа lock-free алгоритмов.

**Q: Гарантирует ли atomic ordering?**
A: Да. Go memory model гарантирует sequential consistency для atomic операций.
