# Functional Options

Реализуй паттерн Functional Options для `Server`:

```go
type Server struct {
    host     string        // default: "localhost"
    port     int           // default: 8080
    timeout  time.Duration // default: 30s
    maxConns int           // default: 100
}
```

Создай: `NewServer(opts ...Option) *Server` и опции `WithHost`, `WithPort`, `WithTimeout`, `WithMaxConns`.
