# Web Security

## SQL Injection

```go
// ❌ УЯЗВИМО — конкатенация строк
query := "SELECT * FROM users WHERE name = '" + name + "'"
db.Query(query)
// name = "'; DROP TABLE users; --" → катастрофа

// ✅ БЕЗОПАСНО — параметризованные запросы
db.QueryContext(ctx, "SELECT * FROM users WHERE name = $1", name)

// ✅ БЕЗОПАСНО — prepared statements
stmt, _ := db.PrepareContext(ctx, "SELECT * FROM users WHERE id = $1")
stmt.QueryRowContext(ctx, userID)

// ✅ ORM (sqlx, GORM) — автоматически параметризуют
db.Get(&user, "SELECT * FROM users WHERE id = $1", id)

// ❌ Даже с ORM можно ошибиться:
db.Where("name = '" + name + "'").Find(&users)  // RAW STRING!
// ✅
db.Where("name = ?", name).Find(&users)

// LIKE тоже нужно экранировать:
// ❌ db.Query("SELECT * FROM users WHERE name LIKE '%' || $1 || '%'", input)
// Input: "%' OR 1=1 --"
// ✅ Экранировать % и _ в input, потом использовать параметр
```

## XSS (Cross-Site Scripting)

```go
// html/template автоматически экранирует HTML
import "html/template"

tmpl := template.Must(template.New("page").Parse(`
    <h1>Hello, {{.Name}}</h1>
`))
// .Name = "<script>alert('xss')</script>"
// Выведет: &lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;

// ❌ text/template НЕ экранирует!
import "text/template" // опасно для HTML!

// ❌ Ручная конкатенация HTML
w.Write([]byte("<h1>" + userInput + "</h1>"))

// API (JSON): экранировать не нужно, но:
// Content-Type: application/json (не text/html!)
// X-Content-Type-Options: nosniff
```

### Security Headers

```go
func securityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "0") // отключить встроенный (ненадёжный)
        w.Header().Set("Content-Security-Policy",
            "default-src 'self'; script-src 'self'; style-src 'self'")
        w.Header().Set("Strict-Transport-Security",
            "max-age=31536000; includeSubDomains")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        next.ServeHTTP(w, r)
    })
}
```

## CSRF (Cross-Site Request Forgery)

```go
// Для cookie-based auth нужна CSRF защита
// Для token-based (Bearer) — не нужна

import "github.com/gorilla/csrf"

// Middleware
csrfMiddleware := csrf.Protect(
    []byte("32-byte-auth-key-here-1234567890"),
    csrf.Secure(true), // требует HTTPS
)
http.ListenAndServe(":8080", csrfMiddleware(router))

// В template:
// <form>
//   {{ .csrfField }}   ← hidden input с токеном
//   ...
// </form>

// SameSite cookie — современная альтернатива:
http.SetCookie(w, &http.Cookie{
    Name:     "session",
    Value:    sessionID,
    HttpOnly: true,       // недоступен из JS
    Secure:   true,       // только HTTPS
    SameSite: http.SameSiteStrictMode, // не отправлять с cross-site запросами
    Path:     "/",
    MaxAge:   3600,
})
```

## CORS (Cross-Origin Resource Sharing)

```go
func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")

        // Whitelist origins
        allowedOrigins := map[string]bool{
            "https://myapp.com":     true,
            "https://staging.myapp.com": true,
        }

        if allowedOrigins[origin] {
            w.Header().Set("Access-Control-Allow-Origin", origin)
            w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
            w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
            w.Header().Set("Access-Control-Allow-Credentials", "true")
            w.Header().Set("Access-Control-Max-Age", "86400")
        }

        // Preflight
        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusNoContent)
            return
        }

        next.ServeHTTP(w, r)
    })
}

// ❌ НИКОГДА:
// Access-Control-Allow-Origin: *
// Access-Control-Allow-Credentials: true
// (браузер это запретит, но и пробовать не стоит)
```

## Input Validation

```go
// Валидация на входе — ВСЕГДА
type CreateUserRequest struct {
    Name  string `json:"name" validate:"required,min=1,max=100"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"gte=0,lte=150"`
}

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, http.StatusBadRequest, "invalid JSON")
        return
    }
    if err := validate.Struct(req); err != nil {
        respondError(w, http.StatusBadRequest, err.Error())
        return
    }
    // safe to use req.Name, req.Email
}

// Sanitize (если нужно хранить user input):
// - Trim whitespace
// - Normalize unicode
// - Limit length
// - Для HTML: bluemonday sanitizer
```

## Rate Limiting

```go
import "golang.org/x/time/rate"

// Per-IP rate limiter
type IPRateLimiter struct {
    mu       sync.Mutex
    limiters map[string]*rate.Limiter
    rate     rate.Limit
    burst    int
}

func (l *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
    l.mu.Lock()
    defer l.mu.Unlock()

    limiter, exists := l.limiters[ip]
    if !exists {
        limiter = rate.NewLimiter(l.rate, l.burst)
        l.limiters[ip] = limiter
    }
    return limiter
}

func rateLimitMiddleware(limiter *IPRateLimiter) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ip := realIP(r)
            if !limiter.GetLimiter(ip).Allow() {
                w.Header().Set("Retry-After", "60")
                http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```
