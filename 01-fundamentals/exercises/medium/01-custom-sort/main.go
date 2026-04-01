package custom_sort

// SortBy сортирует копию слайса используя переданную функцию сравнения.
// Исходный слайс не должен изменяться.
// TODO: реализуй функцию
func SortBy[T any](s []T, less func(a, b T) bool) []T {
	return nil
}

// SortByField сортирует копию слайса структур по имени поля используя reflect.
// Возвращает ошибку если поле не найдено или тип поля не поддерживается.
// Поддерживаемые типы полей: int, float64, string.
// TODO: реализуй функцию
func SortByField[T any](s []T, fieldName string, ascending bool) ([]T, error) {
	return nil, nil
}
