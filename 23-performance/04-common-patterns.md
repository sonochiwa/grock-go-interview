# Common Performance Patterns

## String Interning (дедупликация строк)

```go
// Проблема: миллионы строк с одинаковыми значениями
// "pending" × 1M = 7MB вместо 7 bytes

// Go 1.23+: unique.Handle
import "unique"

type Order struct {
    Status unique.Handle[string] // вместо string
}

pending := unique.Make("pending")
confirmed := unique.Make("confirmed")

// Все "pending" указывают на одну строку в памяти
o1.Status = pending
o2.Status = pending // та же память
o1.Status == o2.Status // true (pointer comparison, быстро)
```

## Избежание конверсий string ↔ []byte

```go
// Каждая конверсия = копирование

// ❌
func contains(s string, sub string) bool {
    return bytes.Contains([]byte(s), []byte(sub)) // 2 копии!
}

// ✅
func contains(s string, sub string) bool {
    return strings.Contains(s, sub) // zero copy
}

// Для map lookup по []byte:
// ❌ string(key) — копирование
m[string(key)]

// Go оптимизирует map lookup с string([]byte) — NO COPY
// Но только для lookup, не для insert
val, ok := m[string(byteKey)] // оптимизировано компилятором

// io.Writer без аллокаций:
// ❌ w.Write([]byte(s)) — копирование
// ✅ io.WriteString(w, s) — без копирования если Writer реализует StringWriter
```

## Избежание лишних аллокаций в горячих путях

```go
// ❌ fmt.Sprintf в горячем пути
for _, req := range requests {
    key := fmt.Sprintf("user:%d:session:%s", req.UserID, req.SessionID)
    cache.Get(key) // аллокация строки каждый раз
}

// ✅ strings.Builder или []byte
var buf []byte
for _, req := range requests {
    buf = buf[:0]
    buf = append(buf, "user:"...)
    buf = strconv.AppendInt(buf, int64(req.UserID), 10)
    buf = append(buf, ":session:"...)
    buf = append(buf, req.SessionID...)
    cache.Get(string(buf))
}

// ❌ Создание error каждый раз
func validate(s string) error {
    if s == "" {
        return fmt.Errorf("empty string") // аллокация!
    }
    return nil
}

// ✅ Sentinel error
var errEmptyString = errors.New("empty string")
func validate(s string) error {
    if s == "" {
        return errEmptyString // без аллокации
    }
    return nil
}
```

## JSON Performance

```go
// encoding/json — медленный (reflection)
// Альтернативы:
// - github.com/goccy/go-json — drop-in replacement, 2-3x быстрее
// - github.com/bytedance/sonic — SIMD, 5-10x быстрее (amd64)
// - github.com/mailru/easyjson — кодогенерация, zero reflection

// easyjson: генерирует Marshal/Unmarshal код
//go:generate easyjson -all user.go

type User struct {
    ID    int64  `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}
// Сгенерированный код: ~5x быстрее стандартного encoding/json

// Для API: если JSON parsing = bottleneck → switch to protobuf (gRPC)
```

## Benchmark-Driven Optimization

```go
// ПРАВИЛО: НЕ оптимизируй без бенчмарка!

// 1. Напиши benchmark
func BenchmarkProcess(b *testing.B) {
    data := loadTestData()
    b.ResetTimer()
    for b.Loop() {
        process(data)
    }
}

// 2. Профилируй
// go test -bench=. -cpuprofile=cpu.prof -memprofile=mem.prof
// go tool pprof cpu.prof

// 3. Оптимизируй bottleneck

// 4. Сравни
// benchstat old.txt new.txt

// 5. Повтори

// Порядок оптимизации:
// 1. Алгоритм (O(n²) → O(n log n)) — самый большой эффект
// 2. I/O (batching, caching, pooling)
// 3. Аллокации (sync.Pool, предаллокация)
// 4. CPU (cache-friendly, SIMD, inlining)
```
