# Table-Driven Tests

## Обзор

Идиоматичный Go подход к тестированию: таблица тест-кейсов + цикл. Чисто, расширяемо, самодокументируемо.

```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"positive", 2, 3, 5},
        {"zero", 0, 0, 0},
        {"negative", -1, -2, -3},
        {"mixed", -1, 5, 4},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Add(tt.a, tt.b)
            if result != tt.expected {
                t.Errorf("Add(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
            }
        })
    }
}
```

### С ошибками

```go
func TestParse(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    int
        wantErr bool
    }{
        {"valid", "42", 42, false},
        {"invalid", "abc", 0, true},
        {"empty", "", 0, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Parse(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("Parse(%q) = %d, want %d", tt.input, got, tt.want)
            }
        })
    }
}
```

### Параллельные table-driven тесты

```go
for _, tt := range tests {
    tt := tt // до Go 1.22
    t.Run(tt.name, func(t *testing.T) {
        t.Parallel() // запуск параллельно
        // ...
    })
}
```

### Почему именно этот стиль

- Легко добавить новый кейс (одна строка)
- `t.Run` — каждый кейс как подтест (можно запускать отдельно)
- name поле — документирует что тестируем
- Один assert — понятно что сломалось
