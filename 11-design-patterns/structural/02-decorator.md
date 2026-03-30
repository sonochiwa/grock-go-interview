# Decorator

## В Go

Decorator оборачивает объект, добавляя поведение. В Go — через обёртки интерфейсов (middleware pattern).

```go
// io.Reader decorator: добавляет логирование
type LoggingReader struct {
    r      io.Reader
    logger *slog.Logger
}

func NewLoggingReader(r io.Reader, logger *slog.Logger) *LoggingReader {
    return &LoggingReader{r: r, logger: logger}
}

func (lr *LoggingReader) Read(p []byte) (int, error) {
    n, err := lr.r.Read(p)
    lr.logger.Info("read", "bytes", n, "error", err)
    return n, err
}

// Чейнинг декораторов:
var r io.Reader = file
r = NewLoggingReader(r, logger)
r = io.LimitReader(r, 1024)
r = bufio.NewReader(r)
```

### HTTP Decorator (middleware)

```go
func WithLogging(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        log.Printf("%s %s %v", r.Method, r.URL, time.Since(start))
    })
}

func WithAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !isAuthed(r) {
            http.Error(w, "unauthorized", 401)
            return
        }
        next.ServeHTTP(w, r)
    })
}

// Чейнинг: WithLogging(WithAuth(handler))
```
