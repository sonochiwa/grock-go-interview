# Map Internals: Classic (до Go 1.24)

## Обзор

До Go 1.24 Go использовал bucket-based хеш-таблицу. Понимание этой реализации важно — вопросы о ней всё ещё задают на собесах.

## Структура

```go
// runtime/map.go (упрощённо)
type hmap struct {
    count     int            // количество элементов
    flags     uint8          // состояние (iterator, writing)
    B         uint8          // log2(кол-во bucket) — 2^B bucket'ов
    noverflow uint16         // приблизительное кол-во overflow buckets
    hash0     uint32         // seed хеш-функции (рандомный)
    buckets   unsafe.Pointer // массив 2^B bucket'ов
    oldbuckets unsafe.Pointer // при evacuation — старые bucket'ы
    nevacuate  uintptr       // прогресс эвакуации
    extra      *mapextra     // overflow bucket'ы
}
```

### Bucket

```go
// runtime/map.go
type bmap struct {
    tophash [8]uint8 // верхние 8 бит хеша для каждого слота
    // За ним в памяти (компилятор генерирует):
    // keys   [8]keyType
    // values [8]valueType
    // overflow *bmap
}
```

```
Bucket (8 слотов):
┌─────────────────────────────────────────┐
│ tophash: [h0][h1][h2][h3][h4][h5][h6][h7] │
│ keys:    [k0][k1][k2][k3][k4][k5][k6][k7] │
│ values:  [v0][v1][v2][v3][v4][v5][v6][v7] │
│ overflow: *bmap ──► (следующий bucket)      │
└─────────────────────────────────────────┘
```

**Почему ключи и значения хранятся отдельно?** Чтобы избежать padding. Для `map[int8]int64`: без разделения — 7 байт padding на каждую пару. С разделением — 0.

### Lookup

1. Вычислить хеш ключа: `hash(key, hash0)`
2. Нижние B бит → номер bucket
3. Верхние 8 бит → tophash для быстрого сравнения
4. Перебрать 8 слотов bucket, сравнивая tophash
5. При совпадении tophash — сравнить полный ключ
6. Если не нашли — проверить overflow bucket

### Growing (рост)

Условие роста: **load factor > 6.5** (в среднем 6.5 элементов на bucket)

```
Рост × 2 (incremental evacuation):
1. Создать новый массив bucket'ов размером 2^(B+1)
2. НЕ копировать сразу — evacuation при каждом доступе
3. При каждой записи/удалении — эвакуировать 1-2 старых bucket
4. Когда все эвакуированы — освободить старые
```

Также есть **same-size grow** при слишком большом количестве overflow buckets (без увеличения размера — просто уплотнение).

## Почему итерация рандомная

```go
// При начале range:
// 1. Выбирается случайный стартовый bucket
// 2. Выбирается случайный offset внутри bucket
// Это НАМЕРЕННО с Go 1.12, чтобы разработчики не зависели от порядка
```

## Конкурентный доступ

```go
// hmap.flags содержит бит hashWriting
// При записи: устанавливается hashWriting
// При чтении: если hashWriting установлен → fatal("concurrent map read and map write")
// Это НЕ race detector — это проверка в runtime
```

## Частые вопросы на собеседованиях

**Q: Какой load factor у Go map?**
A: 6.5 (в среднем 6.5 элементов на bucket из 8 слотов).

**Q: Почему рост инкрементальный?**
A: Чтобы избежать длинных пауз. Эвакуация распределена по операциям.

**Q: Почему нельзя взять адрес элемента map?**
A: При росте элементы перемещаются в новые bucket'ы. Указатель стал бы невалидным.
