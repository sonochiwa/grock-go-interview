package two_sum

import "testing"

func TestTwoSum(t *testing.T) {
	tests := []struct {
		name   string
		nums   []int
		target int
		wantA  int
		wantB  int
	}{
		{"basic", []int{2, 7, 11, 15}, 9, 0, 1},
		{"middle", []int{3, 2, 4}, 6, 1, 2},
		{"same values", []int{3, 3}, 6, 0, 1},
		{"negative", []int{-1, -2, -3, -4, -5}, -8, 2, 4},
		{"large", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 19, 8, 9},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, b := TwoSum(tt.nums, tt.target)
			if a > b {
				a, b = b, a
			}
			if a != tt.wantA || b != tt.wantB {
				t.Errorf("TwoSum(%v, %d) = (%d,%d), want (%d,%d)",
					tt.nums, tt.target, a, b, tt.wantA, tt.wantB)
			}
		})
	}
}
