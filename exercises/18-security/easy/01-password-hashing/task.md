# Password Hashing

Реализуй безопасное хранение паролей:

- `HashPassword(password string) (string, error)` — bcrypt hash
- `CheckPassword(hash, password string) bool` — проверка
- `GenerateToken(n int) (string, error)` — crypto-safe random token (base64url, n bytes)

Используй `golang.org/x/crypto/bcrypt` и `crypto/rand`.
