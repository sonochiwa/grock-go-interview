package min_max

import (
	"cmp"
	"errors"
)

var ErrEmptySlice = errors.New("empty slice")

func Min[T cmp.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func Max[T cmp.Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

func MinSlice[T cmp.Ordered](s []T) (T, error) {
	var zero T
	if len(s) == 0 {
		return zero, ErrEmptySlice
	}

	min := s[0]
	for _, v := range s[1:] {
		if v < min {
			min = v
		}
	}
	return min, nil
}
