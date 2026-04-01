package graceful_server

import (
	"context"
	"net/http"
	"sync/atomic"
)

type GracefulServer struct {
	server  *http.Server
	handler http.Handler
	active  atomic.Int64
}

func NewGracefulServer(handler http.Handler) *GracefulServer {
	return &GracefulServer{handler: handler}
}

func (gs *GracefulServer) ActiveRequests() int64 {
	return gs.active.Load()
}

// TODO: оберни handler для tracking active requests
func (gs *GracefulServer) trackingMiddleware(next http.Handler) http.Handler {
	return next // TODO
}

// TODO: запусти http.Server
func (gs *GracefulServer) Start(addr string) error {
	return nil
}

// TODO: graceful shutdown
func (gs *GracefulServer) Shutdown(ctx context.Context) error {
	return nil
}
