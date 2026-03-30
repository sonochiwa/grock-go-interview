package functional_options

import "time"

type Server struct {
	host     string
	port     int
	timeout  time.Duration
	maxConns int
}

type Option func(*Server)

// Getters для тестов
func (s *Server) Host() string           { return s.host }
func (s *Server) Port() int              { return s.port }
func (s *Server) Timeout() time.Duration { return s.timeout }
func (s *Server) MaxConns() int          { return s.maxConns }

// TODO: конструктор с дефолтами (localhost, 8080, 30s, 100)
func NewServer(opts ...Option) *Server {
	return nil
}

// TODO: реализуй опции
func WithHost(host string) Option        { return nil }
func WithPort(port int) Option           { return nil }
func WithTimeout(d time.Duration) Option { return nil }
func WithMaxConns(n int) Option          { return nil }
