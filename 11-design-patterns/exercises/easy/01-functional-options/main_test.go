package functional_options

import (
	"testing"
	"time"
)

func TestDefaults(t *testing.T) {
	s := NewServer()
	if s.Host() != "localhost" {
		t.Errorf("Host = %q, want localhost", s.Host())
	}
	if s.Port() != 8080 {
		t.Errorf("Port = %d, want 8080", s.Port())
	}
	if s.Timeout() != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", s.Timeout())
	}
	if s.MaxConns() != 100 {
		t.Errorf("MaxConns = %d, want 100", s.MaxConns())
	}
}

func TestWithOptions(t *testing.T) {
	s := NewServer(
		WithHost("0.0.0.0"),
		WithPort(9090),
		WithTimeout(10*time.Second),
		WithMaxConns(50),
	)
	if s.Host() != "0.0.0.0" {
		t.Errorf("Host = %q, want 0.0.0.0", s.Host())
	}
	if s.Port() != 9090 {
		t.Errorf("Port = %d, want 9090", s.Port())
	}
	if s.Timeout() != 10*time.Second {
		t.Errorf("Timeout = %v, want 10s", s.Timeout())
	}
	if s.MaxConns() != 50 {
		t.Errorf("MaxConns = %d, want 50", s.MaxConns())
	}
}

func TestPartialOptions(t *testing.T) {
	s := NewServer(WithPort(3000))
	if s.Port() != 3000 {
		t.Errorf("Port = %d, want 3000", s.Port())
	}
	if s.Host() != "localhost" {
		t.Errorf("Host should still be default, got %q", s.Host())
	}
}
