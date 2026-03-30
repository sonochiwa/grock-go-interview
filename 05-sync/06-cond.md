# sync.Cond

## Обзор

Cond — условная переменная. Позволяет горутинам ждать наступления условия. На практике используется редко — каналы обычно удобнее.

## Концепции

```go
type Queue struct {
    mu    sync.Mutex
    cond  *sync.Cond
    items []int
}

func NewQueue() *Queue {
    q := &Queue{}
    q.cond = sync.NewCond(&q.mu)
    return q
}

func (q *Queue) Push(item int) {
    q.mu.Lock()
    q.items = append(q.items, item)
    q.mu.Unlock()
    q.cond.Signal() // разбудить одного ожидающего
}

func (q *Queue) Pop() int {
    q.mu.Lock()
    defer q.mu.Unlock()
    for len(q.items) == 0 { // ОБЯЗАТЕЛЬНО цикл, не if!
        q.cond.Wait() // атомарно: unlock → sleep → lock при пробуждении
    }
    item := q.items[0]
    q.items = q.items[1:]
    return item
}
```

### Signal vs Broadcast

- `Signal()` — разбудить одного (для producer-consumer)
- `Broadcast()` — разбудить всех (для изменения состояния)

### Почему цикл, а не if?

```go
// Spurious wakeup: горутина может проснуться без Signal
// Другая горутина могла забрать элемент между Signal и Lock
for !condition() {
    cond.Wait()
}
```

## Когда Cond лучше каналов

- Broadcast нескольким ожидающим (канал может уведомить одного)
- Сложные условия ожидания (не просто "есть данные")
- Нет передачи данных, только уведомление

## Частые вопросы на собеседованиях

**Q: Почему Wait() вызывается в цикле?**
A: Spurious wakeups и race condition между Signal и Lock.

**Q: Когда использовать Cond вместо канала?**
A: Broadcast нескольким горутинам, сложные условия ожидания. Но в большинстве случаев каналы проще.
