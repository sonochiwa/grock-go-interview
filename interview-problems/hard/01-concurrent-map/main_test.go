package concurrent_map

import (
	"strconv"
	"sync"
	"testing"
)

func TestBasic(t *testing.T) {
	m := NewShardedMap[string, int](16)
	m.Set("a", 1)
	m.Set("b", 2)

	v, ok := m.Get("a")
	if !ok || v != 1 {
		t.Errorf("Get(a) = (%d, %v), want (1, true)", v, ok)
	}

	m.Delete("a")
	_, ok = m.Get("a")
	if ok {
		t.Error("Get(a) should be false after delete")
	}

	if m.Len() != 1 {
		t.Errorf("Len() = %d, want 1", m.Len())
	}
}

func TestRange(t *testing.T) {
	m := NewShardedMap[string, int](4)
	for i := range 10 {
		m.Set(strconv.Itoa(i), i)
	}

	count := 0
	m.Range(func(k string, v int) bool {
		count++
		return true
	})
	if count != 10 {
		t.Errorf("Range visited %d, want 10", count)
	}
}

func TestRangeStop(t *testing.T) {
	m := NewShardedMap[int, int](4)
	for i := range 100 {
		m.Set(i, i)
	}

	count := 0
	m.Range(func(k, v int) bool {
		count++
		return count < 5
	})
	if count != 5 {
		t.Errorf("Range should stop at 5, got %d", count)
	}
}

func TestConcurrent(t *testing.T) {
	m := NewShardedMap[int, int](32)
	var wg sync.WaitGroup
	n := 1000

	wg.Add(n * 3)
	for i := range n {
		go func() { defer wg.Done(); m.Set(i, i) }()
		go func() { defer wg.Done(); m.Get(i) }()
		go func() { defer wg.Done(); m.Delete(i + n) }()
	}
	wg.Wait()
}
