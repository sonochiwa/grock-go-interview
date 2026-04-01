# JWT Auth Middleware

Реализуй JWT authentication middleware (без внешних библиотек — используй HMAC-SHA256 вручную):

- `GenerateJWT(userID string, secret []byte, ttl time.Duration) (string, error)`
- `ValidateJWT(tokenStr string, secret []byte) (Claims, error)`
- `AuthMiddleware(secret []byte) func(http.Handler) http.Handler`

Claims: `{UserID string, ExpiresAt time.Time}`

JWT формат: `base64(header).base64(payload).base64(signature)`
