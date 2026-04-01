package stringer

import "fmt"

type Money struct {
	Amount   float64
	Currency string
}

func (m Money) String() string {
	return fmt.Sprintf("%.2f %s", m.Amount, m.Currency)
}
