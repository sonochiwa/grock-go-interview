# reflect.Type и reflect.Value

## Обзор

Два основных типа пакета reflect: `Type` описывает тип, `Value` содержит значение.

## Концепции

```go
import "reflect"

x := 42
t := reflect.TypeOf(x)  // reflect.Type — описание типа
v := reflect.ValueOf(x) // reflect.Value — значение

fmt.Println(t.Name())   // "int"
fmt.Println(t.Kind())   // reflect.Int
fmt.Println(v.Int())    // 42
fmt.Println(v.Type())   // int
```

### Kind vs Type

```go
type UserID int64

t := reflect.TypeOf(UserID(0))
fmt.Println(t.Name()) // "UserID"
fmt.Println(t.Kind()) // reflect.Int64

// Kind — базовый вид (int, struct, ptr, slice, map, ...)
// Name — имя типа (может быть пустым для анонимных типов)
```

### Инспекция структур

```go
type User struct {
    Name  string `json:"name" validate:"required"`
    Email string `json:"email" validate:"email"`
    Age   int    `json:"age,omitempty"`
}

t := reflect.TypeOf(User{})
for i := 0; i < t.NumField(); i++ {
    f := t.Field(i)
    fmt.Printf("Field: %s, Type: %s, Tag: %s\n",
        f.Name, f.Type, f.Tag.Get("json"))
}
// Field: Name, Type: string, Tag: name
// Field: Email, Type: string, Tag: email
// Field: Age, Type: int, Tag: age,omitempty
```

### Модификация через reflect

```go
x := 42
v := reflect.ValueOf(&x).Elem() // нужен указатель для модификации!
v.SetInt(100)
fmt.Println(x) // 100

// Без указателя:
v := reflect.ValueOf(x)
v.SetInt(100) // PANIC: reflect.Value.SetInt using unaddressable value

// Проверка:
v.CanSet() // true/false
```

### Создание значений

```go
// Создать новый экземпляр типа
t := reflect.TypeOf(User{})
v := reflect.New(t) // *User (указатель)
user := v.Interface().(*User)
user.Name = "Alice"
```

## Частые вопросы на собеседованиях

**Q: Зачем нужен reflect?**
A: JSON/XML маршалинг, ORM, валидация, dependency injection, обобщённые утилиты (когда дженерики недостаточны).

**Q: Почему reflect медленный?**
A: Нет compile-time оптимизаций (inlining, devirtualization). Каждая операция проходит через runtime проверки типов.

**Q: Чем Kind отличается от Type?**
A: Type — конкретный тип (UserID). Kind — базовый вид (int64). `type UserID int64` имеет Kind=Int64, Type=UserID.
