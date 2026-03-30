# Производительность reflect

## Бенчмарки

```
Direct field access:     ~0.3 ns/op
reflect.ValueOf + Field: ~50  ns/op  (150x медленнее)
reflect FieldByName:     ~200 ns/op  (600x медленнее)
```

## Оптимизации

```go
// 1. Кешируй Type и FieldIndex
var userType = reflect.TypeOf(User{})
nameIdx, _ := userType.FieldByName("Name") // один раз
// Далее: val.Field(nameIdx.Index[0]) вместо val.FieldByName("Name")

// 2. Используй unsafe вместо reflect (только если критично)
// unsafe.Offsetof — прямой доступ к полю по смещению

// 3. Кодогенерация вместо reflect
// go generate + шаблоны для маршалинга, валидации
// easyjson, ffjson — генерируют код вместо reflect

// 4. Дженерики вместо reflect (Go 1.18+)
// Если тип известен в compile time — дженерики быстрее
```

## Когда reflect приемлем

- Инициализация (один раз при старте)
- Не на hot path (не в каждом HTTP запросе)
- Когда альтернативы (дженерики, кодоген) невозможны

## Когда избегать reflect

- Hot path (каждый запрос, каждая итерация)
- Когда тип известен в compile time
- Когда можно заменить кодогенерацией
