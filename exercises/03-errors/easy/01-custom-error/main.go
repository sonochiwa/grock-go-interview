package custom_error

// TODO: определи тип ValidationError с полями Field и Message
// TODO: реализуй метод Error() string → "validation: <Field>: <Message>"

type ValidationError struct {
	Field   string
	Message string
}

// TODO: реализуй ValidateAge
func ValidateAge(age int) error {
	return nil
}
