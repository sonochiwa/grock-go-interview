# Builder

## В Go

Builder — поэтапное создание сложного объекта. В Go часто заменяется Functional Options, но Builder тоже используется.

```go
type Server struct {
    host     string
    port     int
    tls      bool
    maxConns int
    timeout  time.Duration
}

type ServerBuilder struct {
    server Server
}

func NewServerBuilder(host string, port int) *ServerBuilder {
    return &ServerBuilder{
        server: Server{host: host, port: port, timeout: 30 * time.Second},
    }
}

func (b *ServerBuilder) WithTLS() *ServerBuilder {
    b.server.tls = true
    return b // fluent interface
}

func (b *ServerBuilder) WithMaxConns(n int) *ServerBuilder {
    b.server.maxConns = n
    return b
}

func (b *ServerBuilder) WithTimeout(d time.Duration) *ServerBuilder {
    b.server.timeout = d
    return b
}

func (b *ServerBuilder) Build() (*Server, error) {
    if b.server.host == "" {
        return nil, errors.New("host is required")
    }
    s := b.server // копия
    return &s, nil
}

// Использование
srv, err := NewServerBuilder("localhost", 8080).
    WithTLS().
    WithMaxConns(100).
    WithTimeout(5 * time.Second).
    Build()
```

### Реальный пример: strings.Builder

```go
var b strings.Builder
b.Grow(100)
b.WriteString("Hello, ")
b.WriteString("World!")
result := b.String()
```

### Builder vs Functional Options

- Builder: fluent interface, валидация в Build(), мутабельный
- Functional Options: функциональный стиль, валидация в конструкторе, идиоматичнее для Go
