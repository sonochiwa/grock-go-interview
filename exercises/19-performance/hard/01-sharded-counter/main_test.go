package sharded_counter

import (
	"runtime"
	"sync"
	"testing"
)

type Counter interface {
	Inc()
	Value() int64
}

func testCorrectness(t *testing.T, name string, c Counter) {
	t.Run(name, func(t *testing.T) {
		var wg sync.WaitGroup
		n := 10000
		wg.Add(n)
		for range n {
			go func() {
				defer wg.Done()
				c.Inc()
			}()
		}
		wg.Wait()
		if v := c.Value(); v != int64(n) {
			t.Errorf("Value() = %d, want %d", v, n)
		}
	})
}

func TestMutexCounter(t *testing.T) {
	testCorrectness(t, "mutex", &MutexCounter{})
}

func TestAtomicCounter(t *testing.T) {
	testCorrectness(t, "atomic", &AtomicCounter{})
}

func TestShardedCounter(t *testing.T) {
	testCorrectness(t, "sharded", NewShardedCounter(runtime.GOMAXPROCS(0)))
}

func benchCounter(b *testing.B, c Counter) {
	b.SetParallelism(8)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc()
		}
	})
}

func BenchmarkMutexCounter(b *testing.B) {
	benchCounter(b, &MutexCounter{})
}

func BenchmarkAtomicCounter(b *testing.B) {
	benchCounter(b, &AtomicCounter{})
}

func BenchmarkShardedCounter(b *testing.B) {
	benchCounter(b, NewShardedCounter(runtime.GOMAXPROCS(0)))
}
