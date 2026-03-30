# Жизненный цикл горутин

## Обзор

Каждая горутина должна иметь чёткий путь завершения. Если горутина не знает, когда остановиться — она утечёт.

## Паттерны завершения

### Done channel

```go
func worker(done <-chan struct{}, tasks <-chan int) {
    for {
        select {
        case <-done:
            fmt.Println("worker stopped")
            return
        case task, ok := <-tasks:
            if !ok {
                return // канал закрыт
            }
            process(task)
        }
    }
}

done := make(chan struct{})
go worker(done, tasks)
// ...
close(done) // остановить worker
```

### Context cancellation (предпочтительно)

```go
func worker(ctx context.Context, tasks <-chan int) {
    for {
        select {
        case <-ctx.Done():
            return
        case task := <-tasks:
            process(task)
        }
    }
}

ctx, cancel := context.WithCancel(context.Background())
go worker(ctx, tasks)
// ...
cancel() // остановить
```

### Graceful shutdown

```go
func main() {
    ctx, stop := signal.NotifyContext(context.Background(),
        syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    srv := &http.Server{Addr: ":8080"}

    // Запускаем сервер в горутине
    go func() {
        if err := srv.ListenAndServe(); err != http.ErrServerClosed {
            log.Fatal(err)
        }
    }()

    // Ждём сигнал
    <-ctx.Done()
    log.Println("shutting down...")

    // Даём 10 секунд на завершение текущих запросов
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    if err := srv.Shutdown(shutdownCtx); err != nil {
        log.Fatal(err)
    }
    log.Println("server stopped")
}
```

### Обнаружение утечек в тестах

```go
// uber-go/goleak — автоматическая проверка утечек горутин
import "go.uber.org/goleak"

func TestMain(m *testing.M) {
    goleak.VerifyTestMain(m)
}

// Или для отдельного теста
func TestSomething(t *testing.T) {
    defer goleak.VerifyNone(t)
    // ...
}
```

## Частые вопросы на собеседованиях

**Q: Как гарантировать завершение горутины?**
A: Context с cancel или done channel. WaitGroup для ожидания группы горутин.

**Q: Как реализовать graceful shutdown?**
A: signal.NotifyContext → ctx.Done() → http.Server.Shutdown с таймаутом.

## Подводные камни

1. **Горутина без механизма остановки** — утечка.
2. **Panic в горутине** — крашит всю программу, recover() работает только в той же горутине.
3. **os.Exit()** не вызывает defer — cleanup не выполняется.
