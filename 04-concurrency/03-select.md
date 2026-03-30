# Select

## Обзор

select — мультиплексор для каналов. Позволяет ждать несколько каналов одновременно. Один из ключевых инструментов конкурентного Go.

## Концепции

### Базовый синтаксис

```go
select {
case msg := <-ch1:
    fmt.Println("from ch1:", msg)
case msg := <-ch2:
    fmt.Println("from ch2:", msg)
case ch3 <- 42:
    fmt.Println("sent to ch3")
}
// Блокируется, пока один из case не станет готов
// Если несколько готовы — выбирается СЛУЧАЙНО
```

### default case (non-blocking)

```go
select {
case msg := <-ch:
    process(msg)
default:
    // Выполняется если ни один case не готов
    fmt.Println("no message available")
}

// Non-blocking send
select {
case ch <- msg:
    // отправлено
default:
    // канал полон или нет получателя — пропускаем
    log.Println("message dropped")
}
```

### Timeout

```go
select {
case result := <-ch:
    fmt.Println("got result:", result)
case <-time.After(3 * time.Second):
    fmt.Println("timeout!")
}

// ОСТОРОЖНО: time.After в цикле утекает!
// Каждый вызов создаёт новый таймер, который не GC'd до срабатывания
for {
    select {
    case msg := <-ch:
        process(msg)
    case <-time.After(time.Second): // ❌ утечка таймеров!
        return
    }
}

// ПРАВИЛЬНО: time.NewTimer + Reset
timer := time.NewTimer(time.Second)
defer timer.Stop()
for {
    select {
    case msg := <-ch:
        process(msg)
        if !timer.Stop() {
            <-timer.C
        }
        timer.Reset(time.Second)
    case <-timer.C:
        return
    }
}
```

### Done channel pattern

```go
func worker(done <-chan struct{}, tasks <-chan Task) {
    for {
        select {
        case <-done:
            return // сигнал остановки
        case task := <-tasks:
            process(task)
        }
    }
}

done := make(chan struct{})
go worker(done, tasks)
// ...
close(done) // останавливает worker
```

### Empty select (block forever)

```go
select {} // блокирует горутину навсегда
// Полезно в main() когда вся работа в горутинах
```

### for-select loop

```go
// Типичный паттерн — обработка событий в цикле
func eventLoop(ctx context.Context, events <-chan Event) {
    for {
        select {
        case <-ctx.Done():
            log.Println("shutting down:", ctx.Err())
            return
        case event, ok := <-events:
            if !ok {
                return // канал закрыт
            }
            handle(event)
        }
    }
}
```

## Частые вопросы на собеседованиях

**Q: Что произойдёт если несколько case готовы одновременно?**
A: Выбирается один случайно (uniform random). Это intentional — предотвращает starvation.

**Q: Чем select с default отличается от без?**
A: Без default — блокирующий (ждёт пока case станет готов). С default — non-blocking (выполняет default если ничего не готово).

**Q: Как сделать таймаут?**
A: `case <-time.After(duration)` или лучше `context.WithTimeout`.

**Q: Что делает пустой select{}?**
A: Блокирует горутину навсегда. Используется когда вся работа в фоновых горутинах.

## Подводные камни

1. **time.After в цикле** утекает — каждый вызов создаёт новый таймер. Используй time.NewTimer + Reset.
2. **Приоритет case** — нет приоритетов, выбор случайный. Если нужен приоритет — используй вложенные select.
3. **Забытый break** — break в select выходит из select, не из for. Для выхода из for-select используй return, goto или labeled break.
4. **nil канал в select** — nil канал никогда не выбирается. Полезно для "отключения" case:
```go
var ch1, ch2 <-chan int = activeChannel, anotherChannel
for ch1 != nil || ch2 != nil {
    select {
    case v, ok := <-ch1:
        if !ok { ch1 = nil; continue } // "отключаем"
        process(v)
    case v, ok := <-ch2:
        if !ok { ch2 = nil; continue }
        process(v)
    }
}
```
