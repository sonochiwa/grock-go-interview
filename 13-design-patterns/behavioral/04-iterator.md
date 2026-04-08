# Iterator

## В Go

До Go 1.23: `for range` работал только с массивами, слайсами, мапами, каналами, строками.
С Go 1.23: **range-over-func** — можно итерировать по любой функции.

```go
// Range-over-func (Go 1.23+)
// iter.Seq[V] = func(yield func(V) bool)
// iter.Seq2[K, V] = func(yield func(K, V) bool)

func Fibonacci() iter.Seq[int] {
    return func(yield func(int) bool) {
        a, b := 0, 1
        for {
            if !yield(a) {
                return // break вызван
            }
            a, b = b, a+b
        }
    }
}

// Использование
for n := range Fibonacci() {
    if n > 100 {
        break // yield вернёт false
    }
    fmt.Println(n)
}

// Пример с key-value
func Enumerate[T any](s []T) iter.Seq2[int, T] {
    return func(yield func(int, T) bool) {
        for i, v := range s {
            if !yield(i, v) {
                return
            }
        }
    }
}

for i, v := range Enumerate([]string{"a", "b", "c"}) {
    fmt.Println(i, v)
}
```

### Стандартная библиотека

```go
import "slices"
// slices.All — итератор для слайса
// slices.Values — значения слайса
// slices.Backward — обратный порядок

import "maps"
// maps.Keys — ключи мапы
// maps.Values — значения

for k := range maps.Keys(m) { ... }
for v := range slices.Backward(s) { ... }
```
