# Бенчмарки

## Обзор

Go имеет встроенную поддержку бенчмарков в пакете testing. Результаты воспроизводимы и сравнимы.

## Концепции

### Базовый бенчмарк

```go
// bench_test.go
func BenchmarkConcat(b *testing.B) {
    for i := 0; i < b.N; i++ {
        s := ""
        for j := 0; j < 100; j++ {
            s += "x"
        }
    }
}

func BenchmarkBuilder(b *testing.B) {
    for i := 0; i < b.N; i++ {
        var sb strings.Builder
        for j := 0; j < 100; j++ {
            sb.WriteString("x")
        }
        _ = sb.String()
    }
}
```

### Запуск

```bash
# Запуск всех бенчмарков
go test -bench=. ./...

# Конкретный бенчмарк
go test -bench=BenchmarkConcat -benchmem

# Несколько запусков для статистики
go test -bench=. -count=10

# Ограничение по времени
go test -bench=. -benchtime=5s

# Вывод:
# BenchmarkConcat-8      12345    95000 ns/op    50000 B/op    99 allocs/op
# BenchmarkBuilder-8    500000     3000 ns/op      512 B/op     1 allocs/op
```

### Сравнение бенчмарков

```bash
# Сохранить результаты
go test -bench=. -count=10 > old.txt
# ... сделать изменения ...
go test -bench=. -count=10 > new.txt

# Сравнить
go install golang.org/x/perf/cmd/benchstat@latest
benchstat old.txt new.txt
```

### Sub-benchmarks

```go
func BenchmarkSort(b *testing.B) {
    sizes := []int{10, 100, 1000, 10000}
    for _, size := range sizes {
        b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
            data := generateData(size)
            b.ResetTimer() // сбросить таймер после setup
            for i := 0; i < b.N; i++ {
                s := make([]int, len(data))
                copy(s, data)
                sort.Ints(s)
            }
        })
    }
}
```

### Правила написания бенчмарков

```go
// 1. b.ResetTimer() после дорогого setup
func BenchmarkProcess(b *testing.B) {
    data := loadTestData() // не входит в замер
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        process(data)
    }
}

// 2. b.StopTimer() / b.StartTimer() для пауз
func BenchmarkWithCleanup(b *testing.B) {
    for i := 0; i < b.N; i++ {
        result := process()
        b.StopTimer()
        cleanup(result) // не входит в замер
        b.StartTimer()
    }
}

// 3. Предотвращение оптимизации компилятором
var result int // package-level чтобы компилятор не удалил вызов
func BenchmarkCompute(b *testing.B) {
    var r int
    for i := 0; i < b.N; i++ {
        r = compute(42)
    }
    result = r // запись в package-level var
}

// 4. b.ReportAllocs() — показать аллокации
func BenchmarkAllocs(b *testing.B) {
    b.ReportAllocs()
    for i := 0; i < b.N; i++ { ... }
}
```

## Частые вопросы на собеседованиях

**Q: Что такое b.N?**
A: Количество итераций, определяемое testing framework. Увеличивается пока результат не стабилизируется.

**Q: Зачем -benchmem?**
A: Показывает количество аллокаций (allocs/op) и байт на операцию (B/op). Критично для понимания давления на GC.

**Q: Как предотвратить удаление кода компилятором в бенчмарке?**
A: Присвоить результат package-level переменной. Компилятор не может удалить вычисление, если результат используется.
