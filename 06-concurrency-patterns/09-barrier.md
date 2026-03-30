# Barrier

## Обзор

Барьер — точка синхронизации, где все горутины ждут друг друга перед продолжением. Похож на WaitGroup, но переиспользуемый.

## Концепции

```go
type Barrier struct {
    n       int
    count   int
    mu      sync.Mutex
    waitCh  chan struct{}
}

func NewBarrier(n int) *Barrier {
    return &Barrier{n: n, waitCh: make(chan struct{})}
}

func (b *Barrier) Wait() {
    b.mu.Lock()
    b.count++
    if b.count == b.n {
        // Все собрались — отпускаем
        close(b.waitCh)
        b.waitCh = make(chan struct{}) // для следующего цикла
        b.count = 0
        b.mu.Unlock()
        return
    }
    ch := b.waitCh
    b.mu.Unlock()
    <-ch // ждём остальных
}

// Использование: параллельные вычисления по фазам
barrier := NewBarrier(3)
for i := 0; i < 3; i++ {
    go func(id int) {
        // Фаза 1
        computePhase1(id)
        barrier.Wait() // все ждут завершения фазы 1

        // Фаза 2 (все данные фазы 1 готовы)
        computePhase2(id)
        barrier.Wait()
    }(i)
}
```

## Когда использовать

- Итеративные параллельные алгоритмы (симуляции, матричные вычисления)
- Фазовая синхронизация (все должны завершить фазу N перед фазой N+1)
- На практике в Go встречается редко — обычно хватает WaitGroup + каналов
