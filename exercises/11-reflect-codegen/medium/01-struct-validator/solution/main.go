package struct_validator

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type ValidationError struct {
	Field   string
	Tag     string
	Message string
}

func Validate(v any) []ValidationError {
	var errs []ValidationError
	val := reflect.ValueOf(v)
	typ := val.Type()

	for i := range val.NumField() {
		field := typ.Field(i)
		fval := val.Field(i)
		tag := field.Tag.Get("validate")
		if tag == "" {
			continue
		}

		rules := strings.Split(tag, ",")
		for _, rule := range rules {
			if rule == "required" {
				if fval.IsZero() {
					errs = append(errs, ValidationError{
						Field: field.Name, Tag: "required",
						Message: fmt.Sprintf("%s is required", field.Name),
					})
				}
				continue
			}

			if strings.HasPrefix(rule, "min=") {
				n, _ := strconv.Atoi(strings.TrimPrefix(rule, "min="))
				switch fval.Kind() {
				case reflect.String:
					if fval.Len() < n {
						errs = append(errs, ValidationError{
							Field: field.Name, Tag: "min",
							Message: fmt.Sprintf("%s must be at least %d characters", field.Name, n),
						})
					}
				case reflect.Int, reflect.Int64:
					if fval.Int() < int64(n) {
						errs = append(errs, ValidationError{
							Field: field.Name, Tag: "min",
							Message: fmt.Sprintf("%s must be >= %d", field.Name, n),
						})
					}
				}
			}

			if strings.HasPrefix(rule, "max=") {
				n, _ := strconv.Atoi(strings.TrimPrefix(rule, "max="))
				switch fval.Kind() {
				case reflect.String:
					if fval.Len() > n {
						errs = append(errs, ValidationError{
							Field: field.Name, Tag: "max",
							Message: fmt.Sprintf("%s must be at most %d characters", field.Name, n),
						})
					}
				case reflect.Int, reflect.Int64:
					if fval.Int() > int64(n) {
						errs = append(errs, ValidationError{
							Field: field.Name, Tag: "max",
							Message: fmt.Sprintf("%s must be <= %d", field.Name, n),
						})
					}
				}
			}
		}
	}
	return errs
}
