package lru_cache

import "container/list"

// entry хранит ключ и значение в элементе связного списка.
type entry[K comparable, V any] struct {
	key   K
	value V
}

// LRUCache — generic кэш с вытеснением least recently used элементов.
type LRUCache[K comparable, V any] struct {
	capacity int
	items    map[K]*list.Element
	order    *list.List // front = most recent, back = least recent
}

// NewLRUCache создаёт новый LRU кэш с заданной ёмкостью.
func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	return &LRUCache[K, V]{
		capacity: capacity,
		items:    make(map[K]*list.Element, capacity),
		order:    list.New(),
	}
}

// Get возвращает значение по ключу и помечает элемент как recently used.
func (c *LRUCache[K, V]) Get(key K) (V, bool) {
	elem, ok := c.items[key]
	if !ok {
		var zero V
		return zero, false
	}

	c.order.MoveToFront(elem)
	return elem.Value.(*entry[K, V]).value, true
}

// Put добавляет или обновляет элемент в кэше.
func (c *LRUCache[K, V]) Put(key K, value V) {
	if elem, ok := c.items[key]; ok {
		c.order.MoveToFront(elem)
		elem.Value.(*entry[K, V]).value = value
		return
	}

	if c.order.Len() >= c.capacity {
		// Удаляем least recently used (back of list)
		back := c.order.Back()
		if back != nil {
			c.order.Remove(back)
			delete(c.items, back.Value.(*entry[K, V]).key)
		}
	}

	e := &entry[K, V]{key: key, value: value}
	elem := c.order.PushFront(e)
	c.items[key] = elem
}

// Len возвращает текущее количество элементов в кэше.
func (c *LRUCache[K, V]) Len() int {
	return c.order.Len()
}
