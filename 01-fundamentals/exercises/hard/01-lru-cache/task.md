# LRU Cache

Реализуй generic LRU Cache: `type LRUCache[K comparable, V any]` с методами `Get(key K) (V, bool)`, `Put(key K, value V)`, `Len() int`. Конструктор `NewLRUCache[K, V](capacity int)`. При превышении capacity удаляется least recently used элемент.

## Требования

- Generic типы для ключа и значения
- `Get` помечает элемент как recently used
- `Put` добавляет или обновляет элемент; при превышении capacity вытесняет LRU элемент
- `Len` возвращает текущее количество элементов
- O(1) для Get и Put
- Capacity >= 1

## Пример

```go
cache := NewLRUCache[string, int](2)
cache.Put("a", 1)
cache.Put("b", 2)
v, ok := cache.Get("a") // v=1, ok=true (a is now most recently used)
cache.Put("c", 3)       // evicts "b" (least recently used)
_, ok = cache.Get("b")  // ok=false (evicted)
```
