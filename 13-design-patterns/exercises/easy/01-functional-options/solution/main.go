package functional_options

import "time"

type Server struct {
	host     string
	port     int
	timeout  time.Duration
	maxConns int
}

type Option func(*Server)

func (s *Server) Host() string           { return s.host }
func (s *Server) Port() int              { return s.port }
func (s *Server) Timeout() time.Duration { return s.timeout }
func (s *Server) MaxConns() int          { return s.maxConns }

func NewServer(opts ...Option) *Server {
	s := &Server{
		host:     "localhost",
		port:     8080,
		timeout:  30 * time.Second,
		maxConns: 100,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func WithHost(host string) Option        { return func(s *Server) { s.host = host } }
func WithPort(port int) Option           { return func(s *Server) { s.port = port } }
func WithTimeout(d time.Duration) Option { return func(s *Server) { s.timeout = d } }
func WithMaxConns(n int) Option          { return func(s *Server) { s.maxConns = n } }
