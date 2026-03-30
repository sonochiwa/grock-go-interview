# Строки

## Обзор

Строка в Go — неизменяемая последовательность байт. Не символов, не рун — именно байт. Понимание этого различия критично для работы с Unicode и оптимизации.

## Концепции

### Строка = []byte (неизменяемый)

```go
s := "Hello, 世界"
fmt.Println(len(s))    // 13 (байт, не символов!)
fmt.Println(len([]rune(s))) // 9 (рун/символов)

// Обращение по индексу — байт, не символ
fmt.Println(s[0])  // 72 (byte 'H')
fmt.Println(s[7])  // 228 (первый байт '世', не сам символ)
```

### Руны и UTF-8

```go
// rune = int32 = Unicode code point
// Go использует UTF-8 кодировку

s := "Hello, 世界"

// range по строке итерирует РУНЫ, не байты!
for i, r := range s {
    fmt.Printf("byte_offset=%d rune=%c unicode=U+%04X\n", i, r, r)
}
// byte_offset=0 rune=H unicode=U+0048
// byte_offset=7 rune=世 unicode=U+4E16  ← занимает 3 байта
// byte_offset=10 rune=界 unicode=U+754C

// Прямая итерация по байтам
for i := 0; i < len(s); i++ {
    fmt.Printf("%d: byte=%d\n", i, s[i])
}

// Количество рун
utf8.RuneCountInString(s) // 9
```

### Конкатенация строк

```go
// Оператор + (создаёт новую строку каждый раз)
s := "hello" + " " + "world" // 2 аллокации

// fmt.Sprintf — удобно, но медленно
s := fmt.Sprintf("%s %s", "hello", "world")

// strings.Builder — оптимально для множественной конкатенации
var b strings.Builder
b.Grow(100) // предварительная аллокация (опционально)
for i := 0; i < 1000; i++ {
    b.WriteString("hello ")
}
result := b.String()

// strings.Join — для слайса строк
parts := []string{"hello", "world"}
s := strings.Join(parts, " ") // "hello world"
```

### Конвертация string ↔ []byte

```go
s := "hello"
b := []byte(s)   // КОПИРОВАНИЕ! Создаёт новый []byte
s2 := string(b)  // КОПИРОВАНИЕ! Создаёт новую строку

// Почему копирование? Строка неизменяема, []byte — изменяем.
// Без копирования можно было бы нарушить иммутабельность.

// Оптимизации компилятора (НЕ копируют):
// 1. Сравнение: string(b) == "hello"
// 2. Range: for i, r := range []byte(s)
// 3. Map lookup: m[string(b)]
// 4. Конкатенация: "prefix" + string(b) + "suffix"
```

## Под капотом

### StringHeader

```go
// reflect/value.go (устаревшая, но показательная структура)
type StringHeader struct {
    Data uintptr // указатель на байты
    Len  int     // длина в байтах
}
// Размер: 16 байт на 64-bit системе

// Строка — это просто {указатель, длина}
// Нет нуль-терминатора как в C!
// Нет поля capacity — строка неизменяема
```

### Иммутабельность

```go
s := "hello"
// s[0] = 'H' // ОШИБКА: cannot assign to s[0]

// Строки можно "изменять" только создавая новые:
s = "H" + s[1:] // "Hello" — новая строка

// unsafe хак (НЕ делай так в продакшене):
b := unsafe.Slice(unsafe.StringData(s), len(s))
// b[0] = 'H' // undefined behavior! Может крашнуть программу
```

### String interning

```go
// Компилятор может переиспользовать одинаковые строковые литералы
s1 := "hello"
s2 := "hello"
// s1 и s2 МОГУТ указывать на одни и те же байты (compiler optimization)
// Но это не гарантировано — нельзя на это полагаться

// unique.Handle (Go 1.23+) — явный interning
import "unique"
h1 := unique.Make("hello")
h2 := unique.Make("hello")
// h1 == h2 (true) — гарантированно один объект
```

## Частые вопросы на собеседованиях

**Q: Чему равен len("Привет")?**
A: 12. Каждая кириллическая буква в UTF-8 занимает 2 байта. len() возвращает длину в байтах, не в символах.

**Q: Как правильно конкатенировать строки в цикле?**
A: strings.Builder. Каждый + создаёт новую строку и копирует данные → O(n²). Builder накапливает в буфере → O(n).

**Q: Строка передаётся в функцию по значению или по ссылке?**
A: По значению, но копируется только заголовок (16 байт: указатель + длина). Сами байты НЕ копируются. Строка — read-only view.

**Q: Почему string ↔ []byte требует копирования?**
A: Строка неизменяема. Если бы []byte и string разделяли память, изменение []byte нарушило бы инвариант иммутабельности строки.

## Подводные камни

1. **Подстрока удерживает оригинал в памяти**:
```go
// s[:10] разделяет backing array с s
huge := loadHugeString() // 1MB
small := huge[:10] // small удерживает 1MB!

// Решение:
small := strings.Clone(huge[:10]) // Go 1.20+
// или
small := string([]byte(huge[:10]))
```

2. **Невалидный UTF-8**:
```go
s := "\xff\xfe" // невалидный UTF-8
for _, r := range s {
    // r == U+FFFD (replacement character) для невалидных байт
}
// Проверка: utf8.ValidString(s)
```

3. **Сравнение строк**: == сравнивает побайтово (не Unicode normalization). "café" (с combining accent) != "café" (с precomposed é).
