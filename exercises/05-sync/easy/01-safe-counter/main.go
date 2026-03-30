package safe_counter

import (
	"sync"
	"sync/atomic"
)

// TODO: реализуй Counter на sync.Mutex
type Counter struct {
	mu sync.Mutex
	v  int64
}

func (c *Counter) Inc()         {}
func (c *Counter) Dec()         {}
func (c *Counter) Value() int64 { return 0 }

// TODO: реализуй AtomicCounter на atomic.Int64
type AtomicCounter struct {
	v atomic.Int64
}

func (c *AtomicCounter) Inc()         {}
func (c *AtomicCounter) Dec()         {}
func (c *AtomicCounter) Value() int64 { return 0 }
