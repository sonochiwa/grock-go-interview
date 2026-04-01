package reverse_slice

import (
	"slices"
	"testing"
)

func TestReverseSliceInt(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		expected []int
	}{
		{
			name:     "multiple elements",
			input:    []int{1, 2, 3, 4, 5},
			expected: []int{5, 4, 3, 2, 1},
		},
		{
			name:     "two elements",
			input:    []int{10, 20},
			expected: []int{20, 10},
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
			result := ReverseSlice(tt.input)
			if !slices.Equal(result, tt.expected) {
				t.Errorf("ReverseSlice(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestReverseSliceString(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "multiple strings",
			input:    []string{"a", "b", "c", "d"},
			expected: []string{"d", "c", "b", "a"},
		},
		{
			name:     "single string",
			input:    []string{"hello"},
			expected: []string{"hello"},
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ReverseSlice(tt.input)
			if !slices.Equal(result, tt.expected) {
				t.Errorf("ReverseSlice(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestReverseSliceDoesNotMutateOriginal(t *testing.T) {
	original := []int{1, 2, 3, 4, 5}
	copyOriginal := make([]int, len(original))
	copy(copyOriginal, original)

	_ = ReverseSlice(original)

	if !slices.Equal(original, copyOriginal) {
		t.Errorf("original slice was mutated: got %v, want %v", original, copyOriginal)
	}
}
