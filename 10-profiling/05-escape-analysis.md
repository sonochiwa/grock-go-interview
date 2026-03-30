# Escape Analysis

## Обзор

Escape analysis — анализ компилятора, определяющий, где разместить переменную: на стеке (быстро) или в куче (GC).

## Проверка

```bash
go build -gcflags="-m" .
# ./main.go:10:6: moved to heap: x
# ./main.go:15:12: ... escapes to heap

# Больше деталей
go build -gcflags="-m -m" .
```

## Что вызывает escape

```go
// 1. Возврат указателя
func newInt() *int {
    x := 42
    return &x // x escapes to heap
}

// 2. Присвоение интерфейсу
func print(v any) { ... }
x := 42
print(x) // x escapes (упаковка в interface{})

// 3. Замыкание
func closure() func() int {
    x := 0
    return func() int { x++; return x } // x escapes
}

// 4. Слишком большой объект для стека
big := make([]byte, 1<<20) // 1MB — escape

// 5. Отправка в канал
ch <- &myStruct{} // escapes

// 6. Slice append за пределы cap
s := make([]int, 0, 10)
s = append(s, bigSlice...) // может escape
```

## Оптимизация

```go
// Передавай значения вместо указателей (для маленьких структур)
func process(p Point) {}   // стек
func process(p *Point) {}  // p может escape

// Предаллоцируй слайсы
s := make([]int, 0, n) // подсказка компилятору

// Используй массивы вместо слайсов для фиксированного размера
var buf [64]byte // стек
buf2 := make([]byte, 64) // может escape
```

## Частые вопросы на собеседованиях

**Q: Как узнать, что переменная ушла в кучу?**
A: `go build -gcflags="-m"`. Компилятор покажет "escapes to heap".

**Q: Почему стек быстрее кучи?**
A: Стек — простое смещение указателя (O(1)). Куча — поиск свободного блока + GC должен отслеживать и освобождать.
