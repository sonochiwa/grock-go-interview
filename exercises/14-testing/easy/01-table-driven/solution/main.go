package table_driven

import (
	"strconv"
	"testing"
)

func FizzBuzz(n int) string {
	switch {
	case n%15 == 0:
		return "FizzBuzz"
	case n%3 == 0:
		return "Fizz"
	case n%5 == 0:
		return "Buzz"
	default:
		return strconv.Itoa(n)
	}
}

// Solution tests (solution/main_test.go):
func TestFizzBuzzSolution(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want string
	}{
		{"regular 1", 1, "1"},
		{"regular 2", 2, "2"},
		{"fizz 3", 3, "Fizz"},
		{"regular 4", 4, "4"},
		{"buzz 5", 5, "Buzz"},
		{"fizz 6", 6, "Fizz"},
		{"fizz 9", 9, "Fizz"},
		{"buzz 10", 10, "Buzz"},
		{"fizzbuzz 15", 15, "FizzBuzz"},
		{"fizzbuzz 30", 30, "FizzBuzz"},
		{"zero", 0, "FizzBuzz"},
		{"negative fizz", -3, "Fizz"},
		{"negative buzz", -5, "Buzz"},
		{"negative fizzbuzz", -15, "FizzBuzz"},
		{"negative regular", -1, "-1"},
		{"large fizzbuzz", 300, "FizzBuzz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FizzBuzz(tt.n); got != tt.want {
				t.Errorf("FizzBuzz(%d) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}
