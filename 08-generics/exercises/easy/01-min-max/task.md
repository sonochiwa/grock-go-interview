# Min / Max

## Задание

Напиши generic функции `Min[T cmp.Ordered](a, b T) T` и `Max[T cmp.Ordered](a, b T) T`.

Также напиши `MinSlice[T cmp.Ordered](s []T) (T, error)` — возвращает минимальный элемент слайса
или ошибку для пустого слайса.

## Требования

- Используй constraint `cmp.Ordered`
- `MinSlice` возвращает ошибку `ErrEmptySlice` для пустого слайса
- Функции должны работать с `int`, `float64`, `string`

## Запуск тестов

```bash
go test -v ./...
```
