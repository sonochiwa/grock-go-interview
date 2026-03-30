# Fuzzing (Go 1.18+)

## Обзор

```
Fuzz testing = автоматическая генерация входных данных для поиска крашей/багов
Встроен в Go с 1.18. Мутирует seed inputs для нахождения edge cases.

Находит:
  - Panic / nil pointer
  - Index out of range
  - Бесконечные циклы
  - Некорректную обработку невалидных данных
  - Security уязвимости (buffer overflow, injection)
```

## Базовый пример

```go
func FuzzParseJSON(f *testing.F) {
    // Seed corpus — стартовые данные для мутации
    f.Add([]byte(`{"name":"Alice","age":30}`))
    f.Add([]byte(`{}`))
    f.Add([]byte(`[]`))
    f.Add([]byte(`null`))
    f.Add([]byte(`""`))

    f.Fuzz(func(t *testing.T, data []byte) {
        var user User
        err := json.Unmarshal(data, &user)
        if err != nil {
            return // невалидный JSON — OK, не должен паниковать
        }

        // Round-trip: marshal → unmarshal → должно совпасть
        encoded, err := json.Marshal(user)
        if err != nil {
            t.Fatalf("marshal failed for valid input: %v", err)
        }

        var user2 User
        if err := json.Unmarshal(encoded, &user2); err != nil {
            t.Fatalf("round-trip failed: %v", err)
        }
    })
}
```

```bash
# Запуск fuzzing (работает бесконечно, пока не найдёт баг или Ctrl+C)
go test -fuzz=FuzzParseJSON -fuzztime=30s

# Найденные крашащие inputs сохраняются в testdata/fuzz/FuzzParseJSON/
# и автоматически прогоняются при обычном go test
```

## Поддерживаемые типы

```go
// f.Add() и f.Fuzz() поддерживают:
// string, []byte, int, int8/16/32/64, uint/8/16/32/64, float32/64, bool, rune

f.Add("hello", 42, true, 3.14)
f.Fuzz(func(t *testing.T, s string, n int, b bool, f float64) {
    // ...
})
```

## Практический пример: URL parser

```go
func FuzzParseURL(f *testing.F) {
    f.Add("https://example.com/path?key=value")
    f.Add("http://localhost:8080")
    f.Add("")
    f.Add("not-a-url")

    f.Fuzz(func(t *testing.T, input string) {
        u, err := ParseURL(input)
        if err != nil {
            return // невалидный URL — ок
        }

        // Если парсинг успешен — результат должен быть валидным
        if u.Host == "" {
            t.Error("parsed URL has empty host")
        }

        // Строковое представление должно быть парсабельным
        str := u.String()
        u2, err := ParseURL(str)
        if err != nil {
            t.Errorf("round-trip failed: %q → %q → error: %v", input, str, err)
        }
        if u2.Host != u.Host {
            t.Errorf("host mismatch: %q vs %q", u.Host, u2.Host)
        }
    })
}
```

## Corpus management

```
testdata/fuzz/FuzzParseJSON/
├── seed/              ← f.Add() данные (опционально, можно файлами)
│   └── input1
├── corpus_hash1       ← найденные fuzzer'ом интересные inputs
└── corpus_hash2       ← крашащие inputs (сохраняются автоматически)

# Файл формат:
go test fuzz v1
[]byte("crash input here")
```

## Best Practices

```
1. Не проверяй конкретные значения — проверяй инварианты:
   ❌ assert.Equal(t, expected, result)
   ✅ if err == nil, result должен быть валидным
   ✅ round-trip: encode(decode(x)) == x
   ✅ не паникует на любом input

2. Fuzz что:
   - Парсеры (JSON, XML, protobuf, custom formats)
   - Валидаторы
   - Encoder/Decoder
   - Anything that takes []byte/string input

3. Seed corpus — важен:
   - Добавь валидные примеры
   - Добавь edge cases (пустая строка, unicode, очень длинный input)
   - Больше seeds → быстрее найдёт баги

4. CI: используй -fuzztime=30s (не бесконечно)
```
