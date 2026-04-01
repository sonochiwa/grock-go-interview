package lru_cache

import "testing"

func TestLRUCachePutAndGet(t *testing.T) {
	cache := NewLRUCache[string, int](3)

	cache.Put("a", 1)
	cache.Put("b", 2)
	cache.Put("c", 3)

	tests := []struct {
		key      string
		expected int
		found    bool
	}{
		{"a", 1, true},
		{"b", 2, true},
		{"c", 3, true},
		{"d", 0, false},
	}

	for _, tt := range tests {
		t.Run("get_"+tt.key, func(t *testing.T) {
			val, ok := cache.Get(tt.key)
			if ok != tt.found {
				t.Errorf("Get(%q) found = %v, want %v", tt.key, ok, tt.found)
			}
			if val != tt.expected {
				t.Errorf("Get(%q) = %v, want %v", tt.key, val, tt.expected)
			}
		})
	}
}

func TestLRUCacheEviction(t *testing.T) {
	cache := NewLRUCache[string, int](2)

	cache.Put("a", 1)
	cache.Put("b", 2)

	// a is LRU, adding c should evict a
	cache.Put("c", 3)

	if _, ok := cache.Get("a"); ok {
		t.Error("expected 'a' to be evicted")
	}
	if v, ok := cache.Get("b"); !ok || v != 2 {
		t.Errorf("expected 'b' = 2, got %v, %v", v, ok)
	}
	if v, ok := cache.Get("c"); !ok || v != 3 {
		t.Errorf("expected 'c' = 3, got %v, %v", v, ok)
	}
}

func TestLRUCacheGetUpdatesRecency(t *testing.T) {
	cache := NewLRUCache[string, int](2)

	cache.Put("a", 1)
	cache.Put("b", 2)

	// Access a, making b the LRU
	cache.Get("a")

	// Adding c should evict b (LRU), not a
	cache.Put("c", 3)

	if _, ok := cache.Get("b"); ok {
		t.Error("expected 'b' to be evicted after 'a' was accessed")
	}
	if v, ok := cache.Get("a"); !ok || v != 1 {
		t.Errorf("expected 'a' = 1, got %v, %v", v, ok)
	}
	if v, ok := cache.Get("c"); !ok || v != 3 {
		t.Errorf("expected 'c' = 3, got %v, %v", v, ok)
	}
}

func TestLRUCacheUpdateExistingKey(t *testing.T) {
	cache := NewLRUCache[string, int](2)

	cache.Put("a", 1)
	cache.Put("b", 2)

	// Update a, making b the LRU
	cache.Put("a", 10)

	if v, ok := cache.Get("a"); !ok || v != 10 {
		t.Errorf("expected updated 'a' = 10, got %v, %v", v, ok)
	}

	// Adding c should evict b
	cache.Put("c", 3)

	if _, ok := cache.Get("b"); ok {
		t.Error("expected 'b' to be evicted after 'a' was updated")
	}
	if cache.Len() != 2 {
		t.Errorf("expected Len() = 2, got %d", cache.Len())
	}
}

func TestLRUCacheLen(t *testing.T) {
	cache := NewLRUCache[int, string](3)

	if cache.Len() != 0 {
		t.Errorf("expected Len() = 0, got %d", cache.Len())
	}

	cache.Put(1, "a")
	if cache.Len() != 1 {
		t.Errorf("expected Len() = 1, got %d", cache.Len())
	}

	cache.Put(2, "b")
	cache.Put(3, "c")
	if cache.Len() != 3 {
		t.Errorf("expected Len() = 3, got %d", cache.Len())
	}

	// Exceed capacity
	cache.Put(4, "d")
	if cache.Len() != 3 {
		t.Errorf("expected Len() = 3 after eviction, got %d", cache.Len())
	}
}

func TestLRUCacheCapacityOne(t *testing.T) {
	cache := NewLRUCache[string, int](1)

	cache.Put("a", 1)
	if v, ok := cache.Get("a"); !ok || v != 1 {
		t.Errorf("expected 'a' = 1, got %v, %v", v, ok)
	}

	cache.Put("b", 2)
	if _, ok := cache.Get("a"); ok {
		t.Error("expected 'a' to be evicted with capacity 1")
	}
	if v, ok := cache.Get("b"); !ok || v != 2 {
		t.Errorf("expected 'b' = 2, got %v, %v", v, ok)
	}
}

func TestLRUCacheMultipleEvictions(t *testing.T) {
	cache := NewLRUCache[int, int](3)

	for i := 1; i <= 5; i++ {
		cache.Put(i, i*10)
	}

	// 1 and 2 should be evicted
	if _, ok := cache.Get(1); ok {
		t.Error("expected key 1 to be evicted")
	}
	if _, ok := cache.Get(2); ok {
		t.Error("expected key 2 to be evicted")
	}

	// 3, 4, 5 should remain
	for i := 3; i <= 5; i++ {
		if v, ok := cache.Get(i); !ok || v != i*10 {
			t.Errorf("expected key %d = %d, got %v, %v", i, i*10, v, ok)
		}
	}
}
