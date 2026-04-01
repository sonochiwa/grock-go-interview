package graceful_server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestStartAndShutdown(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})

	gs := NewGracefulServer(handler)

	go gs.Start(":0") // random port not testable easily, use specific
	// Use a known port for testing
	go gs.Start(":18923")
	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get("http://localhost:18923/")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "ok\n" {
		t.Errorf("body = %q, want ok", body)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := gs.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown error: %v", err)
	}
}

func TestActiveRequests(t *testing.T) {
	started := make(chan struct{})
	done := make(chan struct{})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-done // block until test says done
		fmt.Fprintln(w, "ok")
	})

	gs := NewGracefulServer(handler)
	go gs.Start(":18924")
	time.Sleep(50 * time.Millisecond)

	go http.Get("http://localhost:18924/")
	<-started

	if n := gs.ActiveRequests(); n != 1 {
		t.Errorf("ActiveRequests() = %d, want 1", n)
	}

	close(done)
	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	gs.Shutdown(ctx)
}
