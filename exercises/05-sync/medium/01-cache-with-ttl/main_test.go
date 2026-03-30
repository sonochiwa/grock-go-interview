package cache_with_ttl

import (
	"sync"
	"testing"
	"time"
)

func TestSetGet(t *testing.T) {
	c := NewCache[string, int](time.Minute)
	defer c.Close()

	c.Set("a", 1, time.Minute)
	v, ok := c.Get("a")
	if !ok || v != 1 {
		t.Errorf("Get(a) = (%d, %v), want (1, true)", v, ok)
	}
}

func TestGetMissing(t *testing.T) {
	c := NewCache[string, int](time.Minute)
	defer c.Close()

	_, ok := c.Get("nope")
	if ok {
		t.Error("expected miss")
	}
}

func TestExpiration(t *testing.T) {
	c := NewCache[string, int](time.Minute)
	defer c.Close()

	c.Set("a", 1, 50*time.Millisecond)
	time.Sleep(80 * time.Millisecond)

	_, ok := c.Get("a")
	if ok {
		t.Error("expected expired")
	}
}

func TestDelete(t *testing.T) {
	c := NewCache[string, int](time.Minute)
	defer c.Close()

	c.Set("a", 1, time.Minute)
	c.Delete("a")
	_, ok := c.Get("a")
	if ok {
		t.Error("expected deleted")
	}
}

func TestLen(t *testing.T) {
	c := NewCache[string, int](time.Minute)
	defer c.Close()

	c.Set("a", 1, time.Minute)
	c.Set("b", 2, 50*time.Millisecond)
	if n := c.Len(); n != 2 {
		t.Errorf("Len() = %d, want 2", n)
	}

	time.Sleep(80 * time.Millisecond)
	if n := c.Len(); n != 1 {
		t.Errorf("Len() after expiry = %d, want 1", n)
	}
}

func TestConcurrent(t *testing.T) {
	c := NewCache[int, int](10 * time.Millisecond)
	defer c.Close()

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Set(i, i, time.Second)
			c.Get(i)
		}()
	}
	wg.Wait()
}
