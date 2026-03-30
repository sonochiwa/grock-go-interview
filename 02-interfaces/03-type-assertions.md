# Type Assertions

## Обзор

Type assertion извлекает конкретный тип из интерфейса. Два варианта: с проверкой (comma-ok) и без (паникует при несоответствии).

## Концепции

```go
var i interface{} = "hello"

// Без проверки — паникует при ошибке
s := i.(string)    // "hello"
// n := i.(int)    // panic: interface conversion: interface {} is string, not int

// С проверкой (comma-ok) — безопасно
s, ok := i.(string) // s="hello", ok=true
n, ok := i.(int)    // n=0, ok=false

// Идиоматичный паттерн
if s, ok := i.(string); ok {
    fmt.Println("String:", s)
}
```

### Assertion к интерфейсу

```go
type Reader interface { Read([]byte) (int, error) }

var w io.Writer = os.Stdout

// Проверяем, реализует ли Writer также Reader
if r, ok := w.(Reader); ok {
    // os.Stdout реализует и Writer, и Reader
    r.Read(buf)
}

// Полезный паттерн: опциональные интерфейсы
type Flusher interface { Flush() error }

func writeData(w io.Writer, data []byte) error {
    _, err := w.Write(data)
    if err != nil {
        return err
    }
    // Flush если поддерживается
    if f, ok := w.(Flusher); ok {
        return f.Flush()
    }
    return nil
}
```

## Под капотом

Type assertion проверяет поле `_type` в структуре интерфейса. Для assertion к интерфейсу — ищет itab (кешируется после первого поиска).

Стоимость:
- Assertion к конкретному типу: сравнение одного указателя — O(1), очень быстро
- Assertion к интерфейсу: поиск itab — O(n) при первом вызове, далее кеш

## Частые вопросы на собеседованиях

**Q: Чем отличается `x.(T)` от `x.(T)` с comma-ok?**
A: Без comma-ok паникует при несоответствии типа. С comma-ok возвращает zero value и false.

**Q: Можно ли делать type assertion на не-интерфейсном типе?**
A: Нет. Type assertion работает только с интерфейсными значениями: `var x string; x.(int)` — ошибка компиляции.

## Подводные камни

1. **Panic без comma-ok** — всегда используй comma-ok в продакшн-коде если тип не гарантирован.

2. **nil interface** — type assertion на nil интерфейсе всегда паникует (без comma-ok) или возвращает false:
```go
var i interface{} // nil
_, ok := i.(string) // ok=false
// i.(string) // panic
```
