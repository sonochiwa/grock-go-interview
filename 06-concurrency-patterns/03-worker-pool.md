# Worker Pool

## Обзор

Worker Pool — фиксированное количество горутин, обрабатывающих задачи из общей очереди. Отличие от fan-out: воркеры создаются заранее и переиспользуются.

## Концепции

```go
func workerPool(ctx context.Context, numWorkers int, jobs <-chan Job) <-chan Result {
    results := make(chan Result)
    var wg sync.WaitGroup

    // Запускаем N воркеров
    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            for job := range jobs {
                select {
                case <-ctx.Done():
                    return
                case results <- processJob(job):
                }
            }
        }(i)
    }

    // Закрываем results когда все воркеры завершились
    go func() {
        wg.Wait()
        close(results)
    }()

    return results
}

// Использование
func main() {
    jobs := make(chan Job, 100) // буферизированный для backpressure
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    results := workerPool(ctx, 10, jobs)

    // Отправляем задачи
    go func() {
        defer close(jobs)
        for _, item := range items {
            jobs <- Job{Data: item}
        }
    }()

    // Получаем результаты
    for result := range results {
        fmt.Println(result)
    }
}
```

### Worker Pool vs Fan-Out

| Worker Pool | Fan-Out |
|---|---|
| Фиксированное кол-во горутин | Горутина на задачу |
| Общая очередь jobs | Каждый воркер — свой канал |
| Переиспользование горутин | Создание и завершение |
| Контроль ресурсов | Проще код |

### С graceful shutdown

```go
type Pool struct {
    jobs    chan Job
    results chan Result
    wg      sync.WaitGroup
}

func NewPool(numWorkers int) *Pool {
    p := &Pool{
        jobs:    make(chan Job, numWorkers*2),
        results: make(chan Result, numWorkers*2),
    }
    for i := 0; i < numWorkers; i++ {
        p.wg.Add(1)
        go p.worker()
    }
    return p
}

func (p *Pool) worker() {
    defer p.wg.Done()
    for job := range p.jobs {
        p.results <- processJob(job)
    }
}

func (p *Pool) Submit(job Job) { p.jobs <- job }

func (p *Pool) Shutdown() {
    close(p.jobs)   // воркеры завершатся после обработки оставшихся
    p.wg.Wait()     // ждём завершения
    close(p.results)
}
```

## Частые вопросы на собеседованиях

**Q: Зачем worker pool если можно просто go func()?**
A: Контроль ресурсов. 1M горутин = 2-8 ГБ RAM. Pool ограничивает параллелизм, даёт backpressure.

**Q: Как выбрать размер пула?**
A: I/O-bound: 10-100 (зависит от внешнего сервиса). CPU-bound: runtime.NumCPU(). Подбирать бенчмарками.
