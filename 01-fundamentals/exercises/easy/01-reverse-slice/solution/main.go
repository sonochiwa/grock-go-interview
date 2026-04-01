package reverse_slice

// ReverseSlice возвращает новый слайс с элементами в обратном порядке.
// Исходный слайс не должен изменяться.
func ReverseSlice[T any](s []T) []T {
	if s == nil {
		return nil
	}

	result := make([]T, len(s))
	for i, v := range s {
		result[len(s)-1-i] = v
	}
	return result
}
