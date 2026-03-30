package generic_set

type Set[T comparable] struct {
	m map[T]struct{}
}

func NewSet[T comparable]() Set[T] {
	return Set[T]{m: make(map[T]struct{})}
}

// TODO: добавь элемент
func (s Set[T]) Add(v T) {}

// TODO: удали элемент
func (s Set[T]) Remove(v T) {}

// TODO: проверь наличие
func (s Set[T]) Contains(v T) bool { return false }

// TODO: объединение двух множеств
func (s Set[T]) Union(other Set[T]) Set[T] { return NewSet[T]() }

// TODO: пересечение
func (s Set[T]) Intersection(other Set[T]) Set[T] { return NewSet[T]() }

func (s Set[T]) Len() int { return len(s.m) }

// TODO: все элементы
func (s Set[T]) Values() []T { return nil }
