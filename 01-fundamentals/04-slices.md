# Слайсы

## Обзор

Слайс — динамический массив в Go. Один из самых часто используемых и часто спрашиваемых типов. Понимание внутреннего устройства — маст-хэв для middle.

## Концепции

### Создание

```go
// Литерал
s1 := []int{1, 2, 3}

// make(тип, длина, ёмкость)
s2 := make([]int, 5)       // len=5, cap=5, заполнен нулями
s3 := make([]int, 0, 100)  // len=0, cap=100, пустой но с зарезервированной памятью

// Из массива (слайс — это "окно" в массив)
arr := [5]int{1, 2, 3, 4, 5}
s4 := arr[1:4]  // [2, 3, 4], len=3, cap=4
```

### nil slice vs empty slice

```go
var s1 []int        // nil slice: s1 == nil, len=0, cap=0
s2 := []int{}       // empty slice: s2 != nil, len=0, cap=0
s3 := make([]int, 0) // empty slice: s3 != nil, len=0, cap=0

// Функционально идентичны:
len(s1) == len(s2) // true (0 == 0)
append(s1, 1)      // работает
append(s2, 1)      // работает

// НО: json.Marshal отличается!
json.Marshal(s1) // "null"
json.Marshal(s2) // "[]"
```

### append

```go
s := []int{1, 2, 3}
s = append(s, 4)        // добавить один элемент
s = append(s, 5, 6, 7)  // добавить несколько
s = append(s, other...)  // добавить другой слайс

// ВАЖНО: append может вернуть НОВЫЙ слайс!
s1 := make([]int, 3, 5) // len=3, cap=5
s2 := append(s1, 4)     // len=4, cap=5 — тот же backing array
s3 := append(s1, 4, 5, 6) // len=6 > cap=5 — НОВЫЙ backing array!
```

### Slice tricks

```go
// Удаление элемента (с сохранением порядка)
s = append(s[:i], s[i+1:]...)

// Удаление элемента (без сохранения порядка — быстрее)
s[i] = s[len(s)-1]
s = s[:len(s)-1]

// Вставка элемента
s = append(s[:i+1], s[i:]...)
s[i] = value

// Копирование
dst := make([]int, len(src))
copy(dst, src)

// Фильтрация (in-place, без аллокации)
n := 0
for _, v := range s {
    if keepCondition(v) {
        s[n] = v
        n++
    }
}
s = s[:n]
```

## Под капотом

### Заголовок слайса (SliceHeader)

```go
// runtime/slice.go (упрощённо)
type slice struct {
    array unsafe.Pointer // указатель на backing array
    len   int            // текущая длина
    cap   int            // ёмкость (до конца backing array)
}
```

Слайс = 24 байта на 64-bit системе (3 × 8 байт). Передача слайса в функцию копирует только заголовок, НЕ данные.

### Алгоритм роста

При `append`, если `len == cap`, создаётся новый backing array:

**До Go 1.18:**
- cap < 1024: удваиваем (×2)
- cap >= 1024: увеличиваем на 25% (×1.25)

**С Go 1.18+ (текущий):**
- Плавный рост без резкого перехода на 1024
- Формула: `newcap = old + (old + 3*256) / 4`
- Маленькие слайсы растут ~×2, большие плавно переходят к ~×1.25
- Финальный размер округляется до size class аллокатора

```go
// Демонстрация роста
s := make([]int, 0)
prev := cap(s)
for i := 0; i < 10000; i++ {
    s = append(s, i)
    if cap(s) != prev {
        fmt.Printf("len=%5d cap=%5d growth=%.2f\n", len(s), cap(s), float64(cap(s))/float64(prev))
        prev = cap(s)
    }
}
```

### Слайс от массива (shared memory)

```go
arr := [5]int{1, 2, 3, 4, 5}
s := arr[1:3] // s = [2, 3], но разделяет память с arr!

s[0] = 99
fmt.Println(arr) // [1, 99, 3, 4, 5] — arr тоже изменился!

// Полный синтаксис слайсинга: [low:high:max]
s2 := arr[1:3:3] // len=2, cap=2 (cap ограничен!)
// Теперь append(s2, ...) гарантированно создаст новый backing array
```

## Частые вопросы на собеседованиях

**Q: Что произойдёт при передаче слайса в функцию?**
A: Копируется заголовок (24 байта). Функция видит те же данные через тот же backing array. Может модифицировать элементы, но `append` внутри функции не повлияет на длину снаружи (len копируется).

**Q: Как происходит рост слайса?**
A: С Go 1.18 — плавный алгоритм: маленькие слайсы ~×2, большие ~×1.25. Размер округляется до size class аллокатора.

**Q: Чем nil slice отличается от empty slice?**
A: Функционально идентичны (len=0, cap=0, append работает). Разница: nil slice == nil (true), json.Marshal даёт "null" vs "[]".

**Q: Как избежать утечки памяти со слайсами?**
A: `s = s[:1]` сохраняет ссылку на весь backing array. Решение — copy в новый слайс.

**Q: Что делает полный синтаксис [low:high:max]?**
A: Третий параметр ограничивает cap, предотвращая случайное разделение backing array при append.

## Подводные камни

1. **Утечка памяти**: слайс от большого массива удерживает весь массив в памяти:
```go
// ПЛОХО
func getFirstByte(data []byte) []byte {
    return data[:1] // удерживает весь backing array!
}

// ХОРОШО
func getFirstByte(data []byte) []byte {
    result := make([]byte, 1)
    copy(result, data[:1])
    return result
}
```

2. **append может изменить чужие данные**:
```go
s := []int{1, 2, 3, 4, 5}
a := s[:3] // [1,2,3], cap=5
b := append(a, 99) // перезаписывает s[3]!
fmt.Println(s) // [1, 2, 3, 99, 5]
```

3. **range создаёт копию элемента**:
```go
type Item struct{ Val int }
items := []Item{{1}, {2}, {3}}
for _, item := range items {
    item.Val = 0 // модифицируем КОПИЮ, оригинал не меняется
}
// items всё ещё [{1}, {2}, {3}]
// Решение: for i := range items { items[i].Val = 0 }
```
