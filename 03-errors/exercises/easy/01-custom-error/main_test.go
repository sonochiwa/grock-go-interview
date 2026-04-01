package custom_error

import (
	"errors"
	"testing"
)

func TestValidateAge(t *testing.T) {
	tests := []struct {
		name    string
		age     int
		wantErr bool
		field   string
		msg     string
	}{
		{"valid 25", 25, false, "", ""},
		{"valid 0", 0, false, "", ""},
		{"valid 150", 150, false, "", ""},
		{"negative", -1, true, "age", "must be non-negative"},
		{"too old", 151, true, "age", "must be <= 150"},
		{"very negative", -100, true, "age", "must be non-negative"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAge(tt.age)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateAge(%d) error = %v, wantErr %v", tt.age, err, tt.wantErr)
			}
			if tt.wantErr {
				var ve *ValidationError
				if !errors.As(err, &ve) {
					t.Fatalf("ValidateAge(%d) error is not *ValidationError: %T", tt.age, err)
				}
				if ve.Field != tt.field {
					t.Errorf("Field = %q, want %q", ve.Field, tt.field)
				}
				if ve.Message != tt.msg {
					t.Errorf("Message = %q, want %q", ve.Message, tt.msg)
				}
			}
		})
	}
}

func TestValidationErrorString(t *testing.T) {
	err := &ValidationError{Field: "age", Message: "must be non-negative"}
	want := "validation: age: must be non-negative"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
