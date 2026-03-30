# Race Detector и Concurrency Testing

## Race Detector

```bash
# Включить race detector
go test -race ./...
go run -race main.go
go build -race -o myapp

# Race detector:
# - Инструментирует каждый memory access
# - Замедляет в 2-10x
# - Увеличивает потребление памяти в 5-10x
# - Находит data races в runtime (не статический анализ!)
# - ОБЯЗАТЕЛЬНО использовать в CI
```

```go
// Race detector найдёт:
func TestRace(t *testing.T) {
    counter := 0
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            counter++ // DATA RACE!
        }()
    }
    wg.Wait()
}
// WARNING: DATA RACE
// Write at 0x00c... by goroutine 8:
// Previous write at 0x00c... by goroutine 7:
```

## Тестирование конкурентного кода

```go
// Тест конкурентного доступа к cache
func TestConcurrentCache(t *testing.T) {
    cache := NewCache[string, int]()

    var wg sync.WaitGroup
    // Параллельные записи
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            cache.Set(fmt.Sprintf("key-%d", i), i)
        }(i)
    }
    // Параллельные чтения
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            cache.Get(fmt.Sprintf("key-%d", i))
        }(i)
    }
    wg.Wait()

    // Проверка (после всех горутин)
    for i := 0; i < 100; i++ {
        v, ok := cache.Get(fmt.Sprintf("key-%d", i))
        assert.True(t, ok)
        assert.Equal(t, i, v)
    }
}
```

## goleak — поиск утечек горутин

```go
import "go.uber.org/goleak"

func TestMain(m *testing.M) {
    goleak.VerifyTestMain(m) // проверяет что все горутины завершились
}

// Или для конкретного теста
func TestNoLeak(t *testing.T) {
    defer goleak.VerifyNone(t)

    ch := make(chan int)
    go func() {
        ch <- 42 // эта горутина утечёт если никто не читает из ch!
    }()
    // Если забыли <-ch → goleak поймает
    <-ch
}

// Игнорировать known горутины
func TestWithIgnore(t *testing.T) {
    defer goleak.VerifyNone(t,
        goleak.IgnoreTopFunction("net/http.(*Server).Serve"),
        goleak.IgnoreAnyFunction("database/sql.(*DB).connectionOpener"),
    )
}
```

## synctest (Go 1.25+ GA)

```go
// testing/synctest — тестирование кода с time и горутинами
// Без реального ожидания!

import "testing/synctest"

func TestTimeout(t *testing.T) {
    synctest.Run(func() {
        ch := make(chan string)

        go func() {
            time.Sleep(10 * time.Second) // НЕ ждёт реальные 10 секунд!
            ch <- "done"
        }()

        select {
        case result := <-ch:
            // synctest "перематывает" время
            assert.Equal(t, "done", result)
        case <-time.After(30 * time.Second):
            t.Fatal("timeout")
        }
    })
    // Тест выполняется мгновенно!
}

func TestTicker(t *testing.T) {
    synctest.Run(func() {
        var count atomic.Int32
        ticker := time.NewTicker(time.Second)
        defer ticker.Stop()

        go func() {
            for range ticker.C {
                count.Add(1)
                if count.Load() >= 5 {
                    return
                }
            }
        }()

        // Ждём "5 секунд" (мгновенно)
        time.Sleep(6 * time.Second)
        assert.Equal(t, int32(5), count.Load())
    })
}
```

## Паттерн: детерминистичный тест конкурентного кода

```go
// Используй каналы для синхронизации порядка в тесте
func TestWorkerPool(t *testing.T) {
    results := make(chan int, 10)
    pool := NewWorkerPool(3)

    // Отправляем 10 задач
    for i := 0; i < 10; i++ {
        i := i
        pool.Submit(func() {
            results <- i * 2
        })
    }

    // Собираем результаты (порядок не важен)
    pool.Shutdown()
    close(results)

    var got []int
    for r := range results {
        got = append(got, r)
    }

    assert.Len(t, got, 10)
    sort.Ints(got)
    expected := []int{0, 2, 4, 6, 8, 10, 12, 14, 16, 18}
    assert.Equal(t, expected, got)
}
```

## Частые вопросы

**Q: Race detector в production?**
A: Нет! Замедление 2-10x. Только в тестах и CI. Некоторые компании запускают canary с -race.

**Q: Race detector нашёл race — насколько это серьёзно?**
A: Очень серьёзно. В Go data race = undefined behavior. Может привести к corrupted data, crash, security vulnerability. Всегда исправляй.

**Q: synctest vs time mock?**
A: synctest — встроенный, работает с time.Sleep/After/Ticker нативно. Time mock (clock interface) — ручной подход, требует dependency injection. synctest проще и надёжнее.
