package concurrent_map

import (
	"fmt"
	"hash/fnv"
	"sync"
)

type shard[K comparable, V any] struct {
	mu sync.RWMutex
	m  map[K]V
}

type ShardedMap[K comparable, V any] struct {
	shards []shard[K, V]
	count  int
}

func NewShardedMap[K comparable, V any](count int) *ShardedMap[K, V] {
	sm := &ShardedMap[K, V]{
		shards: make([]shard[K, V], count),
		count:  count,
	}
	for i := range sm.shards {
		sm.shards[i].m = make(map[K]V)
	}
	return sm
}

func (sm *ShardedMap[K, V]) getShard(key K) *shard[K, V] {
	h := fnv.New32a()
	h.Write([]byte(fmt.Sprint(key)))
	return &sm.shards[h.Sum32()%uint32(sm.count)]
}

// TODO: реализуй Set
func (sm *ShardedMap[K, V]) Set(key K, value V) {}

// TODO: реализуй Get
func (sm *ShardedMap[K, V]) Get(key K) (V, bool) {
	var zero V
	return zero, false
}

// TODO: реализуй Delete
func (sm *ShardedMap[K, V]) Delete(key K) {}

// TODO: реализуй Len (сумма по всем шардам)
func (sm *ShardedMap[K, V]) Len() int { return 0 }

// TODO: реализуй Range — вызывай f для каждого k,v, останови если f вернёт false
func (sm *ShardedMap[K, V]) Range(f func(K, V) bool) {}
