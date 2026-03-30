package parallel_fetch

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestFetchAllSuccess(t *testing.T) {
	urls := []string{"https://a.com", "https://b.com", "https://c.com"}
	fetch := func(_ context.Context, url string) (string, error) {
		return "body:" + url, nil
	}

	results, err := FetchAll(context.Background(), urls, fetch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != len(urls) {
		t.Fatalf("got %d results, want %d", len(results), len(urls))
	}

	for i, r := range results {
		if r.URL != urls[i] {
			t.Errorf("results[%d].URL = %q, want %q", i, r.URL, urls[i])
		}
		expected := "body:" + urls[i]
		if r.Body != expected {
			t.Errorf("results[%d].Body = %q, want %q", i, r.Body, expected)
		}
		if r.Err != nil {
			t.Errorf("results[%d].Err = %v, want nil", i, r.Err)
		}
	}
}

func TestFetchAllWithErrors(t *testing.T) {
	urls := []string{"https://ok.com", "https://fail.com", "https://ok2.com"}
	errFetch := errors.New("fetch failed")

	fetch := func(_ context.Context, url string) (string, error) {
		if url == "https://fail.com" {
			return "", errFetch
		}
		return "body:" + url, nil
	}

	results, err := FetchAll(context.Background(), urls, fetch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results[0].Err != nil {
		t.Errorf("results[0].Err = %v, want nil", results[0].Err)
	}
	if !errors.Is(results[1].Err, errFetch) {
		t.Errorf("results[1].Err = %v, want %v", results[1].Err, errFetch)
	}
	if results[2].Err != nil {
		t.Errorf("results[2].Err = %v, want nil", results[2].Err)
	}
}

func TestFetchAllContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	urls := make([]string, 20)
	for i := range urls {
		urls[i] = fmt.Sprintf("https://example.com/%d", i)
	}

	var started atomic.Int32
	fetch := func(ctx context.Context, url string) (string, error) {
		started.Add(1)
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(5 * time.Second):
			return "body", nil
		}
	}

	// Cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := FetchAll(ctx, urls, fetch)
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestFetchAllEmpty(t *testing.T) {
	fetch := func(_ context.Context, url string) (string, error) {
		return "body", nil
	}

	results, err := FetchAll(context.Background(), []string{}, fetch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

func TestFetchAllConcurrencyLimit(t *testing.T) {
	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32

	urls := make([]string, 30)
	for i := range urls {
		urls[i] = fmt.Sprintf("https://example.com/%d", i)
	}

	fetch := func(_ context.Context, url string) (string, error) {
		cur := concurrent.Add(1)
		for {
			old := maxConcurrent.Load()
			if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
		concurrent.Add(-1)
		return "body", nil
	}

	_, err := FetchAll(context.Background(), urls, fetch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	max := maxConcurrent.Load()
	if max > 10 {
		t.Errorf("max concurrent fetches = %d, want <= 10", max)
	}
}
