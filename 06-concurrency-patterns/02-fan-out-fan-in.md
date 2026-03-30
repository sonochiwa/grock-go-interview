# Fan-Out / Fan-In

## Обзор

**Fan-Out** — один канал распределяет работу нескольким горутинам.
**Fan-In** — несколько каналов сливаются в один.

Это ключевой паттерн для параллелизации обработки. Если ты работал только с CRUD — это первый паттерн, который нужно освоить.

## Визуализация

```
                ┌─ [Worker 1] ─┐
[Source] ──────►├─ [Worker 2] ─┤──► [Merge] ──► [Output]
                └─ [Worker 3] ─┘
     Fan-Out                   Fan-In
```

## Fan-Out

Несколько горутин читают из одного канала:

```go
func fanOut(ctx context.Context, input <-chan int, numWorkers int) []<-chan int {
    workers := make([]<-chan int, numWorkers)
    for i := 0; i < numWorkers; i++ {
        workers[i] = worker(ctx, input)
    }
    return workers
}

func worker(ctx context.Context, input <-chan int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for n := range input {
            select {
            case <-ctx.Done():
                return
            case out <- process(n): // тяжёлая обработка
            }
        }
    }()
    return out
}
```

**Когда fan-out эффективен:**
- I/O-bound задачи (HTTP запросы, БД, файлы) — всегда
- CPU-bound задачи — если numWorkers ≤ runtime.NumCPU()

## Fan-In

Слияние нескольких каналов в один:

```go
func fanIn(ctx context.Context, channels ...<-chan int) <-chan int {
    out := make(chan int)
    var wg sync.WaitGroup

    for _, ch := range channels {
        wg.Add(1)
        go func(c <-chan int) {
            defer wg.Done()
            for val := range c {
                select {
                case <-ctx.Done():
                    return
                case out <- val:
                }
            }
        }(ch)
    }

    // Закрываем out когда все входные каналы закрыты
    go func() {
        wg.Wait()
        close(out)
    }()

    return out
}
```

## Полный пример: Fan-Out/Fan-In

```go
func processURLs(ctx context.Context, urls []string) ([]Result, error) {
    // Стадия 1: Генерация задач
    input := make(chan string)
    go func() {
        defer close(input)
        for _, url := range urls {
            select {
            case <-ctx.Done():
                return
            case input <- url:
            }
        }
    }()

    // Стадия 2: Fan-Out — запускаем N воркеров
    numWorkers := 10
    workers := make([]<-chan Result, numWorkers)
    for i := 0; i < numWorkers; i++ {
        workers[i] = fetchWorker(ctx, input)
    }

    // Стадия 3: Fan-In — собираем результаты
    var results []Result
    for result := range merge(ctx, workers...) {
        results = append(results, result)
    }

    return results, ctx.Err()
}

type Result struct {
    URL    string
    Status int
    Err    error
}

func fetchWorker(ctx context.Context, urls <-chan string) <-chan Result {
    out := make(chan Result)
    go func() {
        defer close(out)
        for url := range urls {
            req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
            resp, err := http.DefaultClient.Do(req)
            result := Result{URL: url}
            if err != nil {
                result.Err = err
            } else {
                result.Status = resp.StatusCode
                resp.Body.Close()
            }
            select {
            case <-ctx.Done():
                return
            case out <- result:
            }
        }
    }()
    return out
}

func merge(ctx context.Context, channels ...<-chan Result) <-chan Result {
    out := make(chan Result)
    var wg sync.WaitGroup

    for _, ch := range channels {
        wg.Add(1)
        go func(c <-chan Result) {
            defer wg.Done()
            for val := range c {
                select {
                case <-ctx.Done():
                    return
                case out <- val:
                }
            }
        }(ch)
    }

    go func() {
        wg.Wait()
        close(out)
    }()
    return out
}
```

## Fan-Out/Fan-In с errgroup (проще)

```go
func processURLs(ctx context.Context, urls []string) ([]Result, error) {
    results := make([]Result, len(urls))
    g, ctx := errgroup.WithContext(ctx)
    g.SetLimit(10) // Fan-Out с ограничением

    for i, url := range urls {
        i, url := i, url
        g.Go(func() error {
            resp, err := http.Get(url)
            if err != nil {
                return err
            }
            defer resp.Body.Close()
            results[i] = Result{URL: url, Status: resp.StatusCode}
            return nil
        })
    }

    if err := g.Wait(); err != nil { // Fan-In (ожидание)
        return nil, err
    }
    return results, nil
}
```

## Частые вопросы на собеседованиях

**Q: Что такое Fan-Out?**
A: Распределение работы из одного источника нескольким воркерам. Несколько горутин читают из одного канала.

**Q: Что такое Fan-In?**
A: Слияние результатов из нескольких каналов в один. WaitGroup + горутина для каждого канала.

**Q: Сколько воркеров запускать?**
A: I/O-bound: десятки-сотни (ограничивает внешняя система). CPU-bound: runtime.NumCPU(). На практике — benchmark.

**Q: Как обработать ошибки в fan-out/fan-in?**
A: Через errgroup (первая ошибка отменяет всё) или через Result struct с полем Err (собираем все ошибки).

## Подводные камни

1. **Забыл close(out) в fan-in** — горутина-потребитель висит навсегда в range.
2. **Забыл ctx.Done() в select** — горутины не останавливаются при отмене.
3. **Слишком много воркеров** для внешнего сервиса — DDoS собственного API.
4. **Порядок не гарантирован** — результаты приходят в произвольном порядке. Если нужен порядок — индексируй.
