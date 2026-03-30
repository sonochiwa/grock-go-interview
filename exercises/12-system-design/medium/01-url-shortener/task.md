# URL Shortener

Реализуй in-memory URL shortener:

- `NewShortener() *Shortener`
- `Shorten(longURL string) string` — возвращает короткий код (base62, 6+ символов)
- `Resolve(code string) (string, error)` — возвращает оригинальный URL
- Один и тот же longURL всегда даёт один и тот же code (idempotent)

Goroutine-safe!
