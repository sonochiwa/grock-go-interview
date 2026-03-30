# Структуры

## Обзор

Структуры — основной способ группировки данных в Go. Нет классов и наследования — вместо этого композиция через embedding. Понимание value vs pointer receiver — один из самых частых вопросов на собесах.

## Концепции

### Определение и инициализация

```go
type User struct {
    ID        int64
    Name      string
    Email     string
    CreatedAt time.Time
}

// Способы создания
u1 := User{ID: 1, Name: "Alice"}          // именованные поля
u2 := User{1, "Bob", "bob@mail.com", time.Now()} // позиционно (хрупко!)
u3 := new(User)                             // *User, все zero values
var u4 User                                  // zero value

// Указатель на структуру
u5 := &User{ID: 5, Name: "Eve"}
```

### Embedding (встраивание)

Go использует **композицию вместо наследования**:

```go
type Animal struct {
    Name string
}

func (a Animal) Speak() string {
    return a.Name + " makes a sound"
}

type Dog struct {
    Animal     // встроенное поле (без имени)
    Breed string
}

d := Dog{
    Animal: Animal{Name: "Rex"},
    Breed:  "Labrador",
}

// Методы Animal "промоутятся" — можно вызывать напрямую
fmt.Println(d.Speak())  // "Rex makes a sound"
fmt.Println(d.Name)     // "Rex" — поле тоже промоутится

// Но это НЕ наследование:
var a Animal = d // ОШИБКА: Dog != Animal
```

### Методы: value vs pointer receiver

```go
type Counter struct {
    n int
}

// Value receiver — работает с КОПИЕЙ
func (c Counter) Value() int {
    return c.n
}

// Pointer receiver — работает с ОРИГИНАЛОМ
func (c *Counter) Increment() {
    c.n++
}

c := Counter{}
c.Increment() // Go автоматически берёт адрес: (&c).Increment()
c.Value()     // Go автоматически разыменовывает если нужно
```

**Правила выбора receiver:**

| Pointer receiver (`*T`) | Value receiver (`T`) |
|---|---|
| Метод изменяет состояние | Метод только читает |
| Структура большая (> 64 байт) | Структура маленькая |
| Нужна консистентность (если хоть один метод pointer — все pointer) | Базовые типы (time.Time, etc.) |
| Реализуешь интерфейс для *T | Неизменяемый тип |

**Важное правило:** если хотя бы один метод имеет pointer receiver, делай ВСЕ методы с pointer receiver для консистентности.

### Анонимные структуры

```go
// Для одноразового использования
point := struct {
    X, Y int
}{10, 20}

// Полезно в тестах (table-driven tests)
tests := []struct {
    name     string
    input    int
    expected int
}{
    {"positive", 5, 25},
    {"zero", 0, 0},
    {"negative", -3, 9},
}
```

### Сравнение структур

```go
// Структура сравнима, если ВСЕ поля сравнимы
type Point struct { X, Y int }
p1, p2 := Point{1, 2}, Point{1, 2}
fmt.Println(p1 == p2) // true

// Структура с несравнимым полем — ошибка компиляции
type Data struct {
    Values []int // слайсы несравнимы!
}
// d1 == d2 // ОШИБКА: не компилируется
// Используй reflect.DeepEqual(d1, d2) или сравнивай вручную
```

## Под капотом: padding и alignment

Компилятор выравнивает поля для оптимального доступа к памяти:

```go
// Неоптимальный порядок (24 байта с padding)
type Bad struct {
    a bool   // 1 байт + 7 padding
    b int64  // 8 байт
    c bool   // 1 байт + 7 padding
}
// sizeof = 24

// Оптимальный порядок (16 байт, без лишнего padding)
type Good struct {
    b int64  // 8 байт
    a bool   // 1 байт
    c bool   // 1 байт + 6 padding
}
// sizeof = 16
```

**Правило:** располагай поля от больших к маленьким.

Проверка размера:
```go
fmt.Println(unsafe.Sizeof(Bad{}))  // 24
fmt.Println(unsafe.Sizeof(Good{})) // 16
```

## Частые вопросы на собеседованиях

**Q: В чём разница между value и pointer receiver?**
A: Value receiver работает с копией — не может изменить оригинал. Pointer receiver работает с оригиналом. Если тип реализует интерфейс через pointer receiver, только указатель удовлетворяет интерфейсу.

**Q: Чем embedding отличается от наследования?**
A: Embedding — это композиция, не наследование. Dog не является Animal, но включает его. Нет полиморфизма — `var a Animal = Dog{}` не работает. Методы промоутятся, но это синтаксический сахар.

**Q: Можно ли сравнивать структуры через ==?**
A: Только если все поля сравнимы. Структуры с слайсами, мапами или функциями несравнимы.

**Q: Что такое padding в структурах?**
A: Компилятор добавляет байты между полями для выравнивания по границам памяти (alignment). Порядок полей влияет на размер структуры.

## Подводные камни

1. **Pointer receiver и интерфейсы**:
```go
type Stringer interface { String() string }
type MyType struct{ s string }
func (m *MyType) String() string { return m.s }

var s Stringer = MyType{}  // ОШИБКА: MyType не реализует Stringer
var s Stringer = &MyType{} // OK
```

2. **Встраивание и конфликт имён**:
```go
type A struct { Name string }
type B struct { Name string }
type C struct { A; B }
// c.Name — ОШИБКА: ambiguous selector
// c.A.Name — OK
```

3. **Не экспортированные поля** не сериализуются:
```go
type User struct {
    Name string // экспортируется, попадёт в JSON
    age  int    // НЕ экспортируется, пропустится
}
```
