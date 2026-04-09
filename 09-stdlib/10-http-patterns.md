# HTTP и REST API паттерны

## REST принципы

REST (Representational State Transfer) -- архитектурный стиль для построения API. Основная идея: API оперирует **ресурсами** (существительные, не глаголы), а HTTP методы определяют действия.

### Ресурсы и URL

```
GET    /users          — список пользователей
GET    /users/42       — конкретный пользователь
POST   /users          — создать пользователя
PUT    /users/42       — полностью заменить пользователя
PATCH  /users/42       — частично обновить пользователя
DELETE /users/42       — удалить пользователя

GET    /users/42/posts — посты конкретного пользователя (вложенный ресурс)
```

### HTTP методы

| Метод   | Действие        | Тело запроса | Тело ответа | Идемпотентный | Безопасный |
|---------|-----------------|:------------:|:-----------:|:-------------:|:----------:|
| GET     | Получить ресурс | Нет          | Да          | Да            | Да         |
| HEAD    | Заголовки       | Нет          | Нет         | Да            | Да         |
| POST    | Создать ресурс  | Да           | Да          | **Нет**       | Нет        |
| PUT     | Заменить ресурс | Да           | Да          | Да            | Нет        |
| PATCH   | Обновить часть  | Да           | Да          | **Нет**       | Нет        |
| DELETE  | Удалить ресурс  | Нет          | Опционально | Да            | Нет        |
| OPTIONS | Доступные методы| Нет          | Да          | Да            | Да         |

**Идемпотентность** -- повторный вызов с теми же параметрами даёт тот же результат. PUT идемпотентен: повторная отправка `PUT /users/42 {name: "Alice"}` не меняет состояние. POST -- нет: повторный `POST /users` создаст дубликат.

**Безопасный метод** -- не изменяет состояние сервера (GET, HEAD, OPTIONS).

### Статус-коды

```
2xx — успех
  200 OK              — GET, PUT, PATCH (с телом ответа)
  201 Created         — POST (ресурс создан), Location header
  204 No Content      — DELETE (тело не нужно)

3xx — перенаправления
  301 Moved Permanently
  304 Not Modified    — кэш актуален

4xx — ошибка клиента
  400 Bad Request     — невалидный JSON, ошибка валидации
  401 Unauthorized    — нет/невалидный токен авторизации
  403 Forbidden       — авторизован, но нет доступа
  404 Not Found       — ресурс не найден
  405 Method Not Allowed
  409 Conflict        — конфликт (дупликат email и т.д.)
  422 Unprocessable Entity — валидация бизнес-логики
  429 Too Many Requests — rate limit

5xx — ошибка сервера
  500 Internal Server Error
  502 Bad Gateway
  503 Service Unavailable
  504 Gateway Timeout
```

---

## Routing (http.ServeMux Go 1.22+)

Начиная с Go 1.22, стандартный `http.ServeMux` поддерживает method matching и path parameters. Раньше для этого нужен был `chi` или `gorilla/mux`.

### Базовый роутинг

```go
func main() {
    mux := http.NewServeMux()

    // Method matching — new in Go 1.22
    mux.HandleFunc("GET /users", listUsers)
    mux.HandleFunc("POST /users", createUser)
    mux.HandleFunc("GET /users/{id}", getUser)
    mux.HandleFunc("PUT /users/{id}", updateUser)
    mux.HandleFunc("DELETE /users/{id}", deleteUser)

    // Wildcard — matches everything under /static/
    mux.Handle("GET /static/", http.StripPrefix("/static/",
        http.FileServer(http.Dir("./public"))))

    // Exact match with {$} — only "/"
    mux.HandleFunc("GET /{$}", handleRoot)

    log.Fatal(http.ListenAndServe(":8080", mux))
}
```

### Path parameters

