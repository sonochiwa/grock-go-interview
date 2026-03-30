# Функции

## Обзор

Функции в Go — first-class citizens: их можно присваивать переменным, передавать как аргументы, возвращать из других функций. Замыкания, defer, init() — всё это спрашивают на собесах.

## Концепции

### Базовый синтаксис

```go
// Обычная функция
func add(a, b int) int {
    return a + b
}

// Множественные возвращаемые значения
func divide(a, b float64) (float64, error) {
    if b == 0 {
        return 0, errors.New("division by zero")
    }
    return a / b, nil
}

// Именованные возвращаемые значения
func divide(a, b float64) (result float64, err error) {
    if b == 0 {
        err = errors.New("division by zero")
        return // "голый" return — возвращает именованные значения
    }
    result = a / b
    return
}
```

### Variadic функции

```go
func sum(nums ...int) int {
    total := 0
    for _, n := range nums {
        total += n
    }
    return total
}

sum(1, 2, 3)           // 6
sum([]int{1, 2, 3}...) // 6 — распаковка слайса

// nums внутри функции — это []int
// Если не передано аргументов — nums == nil (не пустой слайс)
```

### Замыкания (closures)

Замыкание — функция, захватывающая переменные из окружающего scope:

```go
func counter() func() int {
    count := 0
    return func() int {
        count++ // замыкание "захватывает" count
        return count
    }
}

c := counter()
c() // 1
c() // 2
c() // 3
```

### Фикс переменной цикла (Go 1.22)

**До Go 1.22** — классическая ловушка:
```go
// BUG: все горутины печатают "3" (или последнее значение)
for _, v := range []int{1, 2, 3} {
    go func() {
        fmt.Println(v) // замыкание захватывает ОДНУ переменную v
    }()
}
// v переиспользуется — все горутины видят последнее значение

// Обходной путь до 1.22:
for _, v := range []int{1, 2, 3} {
    v := v // shadow: создаём новую переменную
    go func() {
        fmt.Println(v)
    }()
}
```

**С Go 1.22:** каждая итерация создаёт **новую** переменную. Баг больше не воспроизводится. Это одно из самых значимых изменений в языке.

### defer

defer откладывает вызов функции до возврата из текущей функции. Выполняется в порядке **LIFO** (стек):

```go
func example() {
    defer fmt.Println("first")
    defer fmt.Println("second")
    defer fmt.Println("third")
    fmt.Println("main")
}
// Вывод: main, third, second, first

// Типичное использование — cleanup
func readFile(path string) error {
    f, err := os.Open(path)
    if err != nil {
        return err
    }
    defer f.Close() // гарантированно закроем файл

    // ... работа с файлом
    return nil
}
```

**Важно: аргументы defer вычисляются СРАЗУ:**

```go
func example() {
    x := 0
    defer fmt.Println(x) // x=0 вычисляется СЕЙЧАС
    x = 42
}
// Вывод: 0 (не 42!)

// Если нужно "текущее" значение — используй замыкание:
func example() {
    x := 0
    defer func() { fmt.Println(x) }() // замыкание захватывает x
    x = 42
}
// Вывод: 42
```

### defer и именованные возвращаемые значения

```go
func doSomething() (err error) {
    tx, err := db.Begin()
    if err != nil {
        return err
    }

    defer func() {
        if err != nil {
            tx.Rollback() // откатываем при ошибке
        } else {
            err = tx.Commit() // defer может ИЗМЕНИТЬ возвращаемое значение!
        }
    }()

    // ... операции с транзакцией
    return nil
}
```

### defer в циклах

```go
// ПЛОХО: файлы не закроются до выхода из функции
func processFiles(paths []string) error {
    for _, p := range paths {
        f, _ := os.Open(p)
        defer f.Close() // все defer'ы накопятся!
    }
    // ... все файлы открыты одновременно
}

// ХОРОШО: вынеси в отдельную функцию
func processFile(path string) error {
    f, _ := os.Open(path)
    defer f.Close()
    // ... обработка
    return nil
}

func processFiles(paths []string) error {
    for _, p := range paths {
        if err := processFile(p); err != nil {
            return err
        }
    }
    return nil
}
```

### init() функции

```go
// init() вызывается автоматически при импорте пакета
// Может быть несколько init() в одном файле и пакете

var config Config

func init() {
    // Инициализация пакета
    config = loadConfig()
}

// Порядок: переменные пакета → init() → main()
// Зависимости пакетов: init() вызываются в порядке зависимостей

// Побочный эффект импорта:
import _ "net/http/pprof" // только ради init() — регистрирует pprof handlers
```

### Функциональные типы

```go
// Функция как тип
type HandlerFunc func(w http.ResponseWriter, r *http.Request)

// Функция как аргумент
func apply(data []int, fn func(int) int) []int {
    result := make([]int, len(data))
    for i, v := range data {
        result[i] = fn(v)
    }
    return result
}

doubled := apply([]int{1, 2, 3}, func(x int) int { return x * 2 })
// [2, 4, 6]

// Метод как значение
type Logger struct{ prefix string }
func (l *Logger) Log(msg string) { fmt.Println(l.prefix, msg) }

l := &Logger{prefix: "[INFO]"}
logFn := l.Log // method value — привязан к l
logFn("hello") // [INFO] hello
```

## Частые вопросы на собеседованиях

**Q: В каком порядке выполняются defer'ы?**
A: LIFO (Last In, First Out) — последний defer выполняется первым.

**Q: Когда вычисляются аргументы defer?**
A: В момент вызова defer (не в момент выполнения). Если нужно отложенное вычисление — оберни в замыкание.

**Q: Что изменилось с переменной цикла в Go 1.22?**
A: Каждая итерация for/range создаёт новую переменную. Раньше переменная переиспользовалась, и замыкания в горутинах захватывали одну переменную.

**Q: Можно ли иметь несколько init() в одном пакете?**
A: Да, можно несколько init() в одном файле и в разных файлах пакета. Порядок внутри файла — сверху вниз. Между файлами — алфавитный порядок файлов (но не стоит на это полагаться).

**Q: Как defer взаимодействует с именованными возвращаемыми значениями?**
A: defer с замыканием может читать и изменять именованные возвращаемые значения. Это используется для модификации ошибки перед возвратом.

## Подводные камни

1. **defer в цикле** накапливает все отложенные вызовы до выхода из функции — может привести к утечке ресурсов.

2. **Panic в init()** крашит программу при старте без возможности recover.

3. **Функция с именованными возвратами и голым return** ухудшает читаемость — используй осторожно:
```go
// Неочевидно что возвращается
func calc() (x, y int) {
    x = 1
    // y = 0 (zero value) — легко пропустить
    return
}
```
