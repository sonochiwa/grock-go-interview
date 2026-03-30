# stringer

## Обзор

Генерирует метод `String()` для iota-констант.

```bash
go install golang.org/x/tools/cmd/stringer@latest
```

```go
//go:generate stringer -type=Weekday

type Weekday int
const (
    Sunday Weekday = iota
    Monday
    Tuesday
    // ...
)

// Сгенерирует weekday_string.go:
// func (w Weekday) String() string { ... }

fmt.Println(Monday) // "Monday" (вместо "1")
```
