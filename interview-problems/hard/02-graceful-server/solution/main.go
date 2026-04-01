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

func (gs *GracefulServer) trackingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gs.active.Add(1)
		defer gs.active.Add(-1)
		next.ServeHTTP(w, r)
	})
}

func (gs *GracefulServer) Start(addr string) error {
	gs.server = &http.Server{
		Addr:    addr,
		Handler: gs.trackingMiddleware(gs.handler),
	}
	return gs.server.ListenAndServe()
}

func (gs *GracefulServer) Shutdown(ctx context.Context) error {
	if gs.server == nil {
		return nil
	}
	return gs.server.Shutdown(ctx)
}
