package struct_validator

import "testing"

type User struct {
	Name  string `validate:"required,min=2,max=50"`
	Email string `validate:"required"`
	Age   int    `validate:"min=0,max=150"`
}

func TestValid(t *testing.T) {
	u := User{Name: "Alice", Email: "alice@test.com", Age: 25}
	errs := Validate(u)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestRequired(t *testing.T) {
	u := User{Name: "", Email: "test@test.com", Age: 25}
	errs := Validate(u)
	found := false
	for _, e := range errs {
		if e.Field == "Name" && e.Tag == "required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected required error for Name, got %v", errs)
	}
}

func TestMinMax(t *testing.T) {
	u := User{Name: "A", Email: "test@test.com", Age: 200}
	errs := Validate(u)
	var nameMin, ageMax bool
	for _, e := range errs {
		if e.Field == "Name" && e.Tag == "min" {
			nameMin = true
		}
		if e.Field == "Age" && e.Tag == "max" {
			ageMax = true
		}
	}
	if !nameMin {
		t.Error("expected min error for Name")
	}
	if !ageMax {
		t.Error("expected max error for Age")
	}
}

func TestNegativeAge(t *testing.T) {
	u := User{Name: "Bob", Email: "bob@test.com", Age: -5}
	errs := Validate(u)
	found := false
	for _, e := range errs {
		if e.Field == "Age" && e.Tag == "min" {
			found = true
		}
	}
	if !found {
		t.Error("expected min error for negative Age")
	}
}
