# Type Switches

## Обзор

Type switch — элегантный способ обработки нескольких типов через switch по типу значения интерфейса.

## Концепции

```go
func describe(i interface{}) string {
    switch v := i.(type) {
    case int:
        return fmt.Sprintf("int: %d", v)
    case string:
        return fmt.Sprintf("string: %q", v)
    case bool:
        return fmt.Sprintf("bool: %t", v)
    case []int:
        return fmt.Sprintf("[]int with %d elements", len(v))
    case nil:
        return "nil"
    default:
        return fmt.Sprintf("unknown: %T", v)
    }
}

// v в каждом case имеет соответствующий тип!
// case int: v — это int
// case string: v — это string
```

### Несколько типов в одном case

```go
switch v := i.(type) {
case int, int64, float64:
    // v имеет тип interface{} (не конкретный!)
    fmt.Printf("number: %v\n", v)
case string, []byte:
    fmt.Printf("text-like: %v\n", v)
}
```

### Type switch с интерфейсами

```go
type Shape interface { Area() float64 }
type Drawable interface { Draw() }

func process(s Shape) {
    switch s.(type) {
    case Drawable:
        // Shape, который также Drawable
        s.(Drawable).Draw()
    default:
        fmt.Printf("Area: %.2f\n", s.Area())
    }
}
```

### Реальный пример: обработка ошибок

```go
func handleError(err error) {
    switch e := err.(type) {
    case *os.PathError:
        log.Printf("path error: %s on %s", e.Op, e.Path)
    case *net.OpError:
        log.Printf("network error: %s", e.Op)
    case interface{ Timeout() bool }:
        if e.Timeout() {
            log.Println("timeout error")
        }
    default:
        log.Printf("unknown error: %v", err)
    }
}
```

## Частые вопросы на собеседованиях

**Q: Чем type switch отличается от цепочки type assertions?**
A: Type switch чище, безопаснее (нет паники), и компилятор может оптимизировать. Работает как единая конструкция.

**Q: Какой тип у переменной при нескольких типах в case?**
A: `interface{}` — конкретный тип не определён.

## Подводные камни

1. **Нельзя использовать fallthrough** в type switch — ошибка компиляции.
2. **Переменная `v` затеняет** внешнюю переменную `i` — будь внимателен с именованием.
