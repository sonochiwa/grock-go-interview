# Memory Allocator

## Обзор

Go использует собственный аллокатор памяти (основан на TCMalloc). Многоуровневая архитектура минимизирует lock contention.

## Архитектура

```
Горутина → mcache (per-P, no lock) → mcentral (per-size, lock) → mheap (global, lock) → OS
```

### mcache (per-P, без блокировок)

```go
// Каждый P имеет свой mcache
type mcache struct {
    alloc [numSpanClasses]*mspan // по одному span на каждый size class
}

// Size classes: ~70 классов от 8 до 32768 байт
// Аллокация: взять свободный объект из span нужного size class
// Нет мьютекса! Только P владеет своим mcache
```

### mcentral (per-size-class)

```go
// Один mcentral на каждый size class
type mcentral struct {
    spanclass spanClass
    partial   [2]spanSet // span'ы с свободными объектами
    full      [2]spanSet // полностью занятые span'ы
}

// Когда mcache пуст → запросить span у mcentral
// Мьютекс на каждый size class (гранулярный лок)
```

### mheap (глобальный)

```go
// Управляет страницами памяти
type mheap struct {
    pages pageAlloc // аллокатор страниц
    // ...
}

// Когда mcentral нужен новый span → запросить у mheap
// mheap запрашивает память у ОС через mmap/VirtualAlloc
```

## Аллокация объекта

```
1. Размер ≤ 16 байт (без указателей) → tiny allocator
   - Несколько маленьких объектов в одном блоке
   - Экономит память для мелких аллокаций

2. Размер ≤ 32 КБ → mcache → mcentral → mheap
   - Выбрать size class (округление вверх)
   - Взять объект из mcache span
   - Если span пуст → запросить у mcentral
   - Если mcentral пуст → запросить у mheap

3. Размер > 32 КБ → напрямую из mheap
   - Large object allocation
   - Отдельные страницы
```

### Size Classes (примеры)

```
Class  Size    Объектов/span
  1      8      512
  2     16      256
  3     24      170
  4     32      128
  ...
 66   28672      1
 67   32768      1
```

## Стек vs Куча

```go
// Escape analysis определяет размещение
func noEscape() int {
    x := 42     // стек (не убегает)
    return x
}

func escape() *int {
    x := 42     // куча (указатель убегает)
    return &x
}

// Проверка:
// go build -gcflags="-m" .
```

## Частые вопросы на собеседованиях

**Q: Как устроен аллокатор Go?**
A: Трёхуровневый: mcache (per-P, no lock) → mcentral (per-size, granular lock) → mheap (global). Основан на TCMalloc.

**Q: Что такое size class?**
A: Фиксированные размеры блоков (~70 классов). Объект аллоцируется в ближайшем бОльшем size class. Устраняет фрагментацию.

**Q: Почему mcache без мьютекса?**
A: Каждый P имеет свой mcache. Только одна горутина одновременно работает на P → нет конкуренции.

**Q: Что такое tiny allocator?**
A: Объекты ≤16 байт без указателей пакуются вместе в один блок. Экономит память для string headers, small ints, etc.
