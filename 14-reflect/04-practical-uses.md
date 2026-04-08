# Практические применения reflect

## Где reflect используется

- `encoding/json` — маршалинг/анмаршалинг
- `database/sql` — сканирование строк
- `fmt.Printf` — форматирование %v
- Валидаторы (`validator` пакет)
- ORM (GORM, sqlx)
- Dependency injection (wire, dig)
- Testing (testify assertions)

## Пример: struct → map

```go
func StructToMap(v any) map[string]any {
    result := make(map[string]any)
    val := reflect.ValueOf(v)
    typ := val.Type()

    if val.Kind() == reflect.Ptr {
        val = val.Elem()
        typ = val.Type()
    }

    for i := 0; i < val.NumField(); i++ {
        field := typ.Field(i)
        if !field.IsExported() {
            continue
        }
        result[field.Name] = val.Field(i).Interface()
    }
    return result
}
```

## Пример: DeepEqual

```go
// reflect.DeepEqual — глубокое сравнение (включая слайсы, мапы)
a := []int{1, 2, 3}
b := []int{1, 2, 3}
reflect.DeepEqual(a, b) // true (== не работает для слайсов)

// Осторожно: DeepEqual считает nil slice != empty slice
reflect.DeepEqual([]int(nil), []int{}) // false!
```
