# Type Inference

## Обзор

Go может вывести type arguments из аргументов функции, избавляя от явного указания типов.

## Концепции

```go
// Вывод из аргументов
func Max[T cmp.Ordered](a, b T) T { if a > b { return a }; return b }

Max(1, 2)       // T = int (выведен из аргументов)
Max("a", "b")   // T = string
Max[float64](1, 2) // явное указание

// Вывод НЕ работает для:
// 1. Возвращаемых типов
func Zero[T any]() T { var z T; return z }
Zero()    // ОШИБКА: не может вывести T
Zero[int]() // OK

// 2. Когда типы неоднозначны
func Convert[From, To any](v From) To { ... }
Convert(42) // ОШИБКА: не может вывести To
Convert[int, float64](42) // OK
```

## Частые вопросы на собеседованиях

**Q: Когда нужно указывать type arguments явно?**
A: Когда тип нельзя вывести из аргументов: zero-arg функции, тип только в возврате, неоднозначные случаи.
