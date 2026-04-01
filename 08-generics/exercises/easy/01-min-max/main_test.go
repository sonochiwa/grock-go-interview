package min_max

import (
	"errors"
	"testing"
)

func TestMin(t *testing.T) {
	tests := []struct {
		name string
		a, b int
		want int
	}{
		{"a < b", 1, 2, 1},
		{"a > b", 5, 3, 3},
		{"a == b", 4, 4, 4},
		{"negative", -10, -5, -10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Min(tt.a, tt.b); got != tt.want {
				t.Errorf("Min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestMinFloat64(t *testing.T) {
	tests := []struct {
		name string
		a, b float64
		want float64
	}{
		{"a < b", 1.1, 2.2, 1.1},
		{"a > b", 5.5, 3.3, 3.3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Min(tt.a, tt.b); got != tt.want {
				t.Errorf("Min(%f, %f) = %f, want %f", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestMinString(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want string
	}{
		{"alphabetical", "apple", "banana", "apple"},
		{"reverse", "zebra", "alpha", "alpha"},
		{"equal", "go", "go", "go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Min(tt.a, tt.b); got != tt.want {
				t.Errorf("Min(%q, %q) = %q, want %q", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestMax(t *testing.T) {
	tests := []struct {
		name string
		a, b int
		want int
	}{
		{"a < b", 1, 2, 2},
		{"a > b", 5, 3, 5},
		{"a == b", 4, 4, 4},
		{"negative", -10, -5, -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Max(tt.a, tt.b); got != tt.want {
				t.Errorf("Max(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestMaxFloat64(t *testing.T) {
	if got := Max(1.5, 2.5); got != 2.5 {
		t.Errorf("Max(1.5, 2.5) = %f, want 2.5", got)
	}
}

func TestMaxString(t *testing.T) {
	if got := Max("apple", "banana"); got != "banana" {
		t.Errorf("Max(apple, banana) = %q, want banana", got)
	}
}

func TestMinSlice(t *testing.T) {
	tests := []struct {
		name    string
		input   []int
		want    int
		wantErr error
	}{
		{"single element", []int{42}, 42, nil},
		{"multiple elements", []int{3, 1, 4, 1, 5, 9}, 1, nil},
		{"negative elements", []int{-3, -1, -4}, -4, nil},
		{"sorted", []int{1, 2, 3, 4, 5}, 1, nil},
		{"reverse sorted", []int{5, 4, 3, 2, 1}, 1, nil},
		{"empty slice", []int{}, 0, ErrEmptySlice},
		{"nil slice", nil, 0, ErrEmptySlice},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MinSlice(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("MinSlice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("MinSlice() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMinSliceFloat64(t *testing.T) {
	got, err := MinSlice([]float64{3.14, 2.71, 1.41})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 1.41 {
		t.Errorf("MinSlice() = %f, want 1.41", got)
	}
}

func TestMinSliceString(t *testing.T) {
	got, err := MinSlice([]string{"cherry", "apple", "banana"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "apple" {
		t.Errorf("MinSlice() = %q, want apple", got)
	}
}