```go
func getUser(w http.ResponseWriter, r *http.Request) {
    // PathValue extracts named parameter from pattern
    id := r.PathValue("id")
    if id == "" {
        http.Error(w, "missing id", http.StatusBadRequest)
        return
    }

    userID, err := strconv.Atoi(id)
    if err != nil {
        http.Error(w, "invalid id", http.StatusBadRequest)
        return
    }

    // ... fetch user by userID
    _ = userID
}
```

### Вложенные паттерны

```go
func main() {
    mux := http.NewServeMux()

    // Nested resources
    mux.HandleFunc("GET /users/{userID}/posts", listUserPosts)
    mux.HandleFunc("GET /users/{userID}/posts/{postID}", getUserPost)

    // Catch-all wildcard
    mux.HandleFunc("GET /files/{path...}", serveFile)
}

func serveFile(w http.ResponseWriter, r *http.Request) {
    // {path...} captures the rest of the URL
    filePath := r.PathValue("path") // e.g. "docs/readme.txt"
    fmt.Fprintf(w, "serving: %s", filePath)
}
```

### Сравнение с chi и gorilla/mux

| Возможность            | http.ServeMux (1.22+) | chi         | gorilla/mux       |
|------------------------|-----------------------|-------------|-------------------|
| Method matching        | `GET /path`           | `r.Get()`   | `r.Methods("GET")`|
| Path params            | `{id}`                | `{id}`      | `{id}`            |
| Regex в params         | Нет                   | `{id:[0-9]+}` | `{id:[0-9]+}`  |
| Route groups           | Нет (вручную)         | `r.Group()` | `r.Subrouter()`   |
| Встроенный middleware  | Нет                   | Да          | Нет               |
| Stdlib совместимость   | Да (это stdlib)       | Да          | Да                |

Рекомендация: для большинства проектов `http.ServeMux` 1.22+ достаточно. chi полезен когда нужны route groups и встроенные middleware.

---

## Middleware

Middleware -- функция, которая оборачивает `http.Handler`, добавляя логику до и/или после обработки запроса. Стандартный паттерн:

```go
// Standard middleware signature
func middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // before: pre-processing
        next.ServeHTTP(w, r) // call the next handler
        // after: post-processing
    })
}
```

### Цепочка middleware

```go
// Chain applies middlewares in order: first middleware wraps outermost
func Chain(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
    // Apply in reverse so first middleware in the list runs first
    for i := len(middlewares) - 1; i >= 0; i-- {
        handler = middlewares[i](handler)
    }
    return handler
}

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("GET /users", listUsers)

    // Request flow: RequestID -> Logging -> Recovery -> mux
    handler := Chain(mux,
        RequestIDMiddleware,
        LoggingMiddleware,
        RecoveryMiddleware,
    )

    http.ListenAndServe(":8080", handler)
}
```

### Logging middleware

```go
// responseWriter wrapper to capture status code
type wrappedWriter struct {
    http.ResponseWriter
    statusCode int
    written    bool
}

func (w *wrappedWriter) WriteHeader(code int) {
    if !w.written {
        w.statusCode = code
        w.written = true
    }
    w.ResponseWriter.WriteHeader(code)
}

func LoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        wrapped := &wrappedWriter{ResponseWriter: w, statusCode: http.StatusOK}

        next.ServeHTTP(wrapped, r)

        slog.Info("request",
            "method", r.Method,
            "path", r.URL.Path,
            "status", wrapped.statusCode,
            "duration", time.Since(start),
        )
    })
}
```

### Recovery middleware

```go
func RecoveryMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                // Log stack trace for debugging
                slog.Error("panic recovered",
                    "error", err,
                    "stack", string(debug.Stack()),
                )
                http.Error(w, "Internal Server Error", http.StatusInternalServerError)
            }
        }()
        next.ServeHTTP(w, r)
    })
}
```

### Auth middleware

