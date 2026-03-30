# Generic Patterns

## Практические паттерны

### Коллекции

```go
// Filter
func Filter[T any](s []T, pred func(T) bool) []T {
    var result []T
    for _, v := range s {
        if pred(v) { result = append(result, v) }
    }
    return result
}

// Reduce
func Reduce[T any, R any](s []T, init R, f func(R, T) R) R {
    result := init
    for _, v := range s {
        result = f(result, v)
    }
    return result
}

// Keys/Values из map
func Keys[K comparable, V any](m map[K]V) []K {
    keys := make([]K, 0, len(m))
    for k := range m { keys = append(keys, k) }
    return keys
}
```

### Result type

```go
type Result[T any] struct {
    Value T
    Err   error
}

func (r Result[T]) Unwrap() (T, error) {
    return r.Value, r.Err
}

func OK[T any](v T) Result[T] {
    return Result[T]{Value: v}
}

func Fail[T any](err error) Result[T] {
    return Result[T]{Err: err}
}
```

### Set

```go
type Set[T comparable] map[T]struct{}

func NewSet[T comparable](items ...T) Set[T] {
    s := make(Set[T], len(items))
    for _, item := range items { s[item] = struct{}{} }
    return s
}

func (s Set[T]) Contains(v T) bool { _, ok := s[v]; return ok }
func (s Set[T]) Add(v T)          { s[v] = struct{}{} }
func (s Set[T]) Remove(v T)       { delete(s, v) }
```

### Используй стандартную библиотеку (Go 1.21+)

```go
import "slices"
slices.Contains(s, v)
slices.Sort(s)
slices.SortFunc(s, cmp)
slices.Index(s, v)
slices.Compact(s) // удалить дубликаты (сортированный)

import "maps"
maps.Keys(m)
maps.Values(m)
maps.Clone(m)
maps.Equal(m1, m2)
```
