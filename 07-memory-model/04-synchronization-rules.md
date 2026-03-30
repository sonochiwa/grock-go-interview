# Правила синхронизации

## Обзор

Что является sync point (точкой синхронизации) в Go и какие гарантии каждый примитив даёт.

## Sync Points

### Каналы
```go
// Send happens-before receive completes
ch <- x   // HB
v := <-ch // видит x

// Close happens-before receive of zero value
close(ch) // HB
<-ch      // видит все записи до close

// Для unbuffered: receive happens-before send completes
<-ch     // HB
ch <- x  // может использовать данные, модифицированные получателем
```

### Mutex
```go
mu.Lock()   // n-й Lock happens-after (n-1)-й Unlock
// критическая секция
mu.Unlock() // HB следующий Lock
```

### Once
```go
once.Do(f) // вызов f happens-before ЛЮБОЙ Do возвращает
// Все горутины видят эффекты f
```

### Atomic
```go
// С Go 1.19: atomic = sequential consistency
// Все atomic операции наблюдаются всеми горутинами в одном порядке
atomic.StoreInt64(&x, 1) // HB
atomic.LoadInt64(&x)     // видит 1
```

### WaitGroup
```go
wg.Done()  // HB
wg.Wait()  // возвращается после всех Done
```

### Goroutine creation
```go
x = 1
go f()  // запуск горутины HB начало f()
// f() видит x == 1
```

### Init
```go
// import pkg → pkg.init() HB main.init() HB main()
```

## Сводная таблица

| Операция A | Операция B | A happens-before B? |
|---|---|---|
| ch <- x | <-ch (same ch) | Да |
| close(ch) | <-ch returns 0 | Да |
| mu.Unlock() | mu.Lock() (next) | Да |
| wg.Done() | wg.Wait() returns | Да |
| go f() | start of f() | Да |
| atomic.Store | atomic.Load (seq consistent) | Да |
| x = 1 | y = x (другая горутина) | **НЕТ** без sync! |
