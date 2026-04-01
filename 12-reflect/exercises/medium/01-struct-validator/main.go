package struct_validator

type ValidationError struct {
	Field   string
	Tag     string
	Message string
}

// TODO: используй reflect для чтения struct tags и валидации полей
// Поддержи: required, min=N, max=N
// Для string: min/max проверяют len
// Для int: min/max проверяют значение
func Validate(v any) []ValidationError {
	return nil
}
