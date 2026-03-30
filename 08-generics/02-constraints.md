# Constraints

## Обзор

Constraint — интерфейс, ограничивающий допустимые типы для type parameter. Определяет, какие операции можно выполнять над значением.

## Концепции

### Базовые constraints

```go
// any — любой тип (= interface{})
func Print[T any](v T) { fmt.Println(v) }

// comparable — типы, поддерживающие ==
func Contains[T comparable](s []T, v T) bool { ... }

// Constraint из cmp пакета (Go 1.21+)
import "cmp"

func Min[T cmp.Ordered](a, b T) T {
    if a < b { return a }
    return b
}

// cmp.Ordered = interface{ int | float64 | string | ... | ~int | ~float64 | ... }
```

### Интерфейсы с type elements

```go
// Type union — допускает конкретные типы
type Number interface {
    int | int8 | int16 | int32 | int64 |
    float32 | float64
}

func Sum[T Number](nums []T) T {
    var total T
    for _, n := range nums {
        total += n // OK — все типы в union поддерживают +
    }
    return total
}

// Approximation (~) — допускает типы с underlying type
type Integer interface {
    ~int | ~int8 | ~int16 | ~int32 | ~int64
}

type UserID int64 // underlying type = int64
// UserID удовлетворяет Integer благодаря ~int64
```

### Constraint с методами

```go
type Stringer interface {
    String() string
}

func PrintAll[T Stringer](items []T) {
    for _, item := range items {
        fmt.Println(item.String())
    }
}

// Комбинация: типы И методы
type StringableInt interface {
    ~int | ~int64
    String() string
}
```

### constraints пакет (удалён в пользу cmp)

```go
// Раньше: golang.org/x/exp/constraints
// Теперь: cmp.Ordered (Go 1.21+) для сравнимых типов
// Для числовых: пиши свой constraint или используй community пакеты
```

## Частые вопросы на собеседованиях

**Q: Что делает тильда (~) в constraint?**
A: `~int` означает "любой тип с underlying type int". Без ~ — только exact type int. type UserID int не удовлетворяет `int`, но удовлетворяет `~int`.

**Q: Чем constraint отличается от обычного интерфейса?**
A: Constraint может содержать type elements (int | string). Такой интерфейс нельзя использовать как тип переменной — только как constraint.

**Q: Можно ли использовать constraint с type elements как тип переменной?**
A: Нет. `type Num interface { int | float64 }` — нельзя `var x Num`. Только `func F[T Num](x T)`.
