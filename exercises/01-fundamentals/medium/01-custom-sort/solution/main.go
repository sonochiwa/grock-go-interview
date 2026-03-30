package custom_sort

import (
	"fmt"
	"reflect"
	"sort"
)

// SortBy сортирует копию слайса используя переданную функцию сравнения.
func SortBy[T any](s []T, less func(a, b T) bool) []T {
	if len(s) == 0 {
		return []T{}
	}

	result := make([]T, len(s))
	copy(result, s)

	sort.Slice(result, func(i, j int) bool {
		return less(result[i], result[j])
	})

	return result
}

// SortByField сортирует копию слайса структур по имени поля используя reflect.
func SortByField[T any](s []T, fieldName string, ascending bool) ([]T, error) {
	if len(s) == 0 {
		return []T{}, nil
	}

	// Проверяем что поле существует
	t := reflect.TypeOf(s[0])
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	field, found := t.FieldByName(fieldName)
	if !found {
		return nil, fmt.Errorf("field %q not found in type %s", fieldName, t.Name())
	}

	// Проверяем поддерживаемые типы
	switch field.Type.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
	case reflect.Float32, reflect.Float64:
	case reflect.String:
	default:
		return nil, fmt.Errorf("unsupported field type %s for sorting", field.Type.Kind())
	}

	result := make([]T, len(s))
	copy(result, s)

	sort.Slice(result, func(i, j int) bool {
		vi := reflect.ValueOf(result[i]).FieldByName(fieldName)
		vj := reflect.ValueOf(result[j]).FieldByName(fieldName)

		var less bool
		switch field.Type.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			less = vi.Int() < vj.Int()
		case reflect.Float32, reflect.Float64:
			less = vi.Float() < vj.Float()
		case reflect.String:
			less = vi.String() < vj.String()
		}

		if ascending {
			return less
		}
		return !less
	})

	return result, nil
}
