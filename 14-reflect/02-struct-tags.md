# Struct Tags

## Обзор

Struct tags — метаданные полей, доступные через reflect. Используются для JSON, DB, валидации и др.

```go
type User struct {
    ID    int64  `json:"id" db:"id" validate:"required"`
    Name  string `json:"name" db:"user_name" validate:"required,min=2"`
    Email string `json:"email,omitempty" db:"email" validate:"required,email"`
}

// Парсинг тега
t := reflect.TypeOf(User{})
field, _ := t.FieldByName("Email")
jsonTag := field.Tag.Get("json")   // "email,omitempty"
dbTag := field.Tag.Get("db")       // "email"
valTag := field.Tag.Get("validate") // "required,email"

// Полный тег
field.Tag // `json:"email,omitempty" db:"email" validate:"required,email"`

// Lookup с проверкой наличия
val, ok := field.Tag.Lookup("json") // val="email,omitempty", ok=true
val, ok = field.Tag.Lookup("xml")   // val="", ok=false
```

### Формат тегов

```go
// Формат: `key1:"value1" key2:"value2"`
// Ключ: без кавычек, алфавитно-цифровой
// Значение: в двойных кавычках
// Разделитель: пробел

// Стандартные конвенции для json:
`json:"-"`           // игнорировать поле
`json:"name"`        // имя в JSON
`json:"name,omitempty"` // пропустить если zero value
`json:",omitempty"`  // имя по умолчанию + omitempty
```

### Практический пример: простой валидатор

```go
func Validate(v any) error {
    val := reflect.ValueOf(v)
    typ := val.Type()

    for i := 0; i < typ.NumField(); i++ {
        field := typ.Field(i)
        tag := field.Tag.Get("validate")
        if tag == "" {
            continue
        }

        fieldVal := val.Field(i)
        rules := strings.Split(tag, ",")

        for _, rule := range rules {
            switch rule {
            case "required":
                if fieldVal.IsZero() {
                    return fmt.Errorf("field %s is required", field.Name)
                }
            }
        }
    }
    return nil
}
```
