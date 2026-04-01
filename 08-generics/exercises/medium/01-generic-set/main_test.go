package generic_set

import (
	"sort"
	"testing"
)

func TestAddContains(t *testing.T) {
	s := NewSet[int]()
	s.Add(1)
	s.Add(2)
	if !s.Contains(1) {
		t.Error("should contain 1")
	}
	if s.Contains(3) {
		t.Error("should not contain 3")
	}
	if s.Len() != 2 {
		t.Errorf("Len() = %d, want 2", s.Len())
	}
}

func TestRemove(t *testing.T) {
	s := NewSet[string]()
	s.Add("a")
	s.Add("b")
	s.Remove("a")
	if s.Contains("a") {
		t.Error("should not contain a after remove")
	}
	if s.Len() != 1 {
		t.Errorf("Len() = %d, want 1", s.Len())
	}
}

func TestUnion(t *testing.T) {
	a := NewSet[int]()
	a.Add(1)
	a.Add(2)
	b := NewSet[int]()
	b.Add(2)
	b.Add(3)

	u := a.Union(b)
	vals := u.Values()
	sort.Ints(vals)
	if len(vals) != 3 || vals[0] != 1 || vals[1] != 2 || vals[2] != 3 {
		t.Errorf("Union = %v, want [1 2 3]", vals)
	}
}

func TestIntersection(t *testing.T) {
	a := NewSet[int]()
	a.Add(1)
	a.Add(2)
	a.Add(3)
	b := NewSet[int]()
	b.Add(2)
	b.Add(3)
	b.Add(4)

	inter := a.Intersection(b)
	vals := inter.Values()
	sort.Ints(vals)
	if len(vals) != 2 || vals[0] != 2 || vals[1] != 3 {
		t.Errorf("Intersection = %v, want [2 3]", vals)
	}
}

func TestDuplicate(t *testing.T) {
	s := NewSet[int]()
	s.Add(1)
	s.Add(1)
	s.Add(1)
	if s.Len() != 1 {
		t.Errorf("duplicates: Len() = %d, want 1", s.Len())
	}
}
