# Горутины

## Обзор

Горутина — легковесный поток выполнения, управляемый Go runtime. Создание горутины стоит ~2-4 КБ памяти (стек), в отличие от OS потока (~1-8 МБ).

## Концепции

### Создание

```go
// Запуск горутины
go doWork()

// С анонимной функцией
go func() {
    fmt.Println("running in goroutine")
}()

// С аргументами
go func(name string) {
    fmt.Println("Hello,", name)
}("Alice")

// ВАЖНО: main() не ждёт завершения горутин!
func main() {
    go fmt.Println("hello")
    // программа может завершиться ДО печати
}
```

### Горутина vs OS Thread

| Горутина | OS Thread |
|---|---|
| 2-8 КБ начальный стек | 1-8 МБ стек |
| Управляется Go runtime | Управляется ОС |
| Кооперативное + вытесняющее (1.14+) | Вытесняющее |
| Создание ~0.3 мкс | Создание ~10+ мкс |
| Можно миллионы одновременно | Тысячи — предел |
| Мультиплексируются на M OS потоков | 1:1 с ядром |

### Стек горутины

```go
// Начальный стек: 2-8 КБ (зависит от версии Go)
// Стек РАСТЁТ динамически (до ~1 ГБ по умолчанию)

// Механизм роста (contiguous stacks, с Go 1.4):
// 1. Функция проверяет: достаточно ли стека?
// 2. Если нет — runtime выделяет стек в 2x больше
// 3. Копирует старый стек в новый
// 4. Обновляет все указатели на стеке
// 5. Освобождает старый стек
```

### Scheduling (планирование)

Go использует модель **GMP** (подробнее в 09-internals/07-scheduler.md):

- **G** (Goroutine) — горутина
- **M** (Machine) — OS thread
- **P** (Processor) — логический процессор (по умолчанию = кол-во CPU)

```go
// Количество P = GOMAXPROCS
runtime.GOMAXPROCS(0)  // вернёт текущее значение (не меняя)
runtime.GOMAXPROCS(4)  // установить 4 (обычно не нужно менять)

// Количество горутин
runtime.NumGoroutine()

// Уступить процессор другим горутинам
runtime.Gosched() // кооперативный yield (редко нужен)
```

### Вытесняющее планирование (Go 1.14+)

```go
// До Go 1.14: горутина без системных вызовов и вызовов функций
// могла заблокировать P навсегда (tight loop)
go func() {
    for { /* бесконечный цикл без вызовов */ }
}()
// Другие горутины на этом P заблокированы!

// С Go 1.14: runtime использует OS сигналы (SIGURG) для
// прерывания горутин. Tight loop больше не блокирует.
// Проверка — при входе в функцию (stack check) + async preemption
```

## Утечки горутин

```go
// 1. Забытый канал — горутина ждёт вечно
func leak() {
    ch := make(chan int)
    go func() {
        val := <-ch // никто не отправит — горутина висит навсегда
        fmt.Println(val)
    }()
    // return — ch и горутина утекли
}

// 2. Горутина без выхода
func processForever(input <-chan Task) {
    go func() {
        for task := range input {
            process(task)
        }
    }()
    // Если input никогда не закроют — горутина вечная
}

// 3. Блокировка на HTTP/DB без таймаута
go func() {
    resp, _ := http.Get("http://slow-server.com") // может висеть минуты
    _ = resp
}()

// Решение: ВСЕГДА используй context с таймаутом
go func() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    resp, err := http.DefaultClient.Do(req)
    // ...
}()
```

### Обнаружение утечек

```go
// В тестах — проверяй runtime.NumGoroutine()
func TestNoLeak(t *testing.T) {
    before := runtime.NumGoroutine()
    doWork()
    time.Sleep(100 * time.Millisecond) // дать горутинам завершиться
    after := runtime.NumGoroutine()
    if after > before {
        t.Errorf("goroutine leak: before=%d after=%d", before, after)
    }
}

// Лучше: uber-go/goleak
import "go.uber.org/goleak"

func TestMain(m *testing.M) {
    goleak.VerifyTestMain(m)
}
```

## Частые вопросы на собеседованиях

**Q: Чем горутина отличается от потока?**
A: Легковесная (2-8 КБ vs 1-8 МБ стек), управляется Go runtime (не ОС), мультиплексируется на OS потоки (M:N модель). Можно создать миллионы.

**Q: Что такое GOMAXPROCS?**
A: Максимальное количество OS потоков, одновременно выполняющих Go-код. По умолчанию = количество CPU ядер. Контролирует параллелизм.

**Q: Как обнаружить утечку горутин?**
A: runtime.NumGoroutine() в тестах, goleak библиотека, pprof goroutine profile.

**Q: Что изменилось в планировании с Go 1.14?**
A: Добавлено вытесняющее планирование через async preemption (OS сигналы). До 1.14 tight loop мог заблокировать P.

## Подводные камни

1. **Не ждёшь горутину** — main() или функция завершается раньше горутины. Используй WaitGroup, канал или context.

2. **Горутина захватывает переменную цикла** — исправлено в Go 1.22, но знай историю (см. 01-fundamentals/07-functions.md).

3. **Слишком много горутин** — миллион горутин = ~2-8 ГБ стеков. Используй worker pool для ограничения.
