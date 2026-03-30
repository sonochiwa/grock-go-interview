package url_shortener

import (
	"errors"
	"sync"
	"sync/atomic"
)

var ErrNotFound = errors.New("short URL not found")

const base62Chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

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

func toBase62(n int64) string {
	if n == 0 {
		return "0"
	}
	var result []byte
	for n > 0 {
		result = append([]byte{base62Chars[n%62]}, result...)
		n /= 62
	}
	return string(result)
}

func (s *Shortener) Shorten(longURL string) string {
	s.mu.RLock()
	if code, ok := s.long2code[longURL]; ok {
		s.mu.RUnlock()
		return code
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	// double check
	if code, ok := s.long2code[longURL]; ok {
		return code
	}

	id := s.counter.Add(1)
	code := toBase62(id)
	s.long2code[longURL] = code
	s.code2long[code] = longURL
	return code
}

func (s *Shortener) Resolve(code string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if url, ok := s.code2long[code]; ok {
		return url, nil
	}
	return "", ErrNotFound
}
