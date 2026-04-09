# Memory Optimization

## Аллокации: heap vs stack

```go
// Stack allocation — бесплатно (bump pointer)
// Heap allocation — дорого (GC pressure)

// Escape analysis определяет куда:
// go build -gcflags="-m" → показывает escapes

// ❌ Escapes to heap (возврат указателя)
func newUser() *User {
    u := User{Name: "Alice"} // escapes
    return &u
}

// ✅ Stack (не возвращает указатель)
func processUser() {
    u := User{Name: "Alice"} // stays on stack
    validate(u)
}

// ❌ Escapes: interface{}, слишком большой для stack, closure capture
func logValue(v any) { // v escapes (any = interface{})
    fmt.Println(v)
}

// Правила (не абсолютные, escape analysis решает):
// - Указатель возвращается наружу → heap
// - Значение сохраняется в interface{} → heap
// - Closure captures by reference → heap
// - Размер > ~64KB → heap
// - Slice/map растёт непредсказуемо → heap
```

## sync.Pool

```go
// Переиспользование аллокаций между GC циклами
var bufPool = sync.Pool{
    New: func() any {
        buf := make([]byte, 0, 4096)
        return &buf
    },
}

func processRequest(data []byte) []byte {
    bufp := bufPool.Get().(*[]byte)
    buf := (*bufp)[:0] // reset length, keep capacity
    defer func() {
        *bufp = buf
        bufPool.Put(bufp)
    }()

    buf = append(buf, data...)
    // process buf...
    return slices.Clone(buf) // clone перед возвратом!
}

// bytes.Buffer pool
var bufferPool = sync.Pool{
    New: func() any { return new(bytes.Buffer) },
}

func encodeJSON(v any) ([]byte, error) {
    buf := bufferPool.Get().(*bytes.Buffer)
    buf.Reset()
    defer bufferPool.Put(buf)

    if err := json.NewEncoder(buf).Encode(v); err != nil {
        return nil, err
    }
    return slices.Clone(buf.Bytes()), nil
}

// ВАЖНО:
// 1. Pool очищается при каждом GC
// 2. Не для connection pooling (используй database/sql, etc.)
// 3. Всегда Reset перед Put
// 4. Никогда не храни ссылки на pooled объекты после Put
```

## Предаллокация slices и maps

```go
// ❌ Рост slice → множественные аллокации
var result []User
for _, raw := range rawUsers {
    result = append(result, parseUser(raw))
}

// ✅ Предаллокация
result := make([]User, 0, len(rawUsers))
for _, raw := range rawUsers {
    result = append(result, parseUser(raw))
}

// ❌ Map растёт → rehash
m := make(map[string]int)

// ✅ Предаллокация
m := make(map[string]int, expectedSize)

// strings.Builder
var b strings.Builder
b.Grow(estimatedSize) // одна аллокация
for _, s := range parts {
    b.WriteString(s)
}
```

## Избежание лишних копирований

```go
// ❌ Копирование строки → []byte → строки
func process(s string) string {
    b := []byte(s)       // copy!
    // modify b
    return string(b)     // copy!
}

// ✅ unsafe конверсия (если не модифицируешь)
import "unsafe"

func stringToBytes(s string) []byte {
    return unsafe.Slice(unsafe.StringData(s), len(s))
}
// ОСТОРОЖНО: модификация → undefined behavior!

// ✅ Передавай []byte если дальше будешь менять
// ✅ Используй strings.Builder вместо конкатенации

// ❌ Копирование больших структур
func process(big BigStruct) { /* copy 1KB struct */ }

// ✅ Передавай указатель
func process(big *BigStruct) { /* no copy */ }
```

## Struct layout (padding)

```go
// ❌ Плохой layout: 32 bytes (padding)
type Bad struct {
    a bool    // 1 byte + 7 padding
    b int64   // 8 bytes
    c bool    // 1 byte + 7 padding
    d int64   // 8 bytes
}

// ✅ Хороший layout: 24 bytes
type Good struct {
    b int64   // 8 bytes
    d int64   // 8 bytes
    a bool    // 1 byte
    c bool    // 1 byte + 6 padding
}

// Правило: поля от большего к меньшему
// Проверка: unsafe.Sizeof(Bad{}) vs unsafe.Sizeof(Good{})
```

## Reducing GC pressure

```
1. Меньше аллокаций = меньше работы GC
   - sync.Pool для горячих путей
   - Предаллокация slices/maps
   - Value types вместо pointers (где можно)

2. GOGC (default 100):
   - GOGC=200 → GC срабатывает реже, больше memory
   - GOGC=50 → GC чаще, меньше memory

3. GOMEMLIMIT (Go 1.19+):
   - Мягкий лимит памяти
   - GC учитывает лимит при планировании
   - GOMEMLIMIT=1GiB

4. Ballast (до GOMEMLIMIT):
   ballast := make([]byte, 1<<30) // 1GB
   _ = ballast
   // Увеличивает heap → GC реже
   // Устарело: используй GOMEMLIMIT
```