```go
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        if token == "" {
            writeError(w, http.StatusUnauthorized, "missing authorization header")
            return
        }

        // Strip "Bearer " prefix
        token = strings.TrimPrefix(token, "Bearer ")

        userID, err := validateToken(token)
        if err != nil {
            writeError(w, http.StatusUnauthorized, "invalid token")
            return
        }

        // Store user ID in context for downstream handlers
        ctx := context.WithValue(r.Context(), userIDKey, userID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Type-safe context key
type contextKey string

const userIDKey contextKey = "userID"

func getUserID(ctx context.Context) (string, bool) {
    id, ok := ctx.Value(userIDKey).(string)
    return id, ok
}
```

### Request ID middleware

```go
func RequestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID := r.Header.Get("X-Request-ID")
        if requestID == "" {
            requestID = uuid.NewString() // or use crypto/rand
        }

        // Add to response headers
        w.Header().Set("X-Request-ID", requestID)

        // Add to context
        ctx := context.WithValue(r.Context(), requestIDKey, requestID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### CORS middleware

```go
func CORSMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        w.Header().Set("Access-Control-Max-Age", "86400")

        // Handle preflight
        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusNoContent)
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

### Rate limiting middleware

```go
func RateLimitMiddleware(rps int) func(http.Handler) http.Handler {
    limiter := rate.NewLimiter(rate.Limit(rps), rps)

    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if !limiter.Allow() {
                w.Header().Set("Retry-After", "1")
                writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

Порядок middleware важен. Типичный порядок:
1. Recovery (ловит panic из всех нижележащих)
2. Request ID (нужен для логирования)
3. Logging (логирует все запросы)
4. CORS (обрабатывает preflight до auth)
5. Rate Limiting
6. Auth (последний перед бизнес-логикой)

---

## Обработка запросов

### Парсинг JSON body

```go
type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Age   int    `json:"age"`
}

func createUser(w http.ResponseWriter, r *http.Request) {
    // Limit body size to prevent abuse
    r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB

    var req CreateUserRequest

    dec := json.NewDecoder(r.Body)
    dec.DisallowUnknownFields() // strict parsing

    if err := dec.Decode(&req); err != nil {
        var maxBytesErr *http.MaxBytesError
        switch {
        case errors.As(err, &maxBytesErr):
            writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
        case errors.Is(err, io.EOF):
            writeError(w, http.StatusBadRequest, "request body is empty")
        default:
            writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
        }
        return
    }

    // Validate
    if err := validateCreateUser(req); err != nil {
        writeError(w, http.StatusUnprocessableEntity, err.Error())
        return
    }

    // ... create user
}
```

### Query parameters

```go
func listUsers(w http.ResponseWriter, r *http.Request) {
    q := r.URL.Query()

    // Pagination
    page, _ := strconv.Atoi(q.Get("page"))
    if page < 1 {
        page = 1
    }
    perPage, _ := strconv.Atoi(q.Get("per_page"))
    if perPage < 1 || perPage > 100 {
        perPage = 20
    }

    // Filtering
    nameFilter := q.Get("name")

    // Sorting
    sortBy := q.Get("sort_by")
    if sortBy == "" {
        sortBy = "created_at"
    }
    sortOrder := q.Get("sort_order")
    if sortOrder != "asc" {
        sortOrder = "desc"
    }

    // ... query with these parameters
}
```

### Валидация

```go
type ValidationError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
}

