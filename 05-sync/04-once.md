# sync.Once

## Обзор

Once гарантирует, что функция выполнится ровно один раз, даже при вызове из нескольких горутин одновременно.

## Концепции

```go
var (
    once     sync.Once
    instance *Database
)

func GetDB() *Database {
    once.Do(func() {
        instance = connectToDatabase() // выполнится один раз
    })
    return instance // все вызовы получат тот же instance
}
```

### Once с ошибкой (Go 1.21+)

```go
// sync.OnceFunc — выполняет функцию один раз
init := sync.OnceFunc(func() {
    setupExpensive()
})
init() // выполнит setup
init() // ничего не делает

// sync.OnceValue — один раз с результатом
getConfig := sync.OnceValue(func() *Config {
    return loadConfig()
})
cfg := getConfig()

// sync.OnceValues — один раз с результатом и ошибкой
getDB := sync.OnceValues(func() (*sql.DB, error) {
    return sql.Open("postgres", dsn)
})
db, err := getDB()
```

### Под капотом

```go
// Двойная проверка: atomic + mutex
type Once struct {
    done atomic.Uint32
    m    Mutex
}

func (o *Once) Do(f func()) {
    if o.done.Load() == 0 { // быстрый путь (atomic read)
        o.doSlow(f)
    }
}

func (o *Once) doSlow(f func()) {
    o.m.Lock()
    defer o.m.Unlock()
    if o.done.Load() == 0 { // повторная проверка под локом
        defer o.done.Store(1)
        f()
    }
}
// Если f паникует — Once считается выполненным! Повторные вызовы не будут вызывать f.
```

## Частые вопросы на собеседованиях

**Q: Что произойдёт если функция в Once.Do паникует?**
A: Once считается выполненным. Повторные вызовы Do не вызовут функцию. Используй OnceValues если нужна обработка ошибки.

**Q: Как Once реализован?**
A: Double-checked locking: atomic load для быстрого пути, mutex для медленного.
