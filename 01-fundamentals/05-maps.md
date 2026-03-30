# Карты (Maps)

## Обзор

Map — хеш-таблица (ассоциативный массив) в Go. Ключевые вопросы на собесах: nil map, конкурентный доступ, порядок итерации, внутреннее устройство (подробнее в разделе internals).

## Концепции

### Создание и использование

```go
// Литерал
m := map[string]int{
    "alice": 25,
    "bob":   30,
}

// make
m2 := make(map[string]int)    // пустая, готова к использованию
m3 := make(map[string]int, 100) // с подсказкой для начального размера (hint)

// Операции
m["charlie"] = 35      // запись
age := m["alice"]       // чтение (25)
missing := m["nobody"]  // чтение несуществующего → zero value (0)

// Comma-ok idiom
age, ok := m["alice"]   // ok == true
age, ok = m["nobody"]   // ok == false, age == 0

// Удаление
delete(m, "bob")
delete(m, "nonexistent") // OK, не паникует

// Длина
len(m) // количество элементов
```

### nil map

```go
var m map[string]int // nil map

// Чтение из nil map — OK (возвращает zero value)
v := m["key"]     // 0
_, ok := m["key"] // false
len(m)            // 0

// Запись в nil map — PANIC
m["key"] = 1 // panic: assignment to entry in nil map

// Решение: инициализируй map
m = make(map[string]int)
m = map[string]int{}
```

### Порядок итерации

```go
m := map[string]int{"a": 1, "b": 2, "c": 3}

// Порядок РАНДОМНЫЙ и НАМЕРЕННО рандомизирован (с Go 1.12)
for k, v := range m {
    fmt.Println(k, v) // порядок разный при каждом запуске!
}

// Если нужен порядок — собери ключи, отсортируй
keys := make([]string, 0, len(m))
for k := range m {
    keys = append(keys, k)
}
sort.Strings(keys)
for _, k := range keys {
    fmt.Println(k, m[k])
}
```

### Допустимые типы ключей

Ключ должен быть **comparable** (поддерживать ==):

```go
// OK: int, string, bool, float, pointer, array, struct (если все поля comparable), interface
map[string]int{}
map[[3]int]string{}     // массив как ключ — OK
map[struct{X,Y int}]string{} // struct как ключ — OK

// НЕ OK: slice, map, function
// map[[]int]string{} // ошибка компиляции
```

### Конкурентный доступ

```go
// FATAL: concurrent map read and map write
// Maps в Go НЕ потокобезопасны!
m := make(map[string]int)

// Плохо — data race:
go func() { m["a"] = 1 }()
go func() { _ = m["a"] }()

// Решение 1: sync.Mutex
var mu sync.Mutex
mu.Lock()
m["a"] = 1
mu.Unlock()

// Решение 2: sync.RWMutex (много читателей)
var rw sync.RWMutex
rw.RLock()
_ = m["a"]
rw.RUnlock()

// Решение 3: sync.Map (оптимизирован для append-only или stable keys)
var sm sync.Map
sm.Store("key", "value")
v, ok := sm.Load("key")
```

### sync.Map — когда использовать

```go
// sync.Map хорош для двух сценариев:
// 1. Ключи записываются один раз, читаются много раз (кэш)
// 2. Разные горутины работают с непересекающимися наборами ключей

// sync.Map ПЛОХ для:
// - Частых обновлений существующих ключей
// - Когда нужен len() или итерация — нет O(1) длины!
// - Типобезопасности — ключи и значения это any
```

### Map с struct значением

```go
type User struct{ Name string; Age int }
m := map[int]User{
    1: {Name: "Alice", Age: 25},
}

// Нельзя модифицировать поле напрямую:
// m[1].Age = 26 // ОШИБКА: cannot assign to struct field in map

// Решение 1: через временную переменную
u := m[1]
u.Age = 26
m[1] = u

// Решение 2: хранить указатели
m2 := map[int]*User{
    1: {Name: "Alice", Age: 25},
}
m2[1].Age = 26 // OK
```

## Под капотом

Кратко (подробнее в 09-internals):

- **До Go 1.24**: bucket-based хеш-таблица. Каждый bucket — 8 пар ключ-значение. При переполнении — overflow buckets. Рост × 2 при load factor > 6.5.
- **Go 1.24+**: Swiss Table — open addressing с группами по 16 слотов, SIMD-оптимизация на поддерживаемых платформах. Быстрее на 20-40% для lookup.

## Частые вопросы на собеседованиях

**Q: Что будет при записи в nil map?**
A: panic: assignment to entry in nil map.

**Q: Почему порядок итерации рандомный?**
A: Намеренно рандомизирован с Go 1.12, чтобы разработчики не зависели от порядка (раньше порядок казался стабильным, но не был гарантирован).

**Q: Как безопасно использовать map из нескольких горутин?**
A: sync.Mutex/RWMutex для обычных map, или sync.Map для специальных сценариев (append-only кэш, непересекающиеся ключи).

**Q: Почему нельзя взять адрес элемента map?**
A: Потому что map может перераспределить память при росте, и указатель станет невалидным.

## Подводные камни

1. **clear()** (Go 1.21+) — удаляет все элементы, но сохраняет выделенную память:
```go
m := map[string]int{"a": 1, "b": 2}
clear(m)
len(m) // 0, но память не освобождена
```

2. **Итерация и удаление** — безопасно удалять текущий элемент во время range:
```go
for k, v := range m {
    if v < 0 {
        delete(m, k) // OK — это безопасно
    }
}
```

3. **Map как множество (set)**:
```go
set := make(map[string]struct{}) // struct{} занимает 0 байт
set["item"] = struct{}{}
if _, ok := set["item"]; ok { ... }
```
