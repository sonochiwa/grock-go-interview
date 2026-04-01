package custom_sort

import (
	"slices"
	"testing"
)

type Person struct {
	Name  string
	Age   int
	Score float64
}

func TestSortBy(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		less     func(a, b int) bool
		expected []int
	}{
		{
			name:     "ascending",
			input:    []int{3, 1, 4, 1, 5, 9, 2, 6},
			less:     func(a, b int) bool { return a < b },
			expected: []int{1, 1, 2, 3, 4, 5, 6, 9},
		},
		{
			name:     "descending",
			input:    []int{3, 1, 4, 1, 5},
			less:     func(a, b int) bool { return a > b },
			expected: []int{5, 4, 3, 1, 1},
		},
		{
			name:     "single element",
			input:    []int{42},
			less:     func(a, b int) bool { return a < b },
			expected: []int{42},
		},
		{
			name:     "empty slice",
			input:    []int{},
			less:     func(a, b int) bool { return a < b },
			expected: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SortBy(tt.input, tt.less)
			if !slices.Equal(result, tt.expected) {
				t.Errorf("SortBy(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSortByDoesNotMutateOriginal(t *testing.T) {
	original := []int{3, 1, 4, 1, 5}
	copyOriginal := make([]int, len(original))
	copy(copyOriginal, original)

	_ = SortBy(original, func(a, b int) bool { return a < b })

	if !slices.Equal(original, copyOriginal) {
		t.Errorf("original slice was mutated: got %v, want %v", original, copyOriginal)
	}
}

func TestSortByStructs(t *testing.T) {
	people := []Person{
		{Name: "Charlie", Age: 35, Score: 88.5},
		{Name: "Alice", Age: 25, Score: 95.0},
		{Name: "Bob", Age: 30, Score: 82.3},
	}

	result := SortBy(people, func(a, b Person) bool { return a.Age < b.Age })
	expected := []Person{
		{Name: "Alice", Age: 25, Score: 95.0},
		{Name: "Bob", Age: 30, Score: 82.3},
		{Name: "Charlie", Age: 35, Score: 88.5},
	}

	for i := range result {
		if result[i] != expected[i] {
			t.Errorf("index %d: got %v, want %v", i, result[i], expected[i])
		}
	}
}

func TestSortByField(t *testing.T) {
	people := []Person{
		{Name: "Charlie", Age: 35, Score: 88.5},
		{Name: "Alice", Age: 25, Score: 95.0},
		{Name: "Bob", Age: 30, Score: 82.3},
	}

	tests := []struct {
		name      string
		field     string
		ascending bool
		expected  []Person
		wantErr   bool
	}{
		{
			name:      "sort by Age ascending",
			field:     "Age",
			ascending: true,
			expected: []Person{
				{Name: "Alice", Age: 25, Score: 95.0},
				{Name: "Bob", Age: 30, Score: 82.3},
				{Name: "Charlie", Age: 35, Score: 88.5},
			},
		},
		{
			name:      "sort by Age descending",
			field:     "Age",
			ascending: false,
			expected: []Person{
				{Name: "Charlie", Age: 35, Score: 88.5},
				{Name: "Bob", Age: 30, Score: 82.3},
				{Name: "Alice", Age: 25, Score: 95.0},
			},
		},
		{
			name:      "sort by Name ascending",
			field:     "Name",
			ascending: true,
			expected: []Person{
				{Name: "Alice", Age: 25, Score: 95.0},
				{Name: "Bob", Age: 30, Score: 82.3},
				{Name: "Charlie", Age: 35, Score: 88.5},
			},
		},
		{
			name:      "sort by Score ascending",
			field:     "Score",
			ascending: true,
			expected: []Person{
				{Name: "Bob", Age: 30, Score: 82.3},
				{Name: "Charlie", Age: 35, Score: 88.5},
				{Name: "Alice", Age: 25, Score: 95.0},
			},
		},
		{
			name:    "non-existent field",
			field:   "NonExistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SortByField(people, tt.field, tt.ascending)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("index %d: got %v, want %v", i, result[i], tt.expected[i])
				}
			}
		})
	}
}
