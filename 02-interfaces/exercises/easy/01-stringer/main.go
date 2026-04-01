package stringer

import "fmt"

type Money struct {
	Amount   float64
	Currency string
}

// TODO: реализуй fmt.Stringer для Money
// Формат: "100.50 USD"
func (m Money) String() string {
	_ = fmt.Sprintf // hint: используй fmt.Sprintf
	return ""
}
