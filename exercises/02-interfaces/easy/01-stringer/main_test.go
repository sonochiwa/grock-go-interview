package stringer

import (
	"fmt"
	"testing"
)

func TestMoneyString(t *testing.T) {
	tests := []struct {
		name string
		m    Money
		want string
	}{
		{"basic USD", Money{100.50, "USD"}, "100.50 USD"},
		{"zero", Money{0, "EUR"}, "0.00 EUR"},
		{"integer amount", Money{42, "RUB"}, "42.00 RUB"},
		{"small fraction", Money{0.1, "GBP"}, "0.10 GBP"},
		{"large number", Money{1000000.99, "JPY"}, "1000000.99 JPY"},
		{"negative", Money{-50.25, "USD"}, "-50.25 USD"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.m.String()
			if got != tt.want {
				t.Errorf("Money%+v.String() = %q, want %q", tt.m, got, tt.want)
			}
		})
	}
}

func TestMoneyImplementsStringer(t *testing.T) {
	var _ fmt.Stringer = Money{}
}
