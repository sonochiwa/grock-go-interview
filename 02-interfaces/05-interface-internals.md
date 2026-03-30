# Внутреннее устройство интерфейсов

## Обзор

Это самый важный файл раздела. Как интерфейсы устроены внутри — один из топовых вопросов на собеседованиях middle+ уровня. Понимание iface/eface объясняет все "странности" с nil интерфейсами.

## Под капотом

### Две структуры: iface и eface

Go использует две разные структуры для интерфейсов:

```go
// runtime/runtime2.go

// eface — empty interface (interface{} / any)
type eface struct {
    _type *_type        // указатель на информацию о типе
    data  unsafe.Pointer // указатель на данные
}

// iface — non-empty interface (с методами)
type iface struct {
    tab  *itab           // указатель на таблицу типа+методов
    data unsafe.Pointer  // указатель на данные
}
```

Оба занимают **16 байт** (два указателя на 64-bit системе).

### Структура _type

```go
// runtime/type.go (упрощённо)
type _type struct {
    size       uintptr  // размер типа
    ptrdata    uintptr  // размер данных-указателей (для GC)
    hash       uint32   // хеш типа (для быстрого сравнения)
    tflag      tflag    // флаги
    align      uint8    // выравнивание
    fieldAlign uint8    // выравнивание полей
    kind       uint8    // вид типа (int, struct, ptr, ...)
    equal      func(unsafe.Pointer, unsafe.Pointer) bool // функция сравнения
    gcdata     *byte    // GC-данные
    str        nameOff  // имя типа
    ptrToThis  typeOff  // указатель на *T
}
```

### Структура itab

```go
// runtime/runtime2.go (упрощённо)
type itab struct {
    inter *interfacetype // указатель на тип интерфейса
    _type *_type         // указатель на конкретный тип
    hash  uint32         // копия _type.hash (для быстрого type switch)
    _     [4]byte        // padding
    fun   [1]uintptr     // массив указателей на методы (variable-size)
}

// fun — это на самом деле fun[N], где N — количество методов интерфейса
// Методы отсортированы и соответствуют методам интерфейса
```

### Как происходит вызов метода

```go
type Stringer interface { String() string }
type MyType struct { s string }
func (m *MyType) String() string { return m.s }

var s Stringer = &MyType{"hello"}
s.String()
```

Что происходит при `s.String()`:
1. Берём `s.tab` → `itab`
2. Берём `itab.fun[0]` → адрес метода String для *MyType
3. Вызываем метод, передавая `s.data` как receiver

Это **один indirection** сверх обычного вызова метода.

### Кеширование itab

```go
// Runtime кеширует itab'ы в глобальной хеш-таблице
// Ключ: (interface type, concrete type)
// При первом создании interface из конкретного типа — itab создаётся и кешируется
// Все последующие — берут из кеша

// Это значит:
var s1 Stringer = &MyType{"a"} // первый раз: создаёт itab, кеширует
var s2 Stringer = &MyType{"b"} // из кеша — быстро
// s1.tab == s2.tab — один и тот же itab!
```

### Аллокация данных

```go
// Маленькие значения (≤ размер указателя) хранятся INLINE в поле data
var i interface{} = 42
// i._type → *_type для int
// i.data → 42 (значение хранится прямо в указателе, без аллокации)

// Большие значения — аллокация в куче
var i interface{} = [100]int{}
// i.data → указатель на аллоцированный в куче массив
```

## КРИТИЧНО: nil interface vs interface с nil value

**Самый частый вопрос на собеседованиях по интерфейсам!**

```go
// 1. nil interface — оба поля nil
var w io.Writer // w.tab == nil, w.data == nil
fmt.Println(w == nil) // true

// 2. Interface с nil значением — tab заполнен, data == nil
var p *os.File         // p == nil
var w io.Writer = p    // w.tab → itab для (*os.File, io.Writer)
                       // w.data == nil
fmt.Println(w == nil)  // FALSE!!! tab != nil
fmt.Println(p == nil)  // true

// Это классическая ловушка:
func getWriter() io.Writer {
    var f *os.File // nil
    // ... какая-то логика, f остаётся nil
    return f // ОШИБКА: возвращает non-nil interface с nil data!
}

w := getWriter()
if w != nil {
    w.Write([]byte("hello")) // PANIC: nil pointer dereference!
}

// ПРАВИЛЬНО:
func getWriter() io.Writer {
    var f *os.File
    // ...
    if f == nil {
        return nil // возвращает nil interface (оба поля nil)
    }
    return f
}
```

### Визуализация

```
nil interface:
┌──────┬──────┐
│ tab  │ data │
│ nil  │ nil  │
└──────┴──────┘
== nil? → true

interface holding nil *os.File:
┌──────────────┬──────┐
│     tab      │ data │
│ *itab(File)  │ nil  │
└──────────────┴──────┘
== nil? → false (tab != nil!)
```

## Стоимость интерфейсов

```go
// Прямой вызов метода: ~1.5 ns
// Вызов через интерфейс: ~3 ns (дополнительный indirection)
// Разница незначительна для большинства случаев

// Основная стоимость — АЛЛОКАЦИЯ при упаковке в интерфейс:
type Big struct { data [1024]byte }
var i interface{} = Big{} // аллокация 1KB в куче!

// Для указателей — аллокации нет (копируется только указатель):
var i interface{} = &Big{} // нет дополнительной аллокации
```

## Частые вопросы на собеседованиях

**Q: Чем отличается nil interface от interface с nil значением?**
A: nil interface — оба поля (tab/type и data) равны nil, сравнение с nil даёт true. Interface с nil значением — поле типа заполнено, только data nil. Сравнение с nil даёт false.

**Q: Как устроен interface под капотом?**
A: Два слова (16 байт): iface = {itab, data}, eface = {type, data}. itab содержит таблицу методов и кешируется.

**Q: Какова стоимость вызова метода через интерфейс?**
A: Один дополнительный indirection (~1-2 ns сверху). Основная стоимость — возможная аллокация при упаковке значения в интерфейс.

**Q: Когда значение при упаковке в интерфейс уходит в кучу?**
A: Если размер значения > размера указателя. Маленькие значения (int, bool, указатели) могут храниться inline.

## Подводные камни

1. **Возврат конкретного nil через интерфейс** — самая частая ошибка (см. пример выше).

2. **reflect.ValueOf на nil interface** — паникует:
```go
var i interface{} // nil
reflect.ValueOf(i) // zero Value, проверяй IsValid()
```

3. **Неожиданные аллокации** — передача struct (не указателя) в интерфейс аллоцирует в куче.
