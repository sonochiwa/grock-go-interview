package cache_with_ttl

import (
	"sync"
	"time"
)

type item[V any] struct {
	value     V
	expiresAt time.Time
}

type Cache[K comparable, V any] struct {
	mu    sync.RWMutex
	items map[K]item[V]
	done  chan struct{}
}

func NewCache[K comparable, V any](cleanupInterval time.Duration) *Cache[K, V] {
	c := &Cache[K, V]{
		items: make(map[K]item[V]),
		done:  make(chan struct{}),
	}
	go c.cleanup(cleanupInterval)
	return c
}

func (c *Cache[K, V]) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for k, v := range c.items {
				if now.After(v.expiresAt) {
					delete(c.items, k)
				}
			}
			c.mu.Unlock()
		}
	}
}

func (c *Cache[K, V]) Set(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	c.items[key] = item[V]{value: value, expiresAt: time.Now().Add(ttl)}
	c.mu.Unlock()
}

func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	it, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		var zero V
		return zero, false
	}
	if time.Now().After(it.expiresAt) {
		c.Delete(key)
		var zero V
		return zero, false
	}
	return it.value, true
}

func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}

func (c *Cache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	count := 0
	now := time.Now()
	for _, v := range c.items {
		if now.Before(v.expiresAt) {
			count++
		}
	}
	return count
}

func (c *Cache[K, V]) Close() {
	close(c.done)
}
