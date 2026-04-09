# CPU Optimization

## Cache-Friendly Code

```go
// CPU cache: L1 (1ns), L2 (3ns), L3 (10ns), RAM (100ns)
// Cache line = 64 bytes

// ❌ Array of Structs (AoS) — плохо для конкретного поля
type User struct {
    ID    int64
    Name  string  // 16 bytes
    Email string  // 16 bytes
    Age   int64
    Score float64
}
users := make([]User, 1_000_000)
// Итерация по Score: загружает ВСЮ структуру (~56 bytes)
for _, u := range users {
    total += u.Score
}

// ✅ Struct of Arrays (SoA) — cache-friendly для одного поля
type Users struct {
    IDs    []int64
    Names  []string
    Emails []string
    Ages   []int64
    Scores []float64
}
// Итерация по Scores: только float64 в cache line
for _, s := range users.Scores {
    total += s
}
// 8 float64 в одну cache line (64/8 = 8)

// На практике: SoA для hot loops с миллионами элементов
// AoS для обычного кода (проще, читабельнее)
```

## False Sharing

```go
// False sharing: два ядра пишут в разные переменные,
// но они в одной cache line → cache invalidation

// ❌ False sharing
type Counters struct {
    a atomic.Int64 // core 1 пишет
    b atomic.Int64 // core 2 пишет — та же cache line!
}

// ✅ Padding между полями
type Counters struct {
    a atomic.Int64
    _ [56]byte      // padding до 64 bytes (cache line)
    b atomic.Int64
}
```

## Batching

```go
// ❌ По одному
for _, item := range items {
    db.ExecContext(ctx, "INSERT INTO items (name) VALUES ($1)", item.Name)
}

// ✅ Batch insert
const batchSize = 1000
for i := 0; i < len(items); i += batchSize {
    end := min(i+batchSize, len(items))
    batch := items[i:end]

    query := buildBatchInsert(batch) // INSERT INTO items VALUES ($1),($2),...
    db.ExecContext(ctx, query, args...)
}

// ✅ COPY (PostgreSQL, самый быстрый)
stmt, _ := tx.Prepare(pq.CopyIn("items", "name", "price"))
for _, item := range items {
    stmt.Exec(item.Name, item.Price)
}
stmt.Exec() // flush
```

## Precomputation

```go
// ❌ Компиляция regexp каждый раз
func validate(email string) bool {
    re := regexp.MustCompile(`^[a-z]+@[a-z]+\.[a-z]+$`) // compile каждый вызов!
    return re.MatchString(email)
}

// ✅ Compile один раз
var emailRegex = regexp.MustCompile(`^[a-z]+@[a-z]+\.[a-z]+$`)

func validate(email string) bool {
    return emailRegex.MatchString(email) // без compile
}

// ✅ Precompute lookup tables
var statusText = map[int]string{
    200: "OK",
    404: "Not Found",
    500: "Internal Server Error",
}
```

## Concurrency оптимизации

```go
// Шардированный мьютекс (уменьшает contention)
type ShardedMap[V any] struct {
    shards [256]struct {
        mu sync.RWMutex
        m  map[string]V
    }
}

func (s *ShardedMap[V]) getShard(key string) *struct {
    mu sync.RWMutex
    m  map[string]V
} {
    h := fnv.New32a()
    h.Write([]byte(key))
    return &s.shards[h.Sum32()%256]
}

func (s *ShardedMap[V]) Get(key string) (V, bool) {
    shard := s.getShard(key)
    shard.mu.RLock()
    defer shard.mu.RUnlock()
    v, ok := shard.m[key]
    return v, ok
}

// atomic вместо mutex для простых счётчиков
var counter atomic.Int64
counter.Add(1) // lock-free, быстрее mutex
```

## Compiler Hints

```go
// Inlining: маленькие функции инлайнятся автоматически
// go build -gcflags="-m" — показывает inlining decisions

//go:noinline  — запретить inline (для benchmarks)
func doWork() int { return 42 }

// Bounds check elimination
s := make([]int, 10)
_ = s[9] // compiler eliminates bounds check для s[0]..s[8]

// Prevent dead code elimination in benchmarks
var sink any

func BenchmarkProcess(b *testing.B) {
    for b.Loop() {
        result := process()
        sink = result // предотвращает оптимизацию
    }
}
```
