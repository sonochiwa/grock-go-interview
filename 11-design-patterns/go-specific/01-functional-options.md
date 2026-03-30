# Functional Options

## Обзор

Самый идиоматичный Go-паттерн для конфигурации. Придуман Dave Cheney и Rob Pike.

```go
type Server struct {
    host     string
    port     int
    timeout  time.Duration
    maxConns int
    tls      bool
}

type Option func(*Server)

func WithTimeout(d time.Duration) Option {
    return func(s *Server) { s.timeout = d }
}

func WithMaxConns(n int) Option {
    return func(s *Server) { s.maxConns = n }
}

func WithTLS() Option {
    return func(s *Server) { s.tls = true }
}

func NewServer(host string, port int, opts ...Option) *Server {
    s := &Server{
        host:     host,
        port:     port,
        timeout:  30 * time.Second, // defaults
        maxConns: 100,
    }
    for _, opt := range opts {
        opt(s)
    }
    return s
}

// Использование — чисто и расширяемо
srv := NewServer("localhost", 8080,
    WithTimeout(10*time.Second),
    WithTLS(),
)
```

## Преимущества

- Читаемый API: `WithTLS()` понятнее `true` в позиционном аргументе
- Расширяемо: новые опции не ломают существующий код
- Значения по умолчанию: явно определены в конструкторе
- Самодокументируемо: имена опций описывают эффект

## С валидацией

```go
type Option func(*Server) error

func WithPort(port int) Option {
    return func(s *Server) error {
        if port < 0 || port > 65535 {
            return fmt.Errorf("invalid port: %d", port)
        }
        s.port = port
        return nil
    }
}

func NewServer(opts ...Option) (*Server, error) {
    s := &Server{port: 8080}
    for _, opt := range opts {
        if err := opt(s); err != nil {
            return nil, err
        }
    }
    return s, nil
}
```

## Реальные примеры

```go
// gRPC
grpc.NewServer(
    grpc.MaxRecvMsgSize(1024*1024),
    grpc.UnaryInterceptor(myInterceptor),
)

// zap logger
zap.New(core,
    zap.AddCaller(),
    zap.AddStacktrace(zapcore.ErrorLevel),
)
```
