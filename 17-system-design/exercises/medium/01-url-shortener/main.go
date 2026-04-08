package url_shortener

import (
	"errors"
	"sync"
	"sync/atomic"
)

var ErrNotFound = errors.New("short URL not found")

type Shortener struct {
	mu        sync.RWMutex
	counter   atomic.Int64
	long2code map[string]string
	code2long map[string]string
}

func NewShortener() *Shortener {
	return &Shortener{
		long2code: make(map[string]string),
		code2long: make(map[string]string),
	}
}

// TODO: base62 encoding (0-9, a-z, A-Z)
func toBase62(n int64) string {
	return ""
}

// TODO: если longURL уже сокращён — верни тот же код
// Иначе сгенерируй новый через counter + base62
func (s *Shortener) Shorten(longURL string) string {
	return ""
}

// TODO: найди оригинальный URL по коду
func (s *Shortener) Resolve(code string) (string, error) {
	return "", ErrNotFound
}
