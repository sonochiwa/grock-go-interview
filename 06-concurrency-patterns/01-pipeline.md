# Pipeline

## Обзор

Pipeline — цепочка стадий обработки, соединённых каналами. Каждая стадия — горутина, которая читает из входного канала и пишет в выходной.

## Концепции

```
[Generator] → chan → [Square] → chan → [Print]
```

```go
// Стадия 1: генератор
func generate(nums ...int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for _, n := range nums {
            out <- n
        }
    }()
    return out
}

// Стадия 2: обработка
func square(in <-chan int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for n := range in {
            out <- n * n
        }
    }()
    return out
}

// Сборка pipeline
func main() {
    ch := generate(2, 3, 4)
    result := square(ch)

    for v := range result {
        fmt.Println(v) // 4, 9, 16
    }
}

// Можно чейнить: square(square(generate(2, 3)))
```

### Pipeline с контекстом (отмена)

```go
func generate(ctx context.Context, nums ...int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for _, n := range nums {
            select {
            case <-ctx.Done():
                return
            case out <- n:
            }
        }
    }()
    return out
}

func square(ctx context.Context, in <-chan int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for n := range in {
            select {
            case <-ctx.Done():
                return
            case out <- n * n:
            }
        }
    }()
    return out
}

// Отмена
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

pipeline := square(ctx, generate(ctx, 2, 3, 4, 5, 6))
fmt.Println(<-pipeline) // 4
cancel() // все стадии завершатся
```

### Pipeline с ошибками

```go
type Result struct {
    Value int
    Err   error
}

func process(ctx context.Context, in <-chan int) <-chan Result {
    out := make(chan Result)
    go func() {
        defer close(out)
        for n := range in {
            result, err := expensiveOp(n)
            select {
            case <-ctx.Done():
                return
            case out <- Result{result, err}:
            }
        }
    }()
    return out
}
```

## Когда использовать

- Поточная обработка (ETL, data transformation)
- Когда каждая стадия независима
- Когда данные проходят через серию трансформаций

## Частые вопросы на собеседованиях

**Q: Как предотвратить утечку горутин в pipeline?**
A: Context с cancel. Каждая стадия проверяет ctx.Done() в select. cancel() завершает все стадии.

**Q: Кто закрывает каналы в pipeline?**
A: Каждая стадия закрывает свой выходной канал (defer close(out)).
