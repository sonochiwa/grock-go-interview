# Factory

## В Go

Factory — функция-конструктор, возвращающая конкретный тип или интерфейс. В Go это просто `NewXxx()` функции.

```go
// Simple Factory
type Storage interface {
    Save(key string, value []byte) error
    Load(key string) ([]byte, error)
}

func NewStorage(storageType string) Storage {
    switch storageType {
    case "redis":
        return &RedisStorage{...}
    case "postgres":
        return &PostgresStorage{...}
    case "memory":
        return &MemoryStorage{...}
    default:
        return &MemoryStorage{} // fallback
    }
}
```

### Factory с конфигурацией

```go
type ServerConfig struct {
    Host    string
    Port    int
    TLS     bool
    Timeout time.Duration
}

func NewServer(cfg ServerConfig) *Server {
    s := &Server{
        host:    cfg.Host,
        port:    cfg.Port,
        timeout: cfg.Timeout,
    }
    if cfg.TLS {
        s.setupTLS()
    }
    return s
}
```

### Factory method через интерфейс

```go
type Parser interface {
    Parse(data []byte) (Document, error)
}

type ParserFactory func(config Config) Parser

// Регистрация парсеров
var parsers = map[string]ParserFactory{
    "json": func(cfg Config) Parser { return &JSONParser{cfg} },
    "xml":  func(cfg Config) Parser { return &XMLParser{cfg} },
    "yaml": func(cfg Config) Parser { return &YAMLParser{cfg} },
}

func GetParser(format string, cfg Config) (Parser, error) {
    factory, ok := parsers[format]
    if !ok {
        return nil, fmt.Errorf("unknown format: %s", format)
    }
    return factory(cfg), nil
}
```

### В Go factory — это просто функция

В отличие от Java/C#, не нужен AbstractFactoryInterface. `NewXxx()` — идиоматичный Go.
