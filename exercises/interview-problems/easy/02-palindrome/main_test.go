package palindrome

import "testing"

func TestIsPalindrome(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"racecar", true},
		{"hello", false},
		{"", true},
		{"a", true},
		{"A man a plan a canal Panama", true},
		{"Аргентина манит негра", true},
		{"not a palindrome", false},
		{"Was It A Rat I Saw", true},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := IsPalindrome(tt.s); got != tt.want {
				t.Errorf("IsPalindrome(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}
