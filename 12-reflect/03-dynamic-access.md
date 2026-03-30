# Динамический доступ

## Чтение/запись полей

```go
user := &User{Name: "Alice", Age: 25}
v := reflect.ValueOf(user).Elem()

// Чтение
name := v.FieldByName("Name").String() // "Alice"

// Запись (только через указатель!)
v.FieldByName("Name").SetString("Bob")

// Неэкспортированные поля — нельзя SetXxx
// v.FieldByName("secret").SetString("x") // PANIC
```

## Вызов методов

```go
v := reflect.ValueOf(user)
method := v.MethodByName("Greet")
results := method.Call([]reflect.Value{reflect.ValueOf("Hello")})
fmt.Println(results[0].String())
```

## Создание и работа с map/slice

```go
// Map
m := reflect.MakeMap(reflect.TypeOf(map[string]int{}))
m.SetMapIndex(reflect.ValueOf("key"), reflect.ValueOf(42))

// Slice
s := reflect.MakeSlice(reflect.TypeOf([]int{}), 0, 10)
s = reflect.Append(s, reflect.ValueOf(1), reflect.ValueOf(2))
```
