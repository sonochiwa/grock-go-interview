package unique_elements

// Unique возвращает слайс только с уникальными элементами,
// сохраняя порядок первого появления.
func Unique[T comparable](s []T) []T {
	if s == nil {
		return nil
	}

	seen := make(map[T]struct{}, len(s))
	result := make([]T, 0, len(s))

	for _, v := range s {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}

	return result
}
