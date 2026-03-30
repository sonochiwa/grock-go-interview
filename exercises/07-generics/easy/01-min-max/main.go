package min_max

import (
	"cmp"
	"errors"
)

var ErrEmptySlice = errors.New("empty slice")

// TODO: реализуй generic функцию Min, возвращающую меньшее из двух значений.
func Min[T cmp.Ordered](a, b T) T {
	panic("not implemented")
}

// TODO: реализуй generic функцию Max, возвращающую большее из двух значений.
func Max[T cmp.Ordered](a, b T) T {
	panic("not implemented")
}

// TODO: реализуй MinSlice — возвращает минимальный элемент слайса.
// Для пустого слайса верни zero value и ErrEmptySlice.
func MinSlice[T cmp.Ordered](s []T) (T, error) {
	panic("not implemented")
}
