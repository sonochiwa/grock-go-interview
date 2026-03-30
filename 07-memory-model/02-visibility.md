# Visibility

## Обзор

Запись в переменную в одной горутине не обязательно видна другой горутине без синхронизации.

## Концепции

```go
var done bool
var msg string

go func() {
    msg = "hello"
    done = true
}()

// НЕПРАВИЛЬНО: нет гарантии видимости
for !done {
    runtime.Gosched()
}
fmt.Println(msg) // может напечатать "" или "hello" или зависнуть навсегда!

// Почему может зависнуть: компилятор может закешировать done в регистре
// Почему msg может быть пустым: CPU может переупорядочить записи
```

### Правильные способы обеспечить visibility

```go
// 1. Канал
ch := make(chan struct{})
go func() {
    msg = "hello"
    close(ch)  // happens-before
}()
<-ch
fmt.Println(msg) // гарантированно "hello"

// 2. Mutex
var mu sync.Mutex
go func() {
    mu.Lock()
    msg = "hello"
    mu.Unlock()
}()
mu.Lock()
fmt.Println(msg)
mu.Unlock()

// 3. Atomic
var ready atomic.Bool
go func() {
    msg = "hello"
    ready.Store(true) // sequential consistency
}()
for !ready.Load() {}
fmt.Println(msg) // гарантированно "hello"
```

## Частые вопросы на собеседованиях

**Q: Почему `for !done {}` может зависнуть?**
A: Компилятор может оптимизировать цикл, закешировав done в регистре. Без sync point — нет гарантии видимости обновления из другой горутины.

**Q: Достаточно ли volatile в Go?**
A: В Go нет volatile. Используй atomic или sync примитивы.
