# Chain of Responsibility

## В Go

Цепочка обработчиков, где каждый решает: обработать или передать следующему. HTTP middleware — классический пример.

```go
type Middleware func(http.Handler) http.Handler

func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
    // Применяем в обратном порядке
    for i := len(middlewares) - 1; i >= 0; i-- {
        handler = middlewares[i](handler)
    }
    return handler
}

func Logging(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("→ %s %s", r.Method, r.URL)
        next.ServeHTTP(w, r)
        log.Printf("← %s %s", r.Method, r.URL)
    })
}

func Auth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !isAuthenticated(r) {
            http.Error(w, "Unauthorized", 401)
            return // прерываем цепочку
        }
        next.ServeHTTP(w, r) // передаём дальше
    })
}

// Request → Logging → Auth → Handler
handler := Chain(myHandler, Logging, Auth)
```
