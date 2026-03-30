# sync.WaitGroup

## Обзор

WaitGroup ждёт завершения группы горутин. Add(n) увеличивает счётчик, Done() уменьшает, Wait() блокирует до нуля.

## Концепции

```go
var wg sync.WaitGroup

for i := 0; i < 10; i++ {
    wg.Add(1) // ПЕРЕД запуском горутины!
    go func(id int) {
        defer wg.Done()
        process(id)
    }(i)
}

wg.Wait() // ждём все 10 горутин
```

### Типичные ошибки

```go
// ОШИБКА 1: Add внутри горутины
for i := 0; i < 10; i++ {
    go func() {
        wg.Add(1) // ❌ race: Wait() может вернуться до Add
        defer wg.Done()
    }()
}
wg.Wait()

// ОШИБКА 2: Забыл Done()
wg.Add(1)
go func() {
    // ❌ нет Done() — Wait() ждёт вечно
    process()
}()

// ОШИБКА 3: Отрицательный счётчик
wg.Add(1)
wg.Done()
wg.Done() // panic: negative WaitGroup counter
```

## Частые вопросы на собеседованиях

**Q: Почему Add вызывается до go, а не внутри горутины?**
A: Wait() может выполниться до Add() внутри горутины — race condition.

**Q: Можно ли переиспользовать WaitGroup?**
A: Да, после Wait() можно снова вызвать Add(). Но не вызывай Add() после Wait() начал ждать.
