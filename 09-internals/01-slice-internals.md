# Внутреннее устройство слайсов

## Структура

```go
// runtime/slice.go
type slice struct {
    array unsafe.Pointer // указатель на первый элемент backing array
    len   int            // текущее количество элементов
    cap   int            // ёмкость (элементов до конца backing array)
}
// 24 байта на 64-bit
```

```
slice{array, len=3, cap=5}
  │
  ▼
┌───┬───┬───┬───┬───┐
│ 1 │ 2 │ 3 │   │   │   backing array
└───┴───┴───┴───┴───┘
  0   1   2   3   4
        len─┘     cap─┘
```

## Алгоритм роста (Go 1.18+)

```go
// runtime/slice.go: growslice
func growslice(oldPtr unsafe.Pointer, newLen, oldCap, num int, et *_type) slice {
    newcap := nextslicecap(newLen, oldCap)
    // Затем округляем до size class аллокатора
}

func nextslicecap(newLen, oldCap int) int {
    newcap := oldCap
    doublecap := newcap + newcap
    if newLen > doublecap {
        return newLen
    }

    const threshold = 256
    if oldCap < threshold {
        return doublecap // маленькие: ×2
    }
    // Плавный рост
    for {
        newcap += (newcap + 3*threshold) >> 2 // ≈ 1.25x + const
        if newcap >= newLen {
            return newcap
        }
    }
}
```

**Ключевое изменение с Go 1.18:** убран резкий переход с ×2 на ×1.25 при cap=1024. Теперь плавный переход через threshold=256.

## Операции

### append

1. Если `len + n <= cap` → просто копируем, увеличиваем len
2. Если `len + n > cap` → growslice: аллоцируем новый array, копируем, возвращаем новый slice header

### Subslice `s[i:j]`

Новый slice header с тем же backing array:
```
s = array[0:5:5]  →  s[1:3] = {array+1, len=2, cap=4}
// РАЗДЕЛЯЮТ backing array!
```

### copy

```go
// Копирует min(len(dst), len(src)) элементов
// Backing arrays не связаны
n := copy(dst, src)
```

## Частые вопросы на собеседованиях

**Q: Что происходит при append если cap достаточен?**
A: Элемент записывается в backing array, len увеличивается. Новый массив НЕ создаётся. Другие слайсы на тот же массив увидят изменение.

**Q: Когда два слайса разделяют backing array?**
A: При subslice (s[i:j]) и при append без роста. После append с ростом — новый backing array.
