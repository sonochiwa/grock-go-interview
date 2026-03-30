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

func (sm *ShardedMap[K, V]) Set(key K, value V) {
	s := sm.getShard(key)
	s.mu.Lock()
	s.m[key] = value
	s.mu.Unlock()
}

func (sm *ShardedMap[K, V]) Get(key K) (V, bool) {
	s := sm.getShard(key)
	s.mu.RLock()
	v, ok := s.m[key]
	s.mu.RUnlock()
	return v, ok
}

func (sm *ShardedMap[K, V]) Delete(key K) {
	s := sm.getShard(key)
	s.mu.Lock()
	delete(s.m, key)
	s.mu.Unlock()
}

func (sm *ShardedMap[K, V]) Len() int {
	total := 0
	for i := range sm.shards {
		sm.shards[i].mu.RLock()
		total += len(sm.shards[i].m)
		sm.shards[i].mu.RUnlock()
	}
	return total
}

func (sm *ShardedMap[K, V]) Range(f func(K, V) bool) {
	for i := range sm.shards {
		sm.shards[i].mu.RLock()
		for k, v := range sm.shards[i].m {
			if !f(k, v) {
				sm.shards[i].mu.RUnlock()
				return
			}
		}
		sm.shards[i].mu.RUnlock()
	}
}
