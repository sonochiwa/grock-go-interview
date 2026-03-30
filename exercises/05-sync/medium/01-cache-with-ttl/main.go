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
	// TODO: запусти горутину для периодической очистки expired записей
	return c
}

// TODO: сохрани значение с TTL
func (c *Cache[K, V]) Set(key K, value V, ttl time.Duration) {
}

// TODO: верни значение если существует и не expired
func (c *Cache[K, V]) Get(key K) (V, bool) {
	var zero V
	return zero, false
}

// TODO: удали ключ
func (c *Cache[K, V]) Delete(key K) {
}

// TODO: количество не-expired записей
func (c *Cache[K, V]) Len() int {
	return 0
}

// TODO: останови cleanup горутину
func (c *Cache[K, V]) Close() {
}
