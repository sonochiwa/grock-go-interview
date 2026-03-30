# I/O и Networking Optimization

## Connection Pooling

```go
// database/sql — встроенный пул
db, _ := sql.Open("postgres", dsn)
db.SetMaxOpenConns(25)          // макс соединений к БД
db.SetMaxIdleConns(10)          // макс idle соединений
db.SetConnMaxLifetime(5 * time.Minute) // пересоздавать после 5 мин
db.SetConnMaxIdleTime(1 * time.Minute) // закрыть idle после 1 мин

// Формула для MaxOpenConns:
// connections = (core_count * 2) + effective_spindle_count
// Для SSD: ~25-50 соединений на один инстанс БД
// Для несколькихи pod: MaxOpenConns = DB_max_connections / num_pods

// HTTP client — переиспользуй (внутри http.Transport = пул)
var httpClient = &http.Client{
    Timeout: 10 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        MaxConnsPerHost:     25,
        IdleConnTimeout:     90 * time.Second,
    },
}
// ❌ http.Get("...") — создаёт новый client каждый раз
// ✅ httpClient.Get("...") — переиспользует соединения

// Redis — встроенный пул в go-redis
client := redis.NewClient(&redis.Options{
    Addr:         "localhost:6379",
    PoolSize:     10,
    MinIdleConns: 5,
    PoolTimeout:  30 * time.Second,
})

// gRPC — одно соединение мультиплексирует через HTTP/2
conn, _ := grpc.NewClient(addr, opts...)
// Для высокой нагрузки — несколько connections:
// grpc.WithDefaultServiceConfig для round-robin between connections
```

## Buffered I/O

```go
// ❌ Много мелких записей
for _, line := range lines {
    file.Write([]byte(line + "\n")) // syscall на каждую строку!
}

// ✅ Буферизация
w := bufio.NewWriterSize(file, 64*1024) // 64KB buffer
for _, line := range lines {
    w.WriteString(line)
    w.WriteByte('\n')
}
w.Flush() // один или несколько syscalls

// ✅ Буферизированное чтение
r := bufio.NewReaderSize(file, 64*1024)
scanner := bufio.NewScanner(r)
scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // для больших строк
for scanner.Scan() {
    process(scanner.Text())
}
```

## HTTP Response Streaming

```go
// ❌ Буферизация всего ответа в памяти
func handler(w http.ResponseWriter, r *http.Request) {
    var buf bytes.Buffer
    for row := range rows {
        json.NewEncoder(&buf).Encode(row) // всё в памяти
    }
    w.Write(buf.Bytes())
}

// ✅ Streaming (NDJSON)
func handler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/x-ndjson")
    flusher, _ := w.(http.Flusher)

    enc := json.NewEncoder(w)
    for row := range rows {
        enc.Encode(row)  // пишет напрямую в ResponseWriter
        flusher.Flush()  // отправляет клиенту сразу
    }
}
```

## Compression

```go
// gzip middleware (для REST API)
import "github.com/klauspost/compress/gzhttp"

handler = gzhttp.GzipHandler(handler)

// Или manual
func gzipMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
            next.ServeHTTP(w, r)
            return
        }
        w.Header().Set("Content-Encoding", "gzip")
        gz := gzip.NewWriter(w)
        defer gz.Close()
        next.ServeHTTP(&gzipResponseWriter{Writer: gz, ResponseWriter: w}, r)
    })
}

// Для Kafka: compression на уровне producer
config.Producer.Compression = sarama.CompressionLZ4 // или Snappy, Zstd
```

## Pipelining и Batch Operations

```go
// Redis pipeline — одна round-trip для нескольких команд
pipe := client.Pipeline()
for _, key := range keys {
    pipe.Get(ctx, key)
}
results, err := pipe.Exec(ctx)
// 100 GET за одну round-trip вместо 100

// HTTP/2 multiplexing — несколько запросов на одном соединении
// Автоматически с http.Client + HTTP/2 server

// Batch API design
// ❌ GET /users/1, GET /users/2, GET /users/3
// ✅ GET /users?ids=1,2,3 или POST /users/batch {"ids": [1,2,3]}
```

## Timeouts everywhere

```go
// Каждый I/O вызов должен иметь timeout
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()

// DB
db.QueryContext(ctx, query, args...)

// HTTP
req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
resp, err := httpClient.Do(req)

// Redis
client.Get(ctx, key)

// DNS
net.Resolver{}.LookupHost(ctx, host)

// Без timeout → goroutine leak при зависшем upstream
```
