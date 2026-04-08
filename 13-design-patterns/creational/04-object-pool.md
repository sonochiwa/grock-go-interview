# Object Pool

## В Go

sync.Pool — встроенный object pool. Подробнее в 05-sync/05-pool.md.

```go
var bufPool = sync.Pool{
    New: func() any { return new(bytes.Buffer) },
}

func process() {
    buf := bufPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufPool.Put(buf)
    }()
    // использовать buf
}
```

Для пулов соединений — `database/sql.DB` (connection pool встроен).
