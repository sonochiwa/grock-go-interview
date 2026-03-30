# sync.RWMutex

## Обзор

RWMutex — мьютекс для сценария "много читателей, мало писателей". Множество горутин могут читать одновременно, но запись требует эксклюзивного доступа.

## Концепции

```go
type Cache struct {
    mu   sync.RWMutex
    data map[string]string
}

func (c *Cache) Get(key string) (string, bool) {
    c.mu.RLock() // читающий лок — несколько горутин одновременно
    defer c.mu.RUnlock()
    val, ok := c.data[key]
    return val, ok
}

func (c *Cache) Set(key, value string) {
    c.mu.Lock() // пишущий лок — эксклюзивный
    defer c.mu.Unlock()
    c.data[key] = value
}
```

### Правила

- Несколько RLock() одновременно — OK
- Lock() ждёт пока все RLock() будут сняты
- Новые RLock() ждут, пока Lock() в очереди — предотвращает writer starvation
- Не повышай RLock → Lock (deadlock!)

```go
// DEADLOCK: попытка повысить RLock до Lock
c.mu.RLock()
// ...
c.mu.Lock() // DEADLOCK: ждёт снятия RLock, но мы его держим
```

### Когда RWMutex лучше Mutex

- Чтение **значительно** чаще записи (90%+ чтение)
- Критическая секция чтения **не тривиальная** (если она быстрая — overhead RWMutex не окупится)
- Иначе обычный Mutex проще и может быть быстрее

## Частые вопросы на собеседованиях

**Q: Может ли RWMutex голодать writer?**
A: Нет. Go реализует writer-preference: новые RLock() блокируются, если есть ожидающий writer.

**Q: Когда Mutex лучше RWMutex?**
A: Когда запись частая или критическая секция очень короткая. RWMutex имеет бОльший overhead из-за отслеживания читателей.
