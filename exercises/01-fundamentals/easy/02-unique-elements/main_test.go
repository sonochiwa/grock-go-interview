package unique_elements

import (
	"slices"
	"testing"
)

func TestUniqueInt(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		expected []int
	}{
		{
			name:     "with duplicates",
			input:    []int{1, 2, 2, 3, 1, 4},
			expected: []int{1, 2, 3, 4},
		},
		{
			name:     "all same",
			input:    []int{5, 5, 5, 5},
			expected: []int{5},
		},
		{
			name:     "already unique",
			input:    []int{1, 2, 3},
			expected: []int{1, 2, 3},
		},
		{
			name:     "single element",
			input:    []int{42},
			expected: []int{42},
		},
		{
			name:     "empty slice",
			input:    []int{},
			expected: []int{},
		},
		{
			name:     "nil slice",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Unique(tt.input)
			if !slices.Equal(result, tt.expected) {
				t.Errorf("Unique(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestUniqueString(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "with duplicates",
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "empty strings",
			input:    []string{"", "a", "", "b"},
			expected: []string{"", "a", "b"},
		},
		{
			name:     "no duplicates",
			input:    []string{"x", "y", "z"},
			expected: []string{"x", "y", "z"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Unique(tt.input)
			if !slices.Equal(result, tt.expected) {
				t.Errorf("Unique(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestUniquePreservesOrder(t *testing.T) {
	input := []int{3, 1, 4, 1, 5, 9, 2, 6, 5, 3, 5}
	expected := []int{3, 1, 4, 5, 9, 2, 6}
	result := Unique(input)
	if !slices.Equal(result, expected) {
		t.Errorf("Unique(%v) = %v, want %v (order must match first appearance)", input, result, expected)
	}
}
