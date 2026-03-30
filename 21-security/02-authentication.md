# Authentication

## JWT (JSON Web Token)

```
Структура: header.payload.signature (base64url encoded)

Header:  {"alg": "HS256", "typ": "JWT"}
Payload: {"sub": "user123", "exp": 1234567890, "role": "admin"}
Signature: HMAC-SHA256(base64(header) + "." + base64(payload), secret)

Access Token: короткоживущий (15 мин), для доступа к API
Refresh Token: долгоживущий (7 дней), для обновления access token
```

```go
import "github.com/golang-jwt/jwt/v5"

// Генерация
type Claims struct {
    UserID string `json:"user_id"`
    Role   string `json:"role"`
    jwt.RegisteredClaims
}

func GenerateTokens(userID, role, secret string) (accessToken, refreshToken string, err error) {
    // Access token (15 мин)
    accessClaims := Claims{
        UserID: userID,
        Role:   role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            Issuer:    "my-service",
        },
    }
    access := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
    accessToken, err = access.SignedString([]byte(secret))
    if err != nil {
        return "", "", err
    }

    // Refresh token (7 дней)
    refreshClaims := jwt.RegisteredClaims{
        ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
        IssuedAt:  jwt.NewNumericDate(time.Now()),
        Subject:   userID,
    }
    refresh := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
    refreshToken, err = refresh.SignedString([]byte(secret))

    return accessToken, refreshToken, err
}

// Валидация
func ValidateToken(tokenString, secret string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{},
        func(token *jwt.Token) (any, error) {
            // Проверить алгоритм (защита от algorithm switching attack)
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

// Middleware
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

### JWT Best Practices

```
1. HS256 (HMAC) — для single service (один secret)
   RS256 (RSA) — для microservices (public key для верификации)
   ES256 (ECDSA) — компактнее RSA, тоже asymmetric

2. Короткий access token (5-15 мин)
   Refresh token в httpOnly cookie или отдельном хранилище

3. НЕ хранить sensitive данные в payload (видны всем!)

4. Token revocation:
   - Blacklist в Redis (token_id → revoked_at, TTL = token.exp)
   - Или: короткий TTL + refresh token rotation

5. Refresh token rotation:
   Каждый refresh → новый refresh token + invalidate старый
   Защита от replay attack
```

## Password Hashing

```go
import "golang.org/x/crypto/bcrypt"

// Хеширование
func HashPassword(password string) (string, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost) // cost=10
    return string(hash), err
}

// Проверка
func CheckPassword(hash, password string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}

// Argon2id (рекомендуется для новых проектов — memory-hard)
import "golang.org/x/crypto/argon2"

func HashPasswordArgon2(password string) string {
    salt := make([]byte, 16)
    rand.Read(salt)
    hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
    // Сохранить: $argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>
    return encodeArgon2(salt, hash)
}
```

## OAuth2 Flow

```
Authorization Code Flow (для web apps):

1. User → App: "Login with Google"
2. App → Google: redirect to /authorize?
     client_id=...&redirect_uri=...&scope=email&response_type=code&state=random
3. User → Google: вводит логин/пароль, даёт consent
4. Google → App: redirect to callback?code=AUTH_CODE&state=random
5. App → Google (server-to-server):
     POST /token { code=AUTH_CODE, client_id, client_secret, grant_type=authorization_code }
6. Google → App: { access_token, refresh_token, id_token }
7. App: декодирует id_token → получает email, name → создаёт сессию
```

```go
import "golang.org/x/oauth2"
import "golang.org/x/oauth2/google"

var oauthConfig = &oauth2.Config{
    ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
    ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
    RedirectURL:  "http://localhost:8080/auth/callback",
    Scopes:       []string{"openid", "email", "profile"},
    Endpoint:     google.Endpoint,
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
    state := generateRandomState() // CSRF protection
    // Сохранить state в cookie/session
    url := oauthConfig.AuthCodeURL(state)
    http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func handleCallback(w http.ResponseWriter, r *http.Request) {
    // Проверить state!
    code := r.URL.Query().Get("code")
    token, err := oauthConfig.Exchange(r.Context(), code)
    if err != nil {
        http.Error(w, "exchange failed", http.StatusBadRequest)
        return
    }

    // Получить user info
    client := oauthConfig.Client(r.Context(), token)
    resp, _ := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
    defer resp.Body.Close()

    var userInfo struct {
        Email string `json:"email"`
        Name  string `json:"name"`
    }
    json.NewDecoder(resp.Body).Decode(&userInfo)

    // Создать сессию / JWT
}
```
