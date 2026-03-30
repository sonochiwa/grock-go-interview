package safe_counter

import (
	"sync"
	"testing"
)

type CounterIface interface {
	Inc()
	Dec()
	Value() int64
}

func testCounter(t *testing.T, name string, c CounterIface) {
	t.Run(name+"/basic", func(t *testing.T) {
		c.Inc()
		c.Inc()
		c.Dec()
		if v := c.Value(); v != 1 {
			t.Errorf("Value() = %d, want 1", v)
		}
	})
}

func testConcurrent(t *testing.T, name string, newCounter func() CounterIface) {
	t.Run(name+"/concurrent", func(t *testing.T) {
		c := newCounter()
		var wg sync.WaitGroup
		n := 1000
		wg.Add(2 * n)
		for range n {
			go func() { defer wg.Done(); c.Inc() }()
			go func() { defer wg.Done(); c.Dec() }()
		}
		wg.Wait()
		if v := c.Value(); v != 0 {
			t.Errorf("Value() = %d after equal inc/dec, want 0", v)
		}
	})
}

func TestCounter(t *testing.T) {
	testCounter(t, "mutex", &Counter{})
	testConcurrent(t, "mutex", func() CounterIface { return &Counter{} })
}

func TestAtomicCounter(t *testing.T) {
	testCounter(t, "atomic", &AtomicCounter{})
	testConcurrent(t, "atomic", func() CounterIface { return &AtomicCounter{} })
}
