# Adapter

## В Go

Adapter приводит один интерфейс к другому. В Go часто реализуется через обёртку или type conversion.

```go
// Адаптер: http.HandlerFunc адаптирует функцию к интерфейсу Handler
type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}

type HandlerFunc func(ResponseWriter, *Request)

func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
    f(w, r) // функция вызывает сама себя как метод
}

// Теперь обычная функция реализует Handler:
http.Handle("/", http.HandlerFunc(myFunc))
```

### Adapter для внешних библиотек

```go
// Наш интерфейс
type Logger interface {
    Info(msg string, args ...any)
    Error(msg string, args ...any)
}

// Внешний логгер с другим API
type ExternalLogger struct { ... }
func (l *ExternalLogger) Log(level int, message string) { ... }

// Адаптер
type LoggerAdapter struct {
    ext *ExternalLogger
}

func (a *LoggerAdapter) Info(msg string, args ...any) {
    a.ext.Log(0, fmt.Sprintf(msg, args...))
}

func (a *LoggerAdapter) Error(msg string, args ...any) {
    a.ext.Log(1, fmt.Sprintf(msg, args...))
}
```
