# Authentication

## Обзор методов аутентификации

```
Метод              Где хранится состояние   Масштабируемость   Типичное применение
─────────────────  ──────────────────────   ────────────────   ───────────────────
JWT                Клиент (token)           Отличная           SPA, мобилки, микросервисы
Session + Cookie   Сервер (Redis/DB)        Средняя            Классические web-apps
OAuth2 / OIDC      Провайдер (Google, GH)   Отличная           "Login with..." кнопки
API Key            Сервер (DB)              Отличная           Service-to-service, 3rd party
Basic Auth         Нигде (каждый запрос)    Плохая             Внутренние тулы, dev
```

---

## Password Hashing

Золотое правило: **никогда не хранить пароли в открытом виде**. Даже если БД утечёт,
хеши не позволят восстановить пароли (при правильном алгоритме).

### bcrypt

Самый распространённый выбор. Встроенный salt, настраиваемый cost factor.

```go
import "golang.org/x/crypto/bcrypt"

// Hash password with bcrypt (cost = 10 by default)
func HashPassword(password string) (string, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    return string(hash), err
}

// Compare hash with plaintext password
func CheckPassword(hash, password string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

**Cost factor** определяет количество итераций: `2^cost`. Рекомендации:

```
Cost 10 (default) — ~100ms  — минимум для production
Cost 12           — ~300ms  — хороший баланс
Cost 14           — ~1s     — высокая безопасность

Увеличивать cost со временем: железо дешевеет, атаки быстреют.
Можно ре-хешить при логине если cost устарел.
```

### Argon2id (рекомендация для новых проектов)

Memory-hard алгоритм: защищает от GPU/ASIC атак, потому что требует много памяти.

```go
import (
    "crypto/rand"
    "encoding/base64"
    "fmt"

    "golang.org/x/crypto/argon2"
)

type Argon2Params struct {
    Memory      uint32 // KiB of memory
    Iterations  uint32 // number of passes
    Parallelism uint8  // number of threads
    SaltLength  uint32 // bytes
    KeyLength   uint32 // bytes
}

// OWASP recommended defaults
var DefaultParams = Argon2Params{
    Memory:      64 * 1024, // 64 MB
    Iterations:  1,
    Parallelism: 4,
    SaltLength:  16,
    KeyLength:   32,
}

func HashPasswordArgon2(password string) (string, error) {
    salt := make([]byte, DefaultParams.SaltLength)
    if _, err := rand.Read(salt); err != nil {
        return "", err
    }

    hash := argon2.IDKey(
        []byte(password), salt,
        DefaultParams.Iterations,
        DefaultParams.Memory,
        DefaultParams.Parallelism,
        DefaultParams.KeyLength,
    )

    // Encode as: $argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>
    return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
        argon2.Version,
        DefaultParams.Memory,
        DefaultParams.Iterations,
        DefaultParams.Parallelism,
        base64.RawStdEncoding.EncodeToString(salt),
        base64.RawStdEncoding.EncodeToString(hash),
    ), nil
}
```

```
bcrypt vs Argon2id:

bcrypt   — проверен временем, широко поддерживается, ограничен 72 байтами пароля
Argon2id — memory-hard (защита от GPU), настраиваемый, стандарт с 2015

Для новых проектов: Argon2id
Для существующих: bcrypt нормально, можно мигрировать при логине
```

---

## JWT (JSON Web Token)

### Структура

```
JWT = header.payload.signature (base64url encoded)

Header:  {"alg": "HS256", "typ": "JWT"}
Payload: {"sub": "user123", "exp": 1234567890, "role": "admin"}
Signature: HMAC-SHA256(base64(header) + "." + base64(payload), secret)

Access Token:  короткоживущий (5-15 мин), для доступа к API
Refresh Token: долгоживущий (7-30 дней), для обновления access token
```

Payload видим всем (base64 это не шифрование!). Никаких паролей, персональных данных.

### Генерация и валидация

```go
import "github.com/golang-jwt/jwt/v5"

// Custom claims embedded in token payload
type Claims struct {
    UserID string `json:"user_id"`
    Role   string `json:"role"`
    jwt.RegisteredClaims
}

