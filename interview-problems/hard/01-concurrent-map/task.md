# Concurrent Sharded Map

Реализуй шардированную goroutine-safe map:

- `NewShardedMap[K comparable, V any](shards int) *ShardedMap[K, V]`
- `Set(key K, value V)`, `Get(key K) (V, bool)`, `Delete(key K)`
- `Len() int`, `Range(func(K, V) bool)` — итерация, false останавливает

Используй хеширование ключа для выбора шарда. RWMutex на каждый шард.
