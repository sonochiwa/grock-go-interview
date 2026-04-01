package lru_cache

// LRUCache — generic кэш с вытеснением least recently used элементов.
// TODO: определи внутреннюю структуру
type LRUCache[K comparable, V any] struct {
	// TODO: добавь поля
}

// NewLRUCache создаёт новый LRU кэш с заданной ёмкостью.
// TODO: реализуй конструктор
func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	return nil
}

// Get возвращает значение по ключу и помечает элемент как recently used.
// Возвращает false вторым аргументом если ключ не найден.
// TODO: реализуй метод
func (c *LRUCache[K, V]) Get(key K) (V, bool) {
	var zero V
	return zero, false
}

// Put добавляет или обновляет элемент в кэше.
// При превышении capacity удаляет least recently used элемент.
// TODO: реализуй метод
func (c *LRUCache[K, V]) Put(key K, value V) {
}

// Len возвращает текущее количество элементов в кэше.
// TODO: реализуй метод
func (c *LRUCache[K, V]) Len() int {
	return 0
}