func validateCreateUser(req CreateUserRequest) error {
    var errs []ValidationError

    if strings.TrimSpace(req.Name) == "" {
        errs = append(errs, ValidationError{Field: "name", Message: "is required"})
    }
    if !isValidEmail(req.Email) {
        errs = append(errs, ValidationError{Field: "email", Message: "is invalid"})
    }
    if req.Age < 0 || req.Age > 150 {
        errs = append(errs, ValidationError{Field: "age", Message: "must be between 0 and 150"})
    }

    if len(errs) > 0 {
        // Return structured validation errors
        data, _ := json.Marshal(errs)
        return fmt.Errorf("validation failed: %s", data)
    }
    return nil
}
```

### Универсальный decode хелпер

```go
// decode reads JSON body into dst with size limit and strict parsing
func decode[T any](r *http.Request, maxBytes int64) (T, error) {
    var dst T

    r.Body = http.MaxBytesReader(nil, r.Body, maxBytes)

    dec := json.NewDecoder(r.Body)
    dec.DisallowUnknownFields()

    if err := dec.Decode(&dst); err != nil {
        return dst, fmt.Errorf("decode body: %w", err)
    }

    // Check for extra JSON values after first object
    if dec.More() {
        return dst, errors.New("body must contain a single JSON value")
    }

    return dst, nil
}

// Usage
func createUser(w http.ResponseWriter, r *http.Request) {
    req, err := decode[CreateUserRequest](r, 1<<20)
    if err != nil {
        writeError(w, http.StatusBadRequest, err.Error())
        return
    }
    // ... use req
}
```

---

## Паттерны ответов

### JSON response хелперы

```go
// writeJSON writes a JSON response with status code
func writeJSON(w http.ResponseWriter, status int, data any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)

    if err := json.NewEncoder(w).Encode(data); err != nil {
        // Cannot write error response — headers already sent
        slog.Error("failed to encode response", "error", err)
    }
}

// writeError writes a structured error response
func writeError(w http.ResponseWriter, status int, message string) {
    writeJSON(w, status, ErrorResponse{
        Error: APIError{
            Code:    status,
            Message: message,
        },
    })
}
```

### Структурированные ошибки

```go
type ErrorResponse struct {
    Error APIError `json:"error"`
}

type APIError struct {
    Code    int               `json:"code"`
    Message string            `json:"message"`
    Details []ValidationError `json:"details,omitempty"`
}

// Usage in handlers
func getUser(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")

    user, err := store.FindUser(r.Context(), id)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            writeError(w, http.StatusNotFound, "user not found")
            return
        }
        slog.Error("find user", "error", err, "id", id)
        writeError(w, http.StatusInternalServerError, "internal error")
        return
    }

    writeJSON(w, http.StatusOK, user)
}

func createUser(w http.ResponseWriter, r *http.Request) {
    // ... decode and validate

    user, err := store.CreateUser(r.Context(), req)
    if err != nil {
        if errors.Is(err, ErrDuplicateEmail) {
            writeError(w, http.StatusConflict, "email already exists")
            return
        }
        writeError(w, http.StatusInternalServerError, "internal error")
        return
    }

    writeJSON(w, http.StatusCreated, user)
}
```

### Пагинация

**Offset-based** -- простая, но медленная на больших offset:

```go
type PageResponse[T any] struct {
    Data       []T  `json:"data"`
    Page       int  `json:"page"`
    PerPage    int  `json:"per_page"`
    TotalItems int  `json:"total_items"`
    TotalPages int  `json:"total_pages"`
}

func listUsers(w http.ResponseWriter, r *http.Request) {
    page, perPage := parsePagination(r)
    offset := (page - 1) * perPage

    users, total, err := store.ListUsers(r.Context(), offset, perPage)
    if err != nil {
        writeError(w, http.StatusInternalServerError, "internal error")
        return
    }

    writeJSON(w, http.StatusOK, PageResponse[User]{
        Data:       users,
        Page:       page,
        PerPage:    perPage,
        TotalItems: total,
        TotalPages: (total + perPage - 1) / perPage,
    })
}
```

**Cursor-based** -- стабильная при изменениях данных, быстрее на больших объёмах:

```go
type CursorResponse[T any] struct {
    Data       []T    `json:"data"`
    NextCursor string `json:"next_cursor,omitempty"`
    HasMore    bool   `json:"has_more"`
}

