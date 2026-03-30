# sync.Mutex

## Обзор

Mutex (mutual exclusion) — примитив для защиты shared state от одновременного доступа. Только одна горутина может держать Lock одновременно.

## Концепции

```go
type SafeCounter struct {
    mu sync.Mutex
    count int
}

func (c *SafeCounter) Increment() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.count++
}

func (c *SafeCounter) Value() int {
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.count
}
```

### Правила

```go
// 1. ВСЕГДА defer Unlock() — защита от паники
c.mu.Lock()
defer c.mu.Unlock()

// 2. Минимальная критическая секция
c.mu.Lock()
val := c.count // только то, что нужно защитить
c.mu.Unlock()
process(val) // это вне лока

// 3. НЕ копируй mutex (go vet ловит)
type Bad struct { mu sync.Mutex }
b1 := Bad{}
b2 := b1 // КОПИРУЕТ mutex! go vet: copies lock

// 4. Mutex НЕ рекурсивный (не reentrant)
func (c *SafeCounter) Double() {
    c.mu.Lock()
    c.mu.Lock() // DEADLOCK! Та же горутина пытается взять лок дважды
}

// 5. Zero value = unlocked (можно использовать без инициализации)
var mu sync.Mutex // готов к использованию
```

### Deadlock

```go
// Классический deadlock: два мьютекса, обратный порядок
var mu1, mu2 sync.Mutex

// Горутина 1          // Горутина 2
mu1.Lock()             mu2.Lock()
mu2.Lock() // ждёт     mu1.Lock() // ждёт
// DEADLOCK!

// Решение: всегда блокировать в одном порядке
// mu1 → mu2 (никогда mu2 → mu1)
```

## Частые вопросы на собеседованиях

**Q: Почему Mutex в Go не рекурсивный?**
A: Design decision. Рекурсивные мьютексы скрывают проблемы в коде. Если нужен повторный вход — рефактори код (выдели внутренний метод без лока).

**Q: Можно ли копировать sync.Mutex?**
A: Нет. Копия может быть в залоченном состоянии. `go vet` обнаруживает это.

**Q: Zero value Mutex — залочен или разлочен?**
A: Разлочен. Можно использовать без инициализации.
