package custom_error

import "fmt"

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation: %s: %s", e.Field, e.Message)
}

func ValidateAge(age int) error {
	if age < 0 {
		return &ValidationError{Field: "age", Message: "must be non-negative"}
	}
	if age > 150 {
		return &ValidationError{Field: "age", Message: "must be <= 150"}
	}
	return nil
}
