package generic_set

type Set[T comparable] struct {
	m map[T]struct{}
}

func NewSet[T comparable]() Set[T] {
	return Set[T]{m: make(map[T]struct{})}
}

func (s Set[T]) Add(v T)           { s.m[v] = struct{}{} }
func (s Set[T]) Remove(v T)        { delete(s.m, v) }
func (s Set[T]) Contains(v T) bool { _, ok := s.m[v]; return ok }
func (s Set[T]) Len() int          { return len(s.m) }

func (s Set[T]) Values() []T {
	vals := make([]T, 0, len(s.m))
	for v := range s.m {
		vals = append(vals, v)
	}
	return vals
}

func (s Set[T]) Union(other Set[T]) Set[T] {
	result := NewSet[T]()
	for v := range s.m {
		result.Add(v)
	}
	for v := range other.m {
		result.Add(v)
	}
	return result
}

func (s Set[T]) Intersection(other Set[T]) Set[T] {
	result := NewSet[T]()
	for v := range s.m {
		if other.Contains(v) {
			result.Add(v)
		}
	}
	return result
}