// GenerateTokens creates an access/refresh token pair
func GenerateTokens(userID, role, secret string) (accessToken, refreshToken string, err error) {
    // Access token (15 min)
    accessClaims := Claims{
        UserID: userID,
        Role:   role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            Issuer:    "my-service",
            ID:        uuid.NewString(), // jti — for revocation tracking
        },
    }
    access := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
    accessToken, err = access.SignedString([]byte(secret))
    if err != nil {
        return "", "", err
    }

    // Refresh token (7 days)
    refreshClaims := jwt.RegisteredClaims{
        ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
        IssuedAt:  jwt.NewNumericDate(time.Now()),
        Subject:   userID,
        ID:        uuid.NewString(),
    }
    refresh := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
    refreshToken, err = refresh.SignedString([]byte(secret))

    return accessToken, refreshToken, err
}

// ValidateToken parses and validates JWT, returning claims
func ValidateToken(tokenString, secret string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{},
        func(token *jwt.Token) (any, error) {
            // Guard against algorithm switching attack
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
            }
            return []byte(secret), nil
        })
    if err != nil {
        return nil, err
    }

    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return nil, errors.New("invalid token")
    }

    return claims, nil
}
```

### Auth Middleware

```go
// Extract JWT from Authorization header and inject claims into context
func authMiddleware(secret string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            authHeader := r.Header.Get("Authorization")
            if !strings.HasPrefix(authHeader, "Bearer ") {
                http.Error(w, "unauthorized", http.StatusUnauthorized)
                return
            }

            tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
            claims, err := ValidateToken(tokenStr, secret)
            if err != nil {
                http.Error(w, "invalid token", http.StatusUnauthorized)
                return
            }

            ctx := context.WithValue(r.Context(), userClaimsKey, claims)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

### RS256 — ассиметричная подпись для микросервисов

Когда несколько сервисов должны проверять токены, но только один выпускает:

```go
import "crypto/rsa"

// Auth service signs with PRIVATE key
func GenerateTokenRS256(claims Claims, privateKey *rsa.PrivateKey) (string, error) {
    token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
    return token.SignedString(privateKey)
}

// Any service validates with PUBLIC key (no secret sharing)
func ValidateTokenRS256(tokenStr string, publicKey *rsa.PublicKey) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenStr, &Claims{},
        func(t *jwt.Token) (any, error) {
            if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
                return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
            }
            return publicKey, nil
        })
    if err != nil {
        return nil, err
    }
    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return nil, errors.New("invalid token")
    }
    return claims, nil
}
```

```
Алгоритмы подписи:

HS256 (HMAC)  — симметричный, один secret для sign + verify
                Подходит для одного сервиса
RS256 (RSA)   — ассиметричный, private key → sign, public key → verify
                Подходит для микросервисов (public key раздаётся через JWKS)
ES256 (ECDSA) — ассиметричный, компактнее RSA, быстрее верификация
                Хороший выбор для мобильных клиентов
```

### Token Revocation

JWT stateless по природе, поэтому отозвать конкретный токен непросто:

```go
// Strategy 1: Blacklist in Redis (by jti claim)
type TokenBlacklist struct {
    rdb *redis.Client
}

// Revoke token — store jti with TTL matching token expiry
func (b *TokenBlacklist) Revoke(ctx context.Context, jti string, expiry time.Time) error {
    ttl := time.Until(expiry)
    if ttl <= 0 {
        return nil // already expired
    }
    return b.rdb.Set(ctx, "blacklist:"+jti, "1", ttl).Err()
}

// IsRevoked checks if token has been revoked
func (b *TokenBlacklist) IsRevoked(ctx context.Context, jti string) (bool, error) {
    exists, err := b.rdb.Exists(ctx, "blacklist:"+jti).Result()
    return exists > 0, err
}

// Use in middleware after ValidateToken:
// if revoked, _ := blacklist.IsRevoked(ctx, claims.ID); revoked { ... }
```

```
Strategy 2: Короткий TTL + Refresh Token Rotation
  — Access token живёт 5 мин → даже без blacklist долго не поживёт
  — При refresh выдаём НОВЫЙ refresh token, старый инвалидируем
  — Если кто-то использует старый refresh → сигнал о компрометации → revoke all

Strategy 3: Версионирование (token_version в БД пользователя)
  — В claims: token_version = 5
  — В БД пользователя: token_version = 5
  — При logout: version++ → все старые токены невалидны
  — Минус: нужен запрос к БД на каждый request
```

---

## OAuth2

### Типы потоков (Grant Types)

```
Flow                  Для кого                  Безопасность
────────────────────  ────────────────────────   ────────────
Authorization Code    Web apps (server-side)     Высокая: code обменивается на сервере
Auth Code + PKCE      SPA, mobile, CLI           Высокая: без client_secret, с code_verifier
Client Credentials    Service-to-service (M2M)   Высокая: нет пользователя
Implicit              (DEPRECATED)               Низкая: token в URL fragment
Device Code           Smart TV, IoT              Средняя: пользователь авторизует на другом устройстве
```

### Authorization Code Flow (Web Apps)

```
1. User → App:    нажимает "Login with Google"
2. App → Google:  redirect на /authorize?
     client_id=...&redirect_uri=...&scope=email&response_type=code&state=RANDOM
3. User → Google: логинится, даёт consent
4. Google → App:  redirect на callback?code=AUTH_CODE&state=RANDOM
5. App → Google:  POST /token (server-to-server, с client_secret)
     { code=AUTH_CODE, client_id, client_secret, grant_type=authorization_code }
6. Google → App:  { access_token, refresh_token, id_token }
7. App:           декодирует id_token → email, name → создаёт сессию

state параметр = CSRF protection (сгенерированный random, сохранённый в cookie)
```

### Authorization Code + PKCE (SPA / Mobile)

```
PKCE (Proof Key for Code Exchange) — для клиентов, где нельзя хранить client_secret.

1. App генерирует code_verifier (random 43-128 символов)
2. code_challenge = BASE64URL(SHA256(code_verifier))
3. Redirect на /authorize?...&code_challenge=...&code_challenge_method=S256
4. Получаем code
5. POST /token { code, code_verifier } (без client_secret!)
6. Сервер: SHA256(code_verifier) == code_challenge? → выдаёт токены

Защита: даже если кто-то перехватит code, без code_verifier не обменяет.
```

### Client Credentials Flow (Service-to-Service)

```
Нет пользователя. Сервис аутентифицирует сам себя.

POST /token {
    grant_type=client_credentials,
    client_id=...,
    client_secret=...,
    scope=api.read
}
→ { access_token, expires_in }

Используется: микросервисы между собой, cron jobs, background workers.
```

### Go: OAuth2 с Google

```go
import (
    "golang.org/x/oauth2"
    "golang.org/x/oauth2/google"
)

var oauthConfig = &oauth2.Config{
    ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
    ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
    RedirectURL:  "http://localhost:8080/auth/callback",
    Scopes:       []string{"openid", "email", "profile"},
    Endpoint:     google.Endpoint,
}

// Step 1: Redirect user to Google consent page
func handleLogin(w http.ResponseWriter, r *http.Request) {
    state := generateRandomState() // 32 bytes, base64url encoded

    // Store state in httpOnly cookie for CSRF verification
    http.SetCookie(w, &http.Cookie{
        Name:     "oauth_state",
        Value:    state,
        MaxAge:   300, // 5 min
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteLaxMode,
    })

    url := oauthConfig.AuthCodeURL(state)
    http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// Step 2: Handle callback from Google
func handleCallback(w http.ResponseWriter, r *http.Request) {
    // Verify state matches cookie (CSRF protection)
    cookie, err := r.Cookie("oauth_state")
    if err != nil || cookie.Value != r.URL.Query().Get("state") {
        http.Error(w, "invalid state", http.StatusBadRequest)
        return
    }

    // Exchange authorization code for tokens
    code := r.URL.Query().Get("code")
    token, err := oauthConfig.Exchange(r.Context(), code)
    if err != nil {
        http.Error(w, "token exchange failed", http.StatusBadRequest)
        return
    }

    // Fetch user info using the access token
    client := oauthConfig.Client(r.Context(), token)
    resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
    if err != nil {
        http.Error(w, "failed to get user info", http.StatusInternalServerError)
        return
    }
    defer resp.Body.Close()

    var userInfo struct {
        ID    string `json:"id"`
        Email string `json:"email"`
        Name  string `json:"name"`
    }
    json.NewDecoder(resp.Body).Decode(&userInfo)

    // Find or create user, then issue session/JWT
    user, err := findOrCreateUser(r.Context(), userInfo.Email, userInfo.Name)
    if err != nil {
        http.Error(w, "user creation failed", http.StatusInternalServerError)
        return
    }

    accessToken, _, _ := GenerateTokens(user.ID, user.Role, jwtSecret)
    // Set token in httpOnly cookie or return in response
    http.SetCookie(w, &http.Cookie{
        Name:     "access_token",
        Value:    accessToken,
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteStrictMode,
        Path:     "/",
    })
    http.Redirect(w, r, "/dashboard", http.StatusFound)
}
```

### Go: OAuth2 с GitHub

```go
import "golang.org/x/oauth2/github"

var githubOAuth = &oauth2.Config{
    ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
    ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
    RedirectURL:  "http://localhost:8080/auth/github/callback",
    Scopes:       []string{"user:email", "read:user"},
    Endpoint:     github.Endpoint,
}

// Callback handler — same pattern, different userinfo endpoint
func handleGitHubCallback(w http.ResponseWriter, r *http.Request) {
    // ... verify state, exchange code (same as Google) ...

    client := githubOAuth.Client(r.Context(), token)
    resp, _ := client.Get("https://api.github.com/user")
    defer resp.Body.Close()

    var ghUser struct {
        Login string `json:"login"`
        Email string `json:"email"`
        Name  string `json:"name"`
    }
    json.NewDecoder(resp.Body).Decode(&ghUser)

    // If email is private, fetch from /user/emails endpoint
    if ghUser.Email == "" {
        emailResp, _ := client.Get("https://api.github.com/user/emails")
        defer emailResp.Body.Close()
        var emails []struct {
            Email   string `json:"email"`
            Primary bool   `json:"primary"`
        }
        json.NewDecoder(emailResp.Body).Decode(&emails)
        for _, e := range emails {
            if e.Primary {
                ghUser.Email = e.Email
                break
            }
        }
    }

    // ... create user, issue JWT ...
}
```

### Go: Client Credentials (Service-to-Service)

```go
import "golang.org/x/oauth2/clientcredentials"

// No user involved — service authenticates itself
var cc = clientcredentials.Config{
    ClientID:     os.Getenv("SERVICE_CLIENT_ID"),
    ClientSecret: os.Getenv("SERVICE_CLIENT_SECRET"),
    TokenURL:     "https://auth.example.com/oauth/token",
    Scopes:       []string{"api.read", "api.write"},
}

func callInternalAPI(ctx context.Context) (*http.Response, error) {
    // Token is cached and auto-refreshed
    client := cc.Client(ctx)
    return client.Get("https://internal-api.example.com/data")
}
```

### OAuth2 Refresh Flow

```
Access token истёк → клиент использует refresh token:

POST /token {
    grant_type = refresh_token,
    refresh_token = <refresh_token>,
    client_id = ...
}
→ { access_token (новый), refresh_token (новый!), expires_in }

golang.org/x/oauth2 делает refresh автоматически:
  oauth2.Config.Client() → TokenSource → auto-refresh
```

---

## Session-based Authentication

Классический подход: сервер хранит состояние, клиент хранит только session ID.

### Как работает

```
1. User → POST /login { email, password }
2. Server: проверяет пароль (bcrypt.CompareHashAndPassword)
3. Server: создаёт session { user_id, role, created_at, expires_at }
4. Server: сохраняет session в Redis: SET session:<session_id> <data> EX 86400
5. Server → User: Set-Cookie: session_id=abc123; HttpOnly; Secure; SameSite=Lax
6. User → GET /api/profile (Cookie: session_id=abc123)
7. Server: GET session:abc123 из Redis → есть? → достаём user_id → отвечаем
```

### Реализация с Redis

```go
import "github.com/redis/go-redis/v9"

type SessionStore struct {
    rdb *redis.Client
    ttl time.Duration
}

type Session struct {
    UserID    string    `json:"user_id"`
    Role      string    `json:"role"`
    CreatedAt time.Time `json:"created_at"`
    IP        string    `json:"ip"`
    UserAgent string    `json:"user_agent"`
}

// Create a new session and return its ID
func (s *SessionStore) Create(ctx context.Context, sess Session) (string, error) {
    id := generateSessionID() // crypto/rand, 32 bytes, base64url
    sess.CreatedAt = time.Now()

    data, err := json.Marshal(sess)
    if err != nil {
        return "", err
    }

    // Store in Redis with TTL
    err = s.rdb.Set(ctx, "session:"+id, data, s.ttl).Err()
    if err != nil {
        return "", err
    }

    // Track user's active sessions (for "logout everywhere")
    s.rdb.SAdd(ctx, "user_sessions:"+sess.UserID, id)

    return id, nil
}

// Get session by ID
func (s *SessionStore) Get(ctx context.Context, id string) (*Session, error) {
    data, err := s.rdb.Get(ctx, "session:"+id).Bytes()
    if errors.Is(err, redis.Nil) {
        return nil, ErrSessionNotFound
    }
    if err != nil {
        return nil, err
    }

    var sess Session
    if err := json.Unmarshal(data, &sess); err != nil {
        return nil, err
    }
    return &sess, nil
}

// Destroy a specific session (logout)
func (s *SessionStore) Destroy(ctx context.Context, id string) error {
    sess, err := s.Get(ctx, id)
    if err != nil {
        return err
    }
    pipe := s.rdb.Pipeline()
    pipe.Del(ctx, "session:"+id)
    pipe.SRem(ctx, "user_sessions:"+sess.UserID, id)
    _, err = pipe.Exec(ctx)
    return err
}

// DestroyAll — logout from all devices
func (s *SessionStore) DestroyAll(ctx context.Context, userID string) error {
    sessionIDs, err := s.rdb.SMembers(ctx, "user_sessions:"+userID).Result()
    if err != nil {
        return err
    }
    pipe := s.rdb.Pipeline()
    for _, sid := range sessionIDs {
        pipe.Del(ctx, "session:"+sid)
    }
    pipe.Del(ctx, "user_sessions:"+userID)
    _, err = pipe.Exec(ctx)
    return err
}
```

### Session Middleware

```go
func sessionMiddleware(store *SessionStore) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            cookie, err := r.Cookie("session_id")
            if err != nil {
                http.Error(w, "unauthorized", http.StatusUnauthorized)
                return
            }

            sess, err := store.Get(r.Context(), cookie.Value)
            if err != nil {
                // Clear invalid cookie
                http.SetCookie(w, &http.Cookie{
                    Name:   "session_id",
                    MaxAge: -1,
                })
                http.Error(w, "session expired", http.StatusUnauthorized)
                return
            }

            ctx := context.WithValue(r.Context(), sessionKey, sess)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

### Login / Logout Handlers

```go
func handleLogin(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Email    string `json:"email"`
        Password string `json:"password"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    user, err := userRepo.GetByEmail(r.Context(), req.Email)
    if err != nil {
        // Same error for wrong email AND wrong password (timing attack prevention)
        bcrypt.CompareHashAndPassword([]byte("$2a$10$dummy"), []byte(req.Password))
        http.Error(w, "invalid credentials", http.StatusUnauthorized)
        return
    }

    if !CheckPassword(user.PasswordHash, req.Password) {
        http.Error(w, "invalid credentials", http.StatusUnauthorized)
        return
    }

    sess := Session{
        UserID:    user.ID,
        Role:      user.Role,
        IP:        r.RemoteAddr,
        UserAgent: r.UserAgent(),
    }
    sessionID, err := sessionStore.Create(r.Context(), sess)
    if err != nil {
        http.Error(w, "internal error", http.StatusInternalServerError)
        return
    }

    http.SetCookie(w, &http.Cookie{
        Name:     "session_id",
        Value:    sessionID,
        Path:     "/",
        MaxAge:   86400, // 24 hours
        HttpOnly: true,  // JS cannot access
        Secure:   true,  // HTTPS only
        SameSite: http.SameSiteLaxMode,
    })

    w.WriteHeader(http.StatusOK)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
    cookie, _ := r.Cookie("session_id")
    if cookie != nil {
        sessionStore.Destroy(r.Context(), cookie.Value)
    }
    http.SetCookie(w, &http.Cookie{
        Name:   "session_id",
        MaxAge: -1,
    })
    w.WriteHeader(http.StatusOK)
}
```

### Sessions vs JWT

```
Критерий              Session + Cookie              JWT
────────────────────  ────────────────────────────   ─────────────────────────────
Хранение состояния    Сервер (Redis/DB)              Клиент (token)
Масштабируемость      Нужен shared store (Redis)     Stateless, любой сервер
Размер                Cookie ~40 bytes (session ID)  Token ~800+ bytes
Отзыв (revocation)    Удалить из Redis — мгновенно   Нужен blacklist или короткий TTL
Logout everywhere     Удалить все sessions user      Сложно без blacklist
XSS уязвимость        httpOnly cookie — защищён      localStorage → читаем через XSS
CSRF уязвимость       Нужен CSRF token               Bearer header — нет CSRF
Микросервисы           Нужен shared Redis             Любой сервис проверит signature
Mobile клиенты        Cookies неудобны               Bearer token удобен
Нагрузка на сервер    GET из Redis на каждый запрос   Только CPU на verify signature
Payload данные        Любые (хранятся на сервере)    Ограничены (увеличивают размер)
```

```
Когда что выбирать:

Sessions — классические web-apps, нужен мгновенный logout, sensitive данные в сессии
JWT      — SPA + API, микросервисы, мобильные клиенты, serverless
Гибрид   — JWT access (5 мин) + session-like refresh в Redis
```

---

## RBAC / ABAC (Краткий обзор)

> Подробная реализация: [04-authorization.md](04-authorization.md)

### RBAC в контексте аутентификации

Роль пользователя определяется при логине и включается в токен/сессию:

```go
// Role assigned during registration or by admin
type User struct {
    ID           string
    Email        string
    PasswordHash string
    Role         string // "admin", "moderator", "user"
}

// Role goes into JWT claims or session
func loginUser(user *User) {
    // JWT: role is in the token payload
    token := GenerateTokens(user.ID, user.Role, secret)

    // Session: role is stored server-side
    session := Session{UserID: user.ID, Role: user.Role}
    store.Create(ctx, session)
}
```

### Permission Middleware (связь auth + authz)

```go
// Combines authentication (who?) + authorization (can they?)
func requireRole(roles ...string) func(http.Handler) http.Handler {
    allowed := make(map[string]bool, len(roles))
    for _, r := range roles {
        allowed[r] = true
    }
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            claims := claimsFromContext(r.Context()) // set by authMiddleware
            if claims == nil {
                http.Error(w, "unauthorized", http.StatusUnauthorized)
                return
            }
            if !allowed[claims.Role] {
                http.Error(w, "forbidden", http.StatusForbidden)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

// Usage with router
mux.Handle("DELETE /users/{id}",
    authMiddleware(secret)(
        requireRole("admin")(
            http.HandlerFunc(deleteUserHandler),
        ),
    ),
)
```

```
RBAC: роль → набор прав. Простой, достаточен для большинства приложений.
ABAC: решение на основе атрибутов (роль + владелец ресурса + время + ...).
      Гибче, но сложнее. Нужен когда RBAC недостаточно.

Подробнее с примерами → 04-authorization.md
```

---

## Частые вопросы на собеседовании

### Где хранить JWT?

```
Вариант                  XSS       CSRF      Удобство
───────────────────────  ────────  ────────  ────────
localStorage             Уязвим    Нет       Просто
sessionStorage           Уязвим    Нет       Просто, но теряется при закрытии
httpOnly cookie           Защищён   Уязвим    Нужен CSRF protection
httpOnly cookie + CSRF    Защищён   Защищён   Рекомендуемый вариант

Лучший вариант: httpOnly + Secure + SameSite=Strict cookie.
JS не может прочитать → XSS не украдёт токен.
SameSite=Strict → защита от CSRF.
Если нужен cross-site: SameSite=Lax + CSRF-token.

Почему НЕ localStorage:
  - Любой XSS скрипт прочитает: localStorage.getItem("token")
  - В httpOnly cookie — JS вообще не имеет доступа
  - Даже sanitized input может содержать XSS через 3rd party скрипты
```

### Как отозвать JWT?

```
Проблема: JWT stateless. Подпись валидна до exp. Нельзя "удалить" токен.

Стратегии:

1. Short-lived access token (5-15 мин)
   + Минимальное окно уязвимости
   + Не нужен blacklist
   - Частый refresh

2. Blacklist (jti в Redis с TTL)
   + Мгновенный revoke
   - Нужен Redis на каждый запрос (теряется stateless)
   - По сути та же сессия

3. Token versioning (token_version в БД)
   + "Logout everywhere" одной операцией
   - Запрос к БД на каждый request

4. Refresh token rotation + семейства
   + Обнаружение кражи refresh token
   + Не нужен blacklist для access token

На собеседовании: объяснить trade-offs, предложить комбинацию
(короткий access + refresh rotation + blacklist для critical cases)
```

### Refresh Token Rotation

```
Цель: обнаружить кражу refresh token.

Как работает:
1. User логинится → access_token + refresh_token_1
2. Access истёк → POST /refresh { refresh_token_1 }
3. Server: выдаёт access_token_2 + refresh_token_2, инвалидирует refresh_token_1
4. Если кто-то использует refresh_token_1 снова →
   СИГНАЛ: token reuse detected → invalidate ВСЕ токены семейства
```

```go
// Token family — all refresh tokens from one login session
type RefreshTokenRecord struct {
    TokenHash string    // SHA256 of the refresh token
    UserID    string
    FamilyID  string    // shared across rotations
    Used      bool      // true = already rotated
    ExpiresAt time.Time
}

func (s *AuthService) RefreshTokens(ctx context.Context, oldRefreshToken string) (string, string, error) {
    hash := sha256Hex(oldRefreshToken)
    record, err := s.repo.GetRefreshToken(ctx, hash)
    if err != nil {
        return "", "", ErrInvalidToken
    }

    // Reuse detection: if token was already used, revoke entire family
    if record.Used {
        s.repo.RevokeTokenFamily(ctx, record.FamilyID) // revoke ALL tokens in family
        return "", "", ErrTokenReuse // possible theft detected!
    }

    // Mark current token as used
    s.repo.MarkUsed(ctx, hash)

    // Issue new pair with same family ID
    newAccess, _ := GenerateTokens(record.UserID, role, secret)
    newRefresh := generateRefreshToken()
    s.repo.StoreRefreshToken(ctx, RefreshTokenRecord{
        TokenHash: sha256Hex(newRefresh),
        UserID:    record.UserID,
        FamilyID:  record.FamilyID, // same family
        ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
    })

    return newAccess, newRefresh, nil
}
```

### Stateless vs Stateful Auth

```
                     Stateless (JWT)              Stateful (Sessions)
─────────────────    ────────────────────────     ─────────────────────────
Где живёт            В самом токене               В хранилище (Redis, DB)
Верификация          Проверка подписи (CPU)       Lookup по ID (I/O)
Горизонтальное       Любой сервер                 Нужен shared store
масштабирование
Revocation           Сложно (нужен blacklist)     Просто (delete key)
Данные               В payload (лимит размера)    Любые данные на сервере
После компрометации  Ждать expire или blacklist    Удалить session мгновенно

Вопрос-ловушка: "JWT это stateless?"
Ответ: Чистый JWT — да. Но как только добавляешь blacklist или refresh token
в Redis — это уже "partially stateful". На практике чисто stateless auth
встречается редко.
```

### Типичный auth flow для production

```
Регистрация:
  1. Validate input (email format, password strength)
  2. Check email uniqueness
  3. Hash password (bcrypt/argon2id)
  4. Save user to DB
  5. Send email verification link (JWT с коротким TTL)
  6. Return 201 Created

Логин:
  1. Find user by email
  2. Compare password hash (bcrypt)
  3. Check email verified
  4. Check account not locked (brute-force protection)
  5. Generate access + refresh tokens
  6. Store refresh token record (family_id)
  7. Set tokens in httpOnly cookies
  8. Return 200 OK

Аутентифицированный запрос:
  1. Middleware: extract token from cookie/header
  2. Validate signature + expiry
  3. (Optional) Check blacklist
  4. Inject claims into context
  5. Handler: extract claims, process request

Refresh:
  1. Validate refresh token
  2. Check not revoked, not reused
  3. Issue new access + refresh (rotation)
  4. Mark old refresh as used

Logout:
  1. Blacklist access token (if using blacklist)
  2. Revoke refresh token
  3. Clear cookies
  4. (Optional) Revoke all sessions — "logout everywhere"
```
