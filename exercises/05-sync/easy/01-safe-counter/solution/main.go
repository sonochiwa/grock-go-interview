package safe_counter

import (
	"sync"
	"sync/atomic"
)

type Counter struct {
	mu sync.Mutex
	v  int64
}

func (c *Counter) Inc()         { c.mu.Lock(); c.v++; c.mu.Unlock() }
func (c *Counter) Dec()         { c.mu.Lock(); c.v--; c.mu.Unlock() }
func (c *Counter) Value() int64 { c.mu.Lock(); defer c.mu.Unlock(); return c.v }

type AtomicCounter struct {
	v atomic.Int64
}

func (c *AtomicCounter) Inc()         { c.v.Add(1) }
func (c *AtomicCounter) Dec()         { c.v.Add(-1) }
func (c *AtomicCounter) Value() int64 { return c.v.Load() }
