package sharded_counter

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// --- Mutex ---

type MutexCounter struct {
	mu sync.Mutex
	v  int64
}

func (c *MutexCounter) Inc() {
	c.mu.Lock()
	c.v++
	c.mu.Unlock()
}

func (c *MutexCounter) Value() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.v
}

// --- Atomic ---

type AtomicCounter struct {
	v atomic.Int64
}

func (c *AtomicCounter) Inc()         { c.v.Add(1) }
func (c *AtomicCounter) Value() int64 { return c.v.Load() }

// --- Sharded ---

type paddedCounter struct {
	v atomic.Int64
	_ [56]byte // padding to 64-byte cache line (8 bytes atomic + 56 padding)
}

type ShardedCounter struct {
	shards []paddedCounter
	n      int
	next   atomic.Uint64
}

func NewShardedCounter(n int) *ShardedCounter {
	if n <= 0 {
		n = runtime.GOMAXPROCS(0)
	}
	return &ShardedCounter{
		shards: make([]paddedCounter, n),
		n:      n,
	}
}

func (c *ShardedCounter) Inc() {
	idx := c.next.Add(1) % uint64(c.n)
	c.shards[idx].v.Add(1)
}

func (c *ShardedCounter) Value() int64 {
	var total int64
	for i := range c.shards {
		total += c.shards[i].v.Load()
	}
	return total
}
