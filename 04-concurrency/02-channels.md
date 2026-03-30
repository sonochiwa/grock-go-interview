# Каналы

## Обзор

Каналы — основной механизм коммуникации между горутинами. "Don't communicate by sharing memory; share memory by communicating."

## Концепции

### Создание и использование

```go
// Небуферизированный канал (synchronous)
ch := make(chan int)

// Буферизированный канал (async до заполнения буфера)
ch := make(chan int, 100)

// Отправка и получение
ch <- 42       // отправить
val := <-ch    // получить
val, ok := <-ch // ok=false если канал закрыт и пуст
```

### Unbuffered vs Buffered

```go
// Unbuffered: отправитель блокируется до получения
ch := make(chan int) // размер буфера = 0

go func() { ch <- 42 }() // блокируется до получения
val := <-ch               // разблокирует отправителя
// Гарантирует синхронизацию — отправка happens-before получение

// Buffered: отправитель блокируется только при полном буфере
ch := make(chan int, 3)
ch <- 1 // не блокируется (буфер: [1])
ch <- 2 // не блокируется (буфер: [1, 2])
ch <- 3 // не блокируется (буфер: [1, 2, 3])
ch <- 4 // БЛОКИРУЕТСЯ — буфер полон
```

### Directional channels

```go
// Ограничение направления (в сигнатуре функции)
func producer(out chan<- int) { // только отправка
    out <- 42
    // val := <-out // ОШИБКА компиляции
}

func consumer(in <-chan int) { // только получение
    val := <-in
    // in <- 42 // ОШИБКА компиляции
}

// Bidirectional → directional автоматически
ch := make(chan int)
go producer(ch) // chan int → chan<- int — OK
go consumer(ch) // chan int → <-chan int — OK
```

### Закрытие каналов

```go
ch := make(chan int, 3)
ch <- 1
ch <- 2
close(ch) // закрываем

// Можно читать оставшиеся значения
val, ok := <-ch // val=1, ok=true
val, ok = <-ch  // val=2, ok=true
val, ok = <-ch  // val=0, ok=false — канал закрыт и пуст

// range по каналу — читает до закрытия
for val := range ch {
    fmt.Println(val)
}
// Завершится когда канал закрыт и пуст
```

### Аксиомы каналов (ОБЯЗАТЕЛЬНО знать)

| Операция | nil channel | closed channel | open channel |
|---|---|---|---|
| **send** | блок навсегда | **PANIC** | блок или отправит |
| **receive** | блок навсегда | zero value, ok=false | блок или получит |
| **close** | **PANIC** | **PANIC** | OK |

```go
// nil channel — полезен в select для "отключения" case
var ch chan int // nil
// ch <- 1    // блокируется навсегда
// <-ch       // блокируется навсегда

// Паттерн: отключение канала в select
func merge(a, b <-chan int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for a != nil || b != nil {
            select {
            case v, ok := <-a:
                if !ok { a = nil; continue }
                out <- v
            case v, ok := <-b:
                if !ok { b = nil; continue }
                out <- v
            }
        }
    }()
    return out
}
```

### Закрытый канал как broadcast

```go
// close() разблокирует ВСЕ ожидающие горутины — это broadcast!
done := make(chan struct{})

// 100 горутин ждут сигнал
for i := 0; i < 100; i++ {
    go func() {
        <-done // все ждут
        fmt.Println("received signal")
    }()
}

close(done) // ВСЕ 100 горутин разблокированы одновременно
```

## Под капотом

Кратко (подробнее в 09-internals/05-channel-internals.md):

```go
// runtime/chan.go (упрощённо)
type hchan struct {
    qcount   uint           // текущее количество элементов
    dataqsiz uint           // размер буфера
    buf      unsafe.Pointer // ring buffer
    elemsize uint16
    closed   uint32
    sendx    uint            // индекс отправки в буфере
    recvx    uint            // индекс получения
    recvq    waitq           // очередь ожидающих получателей
    sendq    waitq           // очередь ожидающих отправителей
    lock     mutex           // мьютекс (НЕ sync.Mutex — внутренний)
}
```

## Частые вопросы на собеседованиях

**Q: Чем unbuffered канал отличается от buffered с размером 1?**
A: Unbuffered гарантирует синхронизацию: отправитель ждёт получателя. Buffered(1) позволяет отправить одно значение без блокировки.

**Q: Кто должен закрывать канал?**
A: Отправитель. Никогда не закрывай канал со стороны получателя — отправитель может паникнуть при записи в закрытый канал.

**Q: Что произойдёт при записи в закрытый канал?**
A: panic: send on closed channel.

**Q: Зачем нужен nil канал?**
A: В select nil канал "отключает" case — он никогда не выбирается. Полезно для merge/fan-in при закрытии одного из источников.

## Подводные камни

1. **Забыл закрыть канал** — горутины с `range ch` висят вечно.
2. **Закрытие канала дважды** — panic.
3. **Закрытие со стороны получателя** — отправитель паникнет.
4. **Deadlock** — все горутины заблокированы:
```go
func main() {
    ch := make(chan int)
    ch <- 1 // deadlock: main заблокирован, никто не читает
}
```
