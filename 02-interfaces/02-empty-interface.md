# Пустой интерфейс (any)

## Обзор

`any` (алиас для `interface{}`) — интерфейс без методов. Любой тип его реализует. Использовать осторожно — теряется типобезопасность.

## Концепции

```go
// any = interface{} (алиас с Go 1.18)
var x any

x = 42
x = "hello"
x = []int{1, 2, 3}
x = struct{ Name string }{"Go"}

// Используется в:
// - fmt.Println(a ...any)
// - json.Marshal(v any)
// - context.WithValue(parent, key, val any)

// Чтобы достать значение — type assertion или type switch
s, ok := x.(string)
```

### Когда использовать

| Используй any | НЕ используй any |
|---|---|
| Сериализация (JSON, gob) | Когда можно использовать дженерики |
| context.Value | Когда типы известны на этапе компиляции |
| Логирование/отладка | В публичном API без необходимости |

### Стоимость any

```go
// Упаковка в interface{} МОЖЕТ вызвать аллокацию в куче
func printVal(v any) { fmt.Println(v) }

x := 42
printVal(x) // x упаковывается в interface{} → может escape в кучу

// Оптимизация: маленькие значения (≤ pointer size) хранятся inline
// Это implementation detail — не полагайся на это
```

## Частые вопросы на собеседованиях

**Q: Чем any отличается от interface{}?**
A: Ничем. `any` — это type alias для `interface{}`, введённый в Go 1.18 для читаемости.

**Q: Когда использовать any вместо дженериков?**
A: Когда тип неизвестен в compile time (JSON, reflection). Дженерики сохраняют типобезопасность — предпочитай их.

**Q: Есть ли overhead у any?**
A: Да. Значение упаковывается в интерфейс (2 слова: тип + данные). Для маленьких значений компилятор может оптимизировать.

## Подводные камни

1. **Потеря типобезопасности** — ошибки обнаруживаются в runtime, не в compile time.

2. **Сравнение any**: два `any` значения сравнимы через `==` только если underlying type сравним. Если underlying type — slice, будет **panic**:
```go
var a, b any = []int{1}, []int{1}
// a == b // PANIC: comparing uncomparable type []int
```
