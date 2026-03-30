package table_driven

import "testing"

func TestFizzBuzz(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want string
	}{
		{"regular 1", 1, "1"},
		// TODO: добавь минимум 8 тест-кейсов
		// - кратные 3
		// - кратные 5
		// - кратные 15
		// - граничные случаи (0, отрицательные)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FizzBuzz(tt.n); got != tt.want {
				t.Errorf("FizzBuzz(%d) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}
