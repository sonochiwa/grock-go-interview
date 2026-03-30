# Типы и переменные

## Обзор

Go — статически типизированный язык. Понимание системы типов — основа для всего остального: интерфейсов, дженериков, рефлексии.

## Базовые типы

```go
// Целочисленные (знаковые)
int8   // -128 .. 127
int16  // -32768 .. 32767
int32  // -2^31 .. 2^31-1
int64  // -2^63 .. 2^63-1
int    // платформозависимый: 64-bit на 64-bit системе

// Целочисленные (беззнаковые)
uint8  // 0 .. 255 (он же byte)
uint16
uint32
uint64
uint   // платформозависимый
uintptr // достаточно для хранения указателя

// С плавающей точкой
float32
float64

// Комплексные
complex64
complex128

// Другие
bool       // true / false
byte       // алиас для uint8
rune       // алиас для int32 (Unicode code point)
string     // неизменяемая последовательность байт
```

## Zero values

Каждый тип в Go имеет нулевое значение. Переменная без инициализации **всегда** имеет zero value:

```go
var i int       // 0
var f float64   // 0.0
var b bool      // false
var s string    // "" (пустая строка)
var p *int      // nil
var sl []int    // nil
var m map[string]int // nil
var ch chan int  // nil
var fn func()   // nil
var iface error // nil
```

**На собесе:** "Чем nil slice отличается от empty slice?" — см. раздел слайсов.

## Объявление переменных

```go
// Полная форма
var x int = 10

// Тип выводится из значения
var x = 10

// Короткая форма (только внутри функций)
x := 10

// Множественное объявление
var (
    name string = "Go"
    version int = 26
)

// Множественное присваивание
a, b := 1, 2
a, b = b, a // swap без временной переменной
```

## Преобразование типов

В Go **нет неявных** преобразований типов (в отличие от C/C++):

```go
var i int = 42
var f float64 = float64(i)  // явное преобразование
var u uint = uint(f)

// Это НЕ скомпилируется:
// var f float64 = i  // cannot use i (type int) as type float64
```

## Type alias vs Type definition

```go
// Type definition — создаёт НОВЫЙ тип
type UserID int64
type AdminID int64

var u UserID = 1
var a AdminID = 2
// u = a  // ошибка компиляции! Разные типы

// Type alias — просто другое имя для того же типа
type MyInt = int
var x MyInt = 42
var y int = x  // ОК — это один и тот же тип

// Стандартные алиасы в Go:
// byte = uint8
// rune = int32
// any  = interface{} (с Go 1.18)
```

**Важно:** type definition создаёт новый тип с ТЕМИ ЖЕ underlying type, но без методов оригинала. Type alias сохраняет все методы.

## Константы и iota

```go
// Нетипизированные константы (untyped constants)
const Pi = 3.14159 // тип определится при использовании
const Big = 1 << 100 // это работает! untyped constant может быть огромным

// Типизированные константы
const MaxRetries int = 3

// iota — генератор последовательностей (сбрасывается в каждом const-блоке)
const (
    Sunday    = iota // 0
    Monday           // 1
    Tuesday          // 2
    Wednesday        // 3
    Thursday         // 4
    Friday           // 5
    Saturday         // 6
)

// iota с выражениями
const (
    _  = iota             // 0 — пропускаем
    KB = 1 << (10 * iota) // 1 << 10 = 1024
    MB                    // 1 << 20
    GB                    // 1 << 30
    TB                    // 1 << 40
)

// Битовые флаги
const (
    FlagRead    = 1 << iota // 1
    FlagWrite               // 2
    FlagExecute             // 4
)

permissions := FlagRead | FlagWrite // 3
canWrite := permissions&FlagWrite != 0 // true
```

### Нетипизированные константы (untyped constants)

Это уникальная фича Go. Нетипизированная константа имеет "идеальную" точность и приобретает тип только при использовании:

```go
const x = 3 // untyped int

var i int = x       // OK
var f float64 = x   // OK — x становится float64
var b byte = x      // OK — x становится byte

// Работает даже с огромными числами
const huge = 1 << 200 // OK как константа
// var h int = huge   // ошибка: переполнение
const ratio = huge / (1 << 190) // = 1024, это уже помещается в int
var r int = ratio // OK
```

## Частые вопросы на собеседованиях

**Q: Чем отличается type alias от type definition?**
A: Type definition (`type X int`) создаёт новый тип — нельзя присвоить значение другого типа без конверсии. Type alias (`type X = int`) — просто другое имя, полная совместимость.

**Q: Что такое untyped constant?**
A: Константа без явного типа, имеющая произвольную точность. Тип определяется в момент использования. Это позволяет `const Pi = 3.14` работать и с float32, и с float64.

**Q: Какой размер у int?**
A: Зависит от платформы. На 64-bit системах — 8 байт (64 бита). На 32-bit — 4 байта. Если нужен конкретный размер — используй int64/int32 явно.

**Q: Есть ли enum в Go?**
A: Нет полноценных enum. Используется `const` + `iota`. Для строкового представления — `go generate` + `stringer`.

## Подводные камни

1. **int != int64**: на 64-bit системе `int` и `int64` имеют одинаковый размер, но это **разные типы**. Нужно явное приведение.

2. **Целочисленное переполнение**: Go не паникует при переполнении — значение молча "заворачивается":
```go
var x uint8 = 255
x++ // x == 0, без ошибки!
```

3. **Деление целых чисел**:
```go
fmt.Println(7 / 2)   // 3, не 3.5!
fmt.Println(7.0 / 2) // 3.5
```