func listUsers(w http.ResponseWriter, r *http.Request) {
    cursor := r.URL.Query().Get("cursor")
    limit := 20

    users, nextCursor, err := store.ListUsersAfter(r.Context(), cursor, limit+1)
    if err != nil {
        writeError(w, http.StatusInternalServerError, "internal error")
        return
    }

    hasMore := len(users) > limit
    if hasMore {
        users = users[:limit] // trim extra item
    }

    writeJSON(w, http.StatusOK, CursorResponse[User]{
        Data:       users,
        NextCursor: nextCursor,
        HasMore:    hasMore,
    })
}
```

| Критерий            | Offset          | Cursor              |
|---------------------|-----------------|---------------------|
| Произвольная стр.   | Да              | Нет (только вперёд) |
| Скорость на стр.1000| `O(offset+limit)`| `O(limit)`         |
| Стабильность        | Пропуски/дубли при INSERT/DELETE | Стабильно |
| Сложность           | Простая         | Нужен уникальный курсор |

### API versioning

**URL-based** (проще, нагляднее):
```go
mux.HandleFunc("GET /v1/users", listUsersV1)
mux.HandleFunc("GET /v2/users", listUsersV2) // new fields, different format
```

**Header-based** (чище URL, сложнее в реализации):
```go
func listUsers(w http.ResponseWriter, r *http.Request) {
    version := r.Header.Get("API-Version")
    switch version {
    case "2", "2024-01-15":
        listUsersV2(w, r)
    default:
        listUsersV1(w, r)
    }
}
```

На практике URL versioning (`/v1/`, `/v2/`) используется чаще -- проще тестировать через curl и документировать.

---

## Graceful Shutdown

Корректное завершение HTTP сервера: перестаём принимать новые соединения, дожидаемся завершения текущих запросов.

```go
func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("GET /health", healthCheck)
    // ... register routes

    srv := &http.Server{
        Addr:         ":8080",
        Handler:      mux,
        ReadTimeout:  5 * time.Second,
        WriteTimeout: 10 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    // Start server in goroutine
    go func() {
        slog.Info("server starting", "addr", srv.Addr)
        if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            slog.Error("server error", "error", err)
            os.Exit(1)
        }
    }()

    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    sig := <-quit
    slog.Info("shutting down", "signal", sig)

    // Give active connections time to finish
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        slog.Error("forced shutdown", "error", err)
        os.Exit(1)
    }

    slog.Info("server stopped gracefully")
}
```

**Важные моменты:**
- `ListenAndServe` после `Shutdown` возвращает `http.ErrServerClosed` -- это нормально
- `Shutdown` не прерывает активные соединения, а ждёт их завершения
- Timeout в `context.WithTimeout` -- страховка от зависших соединений
- `signal.Notify` с буферизованным каналом (размер 1) -- иначе сигнал может потеряться

### Порядок завершения с зависимостями

```go
func main() {
    // Initialize dependencies
    db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
    if err != nil {
        log.Fatal(err)
    }

    srv := &http.Server{
        Addr:    ":8080",
        Handler: newRouter(db),
    }

    go func() {
        if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            log.Fatal(err)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    // 1. Stop accepting new requests
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        slog.Error("http shutdown", "error", err)
    }

    // 2. Close dependencies AFTER server is stopped
    if err := db.Close(); err != nil {
        slog.Error("db close", "error", err)
    }

    slog.Info("clean shutdown complete")
}
```

Порядок завершения: сначала HTTP сервер (дожидаемся текущих запросов), потом зависимости (БД, кэш и т.д.).

---

## Тестирование HTTP handlers

### httptest.NewRecorder -- unit-тесты handler-ов

```go
func TestGetUser(t *testing.T) {
    // Create request with path parameter
    req := httptest.NewRequest("GET", "/users/42", nil)
    req.SetPathValue("id", "42") // Go 1.22+

    rec := httptest.NewRecorder()

    getUser(rec, req)

    // Check status
    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rec.Code)
    }

    // Check content type
    ct := rec.Header().Get("Content-Type")
    if ct != "application/json" {
        t.Fatalf("expected application/json, got %s", ct)
    }

    // Decode response body
    var user User
    if err := json.NewDecoder(rec.Body).Decode(&user); err != nil {
        t.Fatalf("decode response: %v", err)
    }

    if user.ID != 42 {
        t.Errorf("expected user ID 42, got %d", user.ID)
    }
}
```

### Table-driven handler tests

```go
func TestCreateUser(t *testing.T) {
    tests := []struct {
        name       string
        body       string
        wantStatus int
        wantErr    string
    }{
        {
            name:       "valid request",
            body:       `{"name":"Alice","email":"alice@example.com","age":30}`,
            wantStatus: http.StatusCreated,
        },
        {
            name:       "empty body",
            body:       "",
            wantStatus: http.StatusBadRequest,
            wantErr:    "request body is empty",
        },
        {
            name:       "missing name",
            body:       `{"email":"alice@example.com"}`,
            wantStatus: http.StatusUnprocessableEntity,
            wantErr:    "name",
        },
        {
            name:       "invalid json",
            body:       `{invalid}`,
            wantStatus: http.StatusBadRequest,
            wantErr:    "invalid JSON",
        },
        {
            name:       "unknown field",
            body:       `{"name":"Alice","email":"a@b.com","age":30,"extra":"field"}`,
            wantStatus: http.StatusBadRequest,
            wantErr:    "unknown field",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var body io.Reader
            if tt.body != "" {
                body = strings.NewReader(tt.body)
            }

            req := httptest.NewRequest("POST", "/users", body)
            req.Header.Set("Content-Type", "application/json")
            rec := httptest.NewRecorder()

            createUser(rec, req)

            if rec.Code != tt.wantStatus {
                t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
            }

            if tt.wantErr != "" {
                respBody := rec.Body.String()
                if !strings.Contains(respBody, tt.wantErr) {
                    t.Errorf("body = %s, want to contain %q", respBody, tt.wantErr)
                }
            }
        })
    }
}
```

### httptest.NewServer -- интеграционные тесты

```go
func TestAPI(t *testing.T) {
    // Set up router with all dependencies
    mux := http.NewServeMux()
    mux.HandleFunc("GET /users/{id}", getUser)
    mux.HandleFunc("POST /users", createUser)

    // Start test server
    ts := httptest.NewServer(mux)
    defer ts.Close()

    client := ts.Client()

    // Test: create user
    body := strings.NewReader(`{"name":"Alice","email":"alice@test.com","age":25}`)
    resp, err := client.Post(ts.URL+"/users", "application/json", body)
    if err != nil {
        t.Fatalf("POST /users: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusCreated {
        t.Fatalf("expected 201, got %d", resp.StatusCode)
    }

    // Test: get user
    resp, err = client.Get(ts.URL + "/users/1")
    if err != nil {
        t.Fatalf("GET /users/1: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        t.Fatalf("expected 200, got %d", resp.StatusCode)
    }
}
```

### Тестирование middleware

```go
func TestLoggingMiddleware(t *testing.T) {
    // Create a handler that returns 200
    inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("ok"))
    })

    // Wrap with middleware
    handler := LoggingMiddleware(inner)

    req := httptest.NewRequest("GET", "/test", nil)
    rec := httptest.NewRecorder()

    handler.ServeHTTP(rec, req)

    if rec.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", rec.Code)
    }
}

