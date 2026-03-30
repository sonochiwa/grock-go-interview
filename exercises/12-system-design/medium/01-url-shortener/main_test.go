package url_shortener

import (
	"sync"
	"testing"
)

func TestShortenAndResolve(t *testing.T) {
	s := NewShortener()
	code := s.Shorten("https://google.com")
	if code == "" {
		t.Fatal("Shorten returned empty string")
	}

	got, err := s.Resolve(code)
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if got != "https://google.com" {
		t.Errorf("Resolve = %q, want https://google.com", got)
	}
}

func TestIdempotent(t *testing.T) {
	s := NewShortener()
	code1 := s.Shorten("https://example.com")
	code2 := s.Shorten("https://example.com")
	if code1 != code2 {
		t.Errorf("same URL gave different codes: %q vs %q", code1, code2)
	}
}

func TestDifferentURLs(t *testing.T) {
	s := NewShortener()
	code1 := s.Shorten("https://a.com")
	code2 := s.Shorten("https://b.com")
	if code1 == code2 {
		t.Error("different URLs should have different codes")
	}
}

func TestResolveNotFound(t *testing.T) {
	s := NewShortener()
	_, err := s.Resolve("nonexistent")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestConcurrent(t *testing.T) {
	s := NewShortener()
	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			code := s.Shorten("https://example.com/" + string(rune('a'+i%26)))
			s.Resolve(code)
		}()
	}
	wg.Wait()
}
