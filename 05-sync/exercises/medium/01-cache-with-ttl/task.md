# Cache with TTL

Реализуй generic кэш с временем жизни записей:

- `NewCache[K comparable, V any](cleanupInterval time.Duration) *Cache[K, V]`
- `Set(key K, value V, ttl time.Duration)`
- `Get(key K) (V, bool)` — false если не найден или expired
- `Delete(key K)`
- `Len() int` — только не-expired
- `Close()` — останавливает фоновую cleanup горутину

Goroutine-safe! Expired записи удаляются при Get и фоновым cleanup.