func TestAuthMiddleware_NoToken(t *testing.T) {
    inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        t.Error("handler should not be called without auth")
    })

    handler := AuthMiddleware(inner)

    req := httptest.NewRequest("GET", "/protected", nil)
    // No Authorization header
    rec := httptest.NewRecorder()

    handler.ServeHTTP(rec, req)

    if rec.Code != http.StatusUnauthorized {
        t.Errorf("expected 401, got %d", rec.Code)
    }
}
```

---

## Полный пример: минимальный REST API

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "log/slog"
    "net/http"
    "os"
    "os/signal"
    "strconv"
    "strings"
    "sync"
    "syscall"
    "time"
)

// --- Domain ---

type User struct {
    ID        int       `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"created_at"`
}

// --- In-memory store ---

type Store struct {
    mu    sync.RWMutex
    users map[int]User
    nextID int
}

func NewStore() *Store {
    return &Store{users: make(map[int]User), nextID: 1}
}

func (s *Store) Create(name, email string) User {
    s.mu.Lock()
    defer s.mu.Unlock()
    u := User{ID: s.nextID, Name: name, Email: email, CreatedAt: time.Now()}
    s.users[s.nextID] = u
    s.nextID++
    return u
}

func (s *Store) Get(id int) (User, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    u, ok := s.users[id]
    return u, ok
}

func (s *Store) List() []User {
    s.mu.RLock()
    defer s.mu.RUnlock()
    result := make([]User, 0, len(s.users))
    for _, u := range s.users {
        result = append(result, u)
    }
    return result
}

func (s *Store) Delete(id int) bool {
    s.mu.Lock()
    defer s.mu.Unlock()
    if _, ok := s.users[id]; !ok {
        return false
    }
    delete(s.users, id)
    return true
}

// --- Handlers ---

type API struct {
    store *Store
}

func (a *API) listUsers(w http.ResponseWriter, r *http.Request) {
    users := a.store.List()
    writeJSON(w, http.StatusOK, users)
}

func (a *API) getUser(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.Atoi(r.PathValue("id"))
    if err != nil {
        writeError(w, http.StatusBadRequest, "invalid user id")
        return
    }

    user, ok := a.store.Get(id)
    if !ok {
        writeError(w, http.StatusNotFound, "user not found")
        return
    }

    writeJSON(w, http.StatusOK, user)
}

func (a *API) createUser(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Name  string `json:"name"`
        Email string `json:"email"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid JSON")
        return
    }

    if strings.TrimSpace(req.Name) == "" {
        writeError(w, http.StatusUnprocessableEntity, "name is required")
        return
    }

    user := a.store.Create(req.Name, req.Email)
    writeJSON(w, http.StatusCreated, user)
}

