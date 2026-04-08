# Interface Internals (подробно)

Этот файл дополняет 02-interfaces/05-interface-internals.md с фокусом на реализацию.

## Converson: concrete → interface

```go
var s Stringer = myValue
// Компилятор генерирует:
// 1. Найти/создать itab для (Stringer, *MyType) — кешируется
// 2. Если myValue помещается в pointer — inline
// 3. Иначе: аллоцировать в куче, сохранить указатель в data
```

## Method dispatch

```go
s.String()
// Скомпилируется в:
// MOVQ 24(AX), CX   // загрузить itab.fun[0]
// MOVQ 8(AX), DX    // загрузить data pointer
// CALL CX            // вызвать метод с data как receiver
```

## itab кеш

```go
// runtime/iface.go
// Глобальная хеш-таблица itab'ов
// Ключ: (inter *interfacetype, _type *_type)
// При первом создании: линейный поиск методов O(n*m)
// Далее: O(1) из кеша
```

## Стоимость

- Создание interface: поиск/создание itab + возможная аллокация data
- Вызов метода: один indirect call (~1-2ns overhead)
- Type assertion к конкретному типу: сравнение pointer — O(1)
- Type assertion к интерфейсу: поиск itab — O(1) из кеша

Подробнее о nil interface проблеме — см. 02-interfaces/05-interface-internals.md.
