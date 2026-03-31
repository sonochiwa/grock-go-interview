package sharded_counter

import (
	"sync"
	"sync/atomic"
)

// --- Вариант 1: Mutex ---

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

// --- Вариант 2: Single Atomic ---

type AtomicCounter struct {
	v atomic.Int64
}

func (c *AtomicCounter) Inc()         { c.v.Add(1) }
func (c *AtomicCounter) Value() int64 { return c.v.Load() }

// --- Вариант 3: Sharded (TODO) ---

// TODO: добавь padding чтобы каждый шард занимал отдельную cache line (64 bytes)
type paddedCounter struct {
	v atomic.Int64
	// TODO: padding
}

type ShardedCounter struct {
	shards []paddedCounter
	n      int
}

// TODO: создай ShardedCounter с n шардами
func NewShardedCounter(n int) *ShardedCounter {
	return nil
}

// TODO: Inc — выбери шард (по round-robin или runtime ID) и инкрементируй
func (c *ShardedCounter) Inc() {}

// TODO: Value — сумма всех шардов
func (c *ShardedCounter) Value() int64 { return 0 }
