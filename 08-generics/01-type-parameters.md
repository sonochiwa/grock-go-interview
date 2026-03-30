# Type Parameters

## Обзор

Type parameters позволяют функциям и типам работать с любым типом, удовлетворяющим constraint.

## Концепции

### Функции

```go
// Без дженериков
func ContainsInt(s []int, v int) bool { ... }
func ContainsString(s []string, v string) bool { ... }

// С дженериками (одна функция для всех типов)
func Contains[T comparable](s []T, v T) bool {
    for _, item := range s {
        if item == v {
            return true
        }
    }
    return false
}

Contains([]int{1, 2, 3}, 2)         // true — T = int (выведен)
Contains([]string{"a", "b"}, "c")    // false — T = string
Contains[int]([]int{1, 2, 3}, 2)     // явная инстанциация
```

### Типы

```go
// Generic стек
type Stack[T any] struct {
    items []T
}

func (s *Stack[T]) Push(v T) {
    s.items = append(s.items, v)
}

func (s *Stack[T]) Pop() (T, bool) {
    if len(s.items) == 0 {
        var zero T
        return zero, false
    }
    v := s.items[len(s.items)-1]
    s.items = s.items[:len(s.items)-1]
    return v, true
}

intStack := Stack[int]{}
intStack.Push(42)

strStack := Stack[string]{}
strStack.Push("hello")
```

### Множественные type parameters

```go
func Map[T any, R any](s []T, f func(T) R) []R {
    result := make([]R, len(s))
    for i, v := range s {
        result[i] = f(v)
    }
    return result
}

lengths := Map([]string{"go", "rust"}, func(s string) int { return len(s) })
// [2, 4]
```

### Zero value

```go
func Zero[T any]() T {
    var zero T
    return zero // zero value для любого типа
}

Zero[int]()    // 0
Zero[string]() // ""
Zero[*int]()   // nil
```

## Под капотом: монорфизация vs GCShape stenciling

Go использует **GCShape stenciling** — компромисс между:
- Полной монорфизацией (C++: копия кода для каждого типа → быстро, но большой бинарник)
- Boxing (Java: всё через интерфейс → медленно)

Go создаёт одну копию кода для каждой "GC shape":
- Все типы-указатели делят одну копию
- Каждый value type уникального размера — своя копия
- Dispatch через словарь (dict) для методов

## Частые вопросы на собеседованиях

**Q: Когда использовать дженерики, а когда интерфейсы?**
A: Дженерики — когда нужна типобезопасность без boxing (коллекции, утилиты). Интерфейсы — когда полиморфизм по поведению (io.Reader, разные стратегии).

**Q: Как получить zero value generic типа?**
A: `var zero T` или `*new(T)`.