func (a *API) deleteUser(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.Atoi(r.PathValue("id"))
    if err != nil {
        writeError(w, http.StatusBadRequest, "invalid user id")
        return
    }

    if !a.store.Delete(id) {
        writeError(w, http.StatusNotFound, "user not found")
        return
    }

    w.WriteHeader(http.StatusNoContent)
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, data any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(data)
}

type errorResp struct {
    Error struct {
        Code    int    `json:"code"`
        Message string `json:"message"`
    } `json:"error"`
}

func writeError(w http.ResponseWriter, status int, msg string) {
    resp := errorResp{}
    resp.Error.Code = status
    resp.Error.Message = msg
    writeJSON(w, status, resp)
}

// --- Main ---

func main() {
    store := NewStore()
    api := &API{store: store}

    mux := http.NewServeMux()
    mux.HandleFunc("GET /users", api.listUsers)
    mux.HandleFunc("GET /users/{id}", api.getUser)
    mux.HandleFunc("POST /users", api.createUser)
    mux.HandleFunc("DELETE /users/{id}", api.deleteUser)

    srv := &http.Server{
        Addr:         ":8080",
        Handler:      mux,
        ReadTimeout:  5 * time.Second,
        WriteTimeout: 10 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    go func() {
        fmt.Println("listening on :8080")
        if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            slog.Error("server error", "error", err)
            os.Exit(1)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    srv.Shutdown(ctx)
}
```

---

## Частые вопросы на собеседовании

### В чём разница между PUT и PATCH?

**PUT** -- полная замена ресурса. Клиент отправляет **все** поля. Если поле пропущено, оно обнуляется.

**PATCH** -- частичное обновление. Клиент отправляет только изменённые поля. Пропущенные поля не трогаются.

```go
// PUT /users/1 — replaces entirely
// Sending {"name": "Bob"} without email → email becomes ""

// PATCH /users/1 — partial update
// Sending {"name": "Bob"} → only name changes, email stays
```

PUT идемпотентен (повторный запрос с тем же телом даёт тот же результат). PATCH формально не гарантирует идемпотентность (зависит от реализации).

### Как реализовать PATCH в Go?

```go
type UpdateUserRequest struct {
    Name  *string `json:"name"`  // pointer = field is optional
    Email *string `json:"email"`
    Age   *int    `json:"age"`
}

func (a *API) patchUser(w http.ResponseWriter, r *http.Request) {
    id, _ := strconv.Atoi(r.PathValue("id"))

    var req UpdateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid JSON")
        return
    }

    user, ok := a.store.Get(id)
    if !ok {
        writeError(w, http.StatusNotFound, "user not found")
        return
    }

    // Only update fields that were sent (non-nil pointers)
    if req.Name != nil {
        user.Name = *req.Name
    }
    if req.Email != nil {
        user.Email = *req.Email
    }

    // ... save user
    writeJSON(w, http.StatusOK, user)
}
```

Указатели (`*string`) позволяют отличить "поле не отправлено" (nil) от "поле отправлено пустым" ("").

### Какой статус-код вернуть?

| Ситуация                    | Статус-код              |
|-----------------------------|-------------------------|
| GET успешный                | 200 OK                  |
| POST создал ресурс          | 201 Created             |
| DELETE успешный             | 204 No Content          |
| Невалидный JSON             | 400 Bad Request         |
| Нет токена авторизации      | 401 Unauthorized        |
| Токен есть, нет прав        | 403 Forbidden           |
| Ресурс не найден            | 404 Not Found           |
| Дупликат (email уже есть)   | 409 Conflict            |
| Ошибка валидации бизнес-правил | 422 Unprocessable Entity |
| Слишком много запросов      | 429 Too Many Requests   |
| Баг на сервере              | 500 Internal Server Error|

### Влияет ли порядок middleware?

Да. Middleware выполняются в порядке оборачивания -- первый middleware в цепочке обрабатывает запрос первым и ответ последним.

```
Request:  Client → Recovery → Logger → Auth → Handler
Response: Client ← Recovery ← Logger ← Auth ← Handler
```

Recovery должен быть внешним (ловит panic от всех остальных). Auth -- внутренним (ближе к бизнес-логике). Logger -- между ними (логирует все запросы, включая 401).

### Почему http.ServeMux 1.22+ достаточно для большинства проектов?

До Go 1.22 `http.ServeMux` не поддерживал method matching и path parameters -- нужен был `chi` или `gorilla/mux`. Теперь `http.ServeMux` покрывает:
- Маршрутизацию по методу: `"GET /users/{id}"`
- Path parameters: `r.PathValue("id")`
- Wildcard: `{path...}`
- Exact match: `{$}`

Внешний роутер по-прежнему нужен для: regex в path params, route groups, встроенных middleware chains.

### Что такое idempotency key?

Для POST запросов (которые не идемпотентны) можно реализовать идемпотентность через специальный заголовок:

```go
func (a *API) createPayment(w http.ResponseWriter, r *http.Request) {
    idempotencyKey := r.Header.Get("Idempotency-Key")
    if idempotencyKey == "" {
        writeError(w, http.StatusBadRequest, "Idempotency-Key header is required")
        return
    }

    // Check if we already processed this key
    if result, ok := a.store.GetByIdempotencyKey(idempotencyKey); ok {
        writeJSON(w, http.StatusOK, result) // return cached result
        return
    }

    // Process payment and save result with idempotency key
    // ...
}
```

Это критически важно для платёжных API: повторный запрос с тем же ключом не создаёт дубликат транзакции.
