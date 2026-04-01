package palindrome

import (
	"strings"
	"unicode"
)

func IsPalindrome(s string) bool {
	s = strings.ToLower(s)
	runes := make([]rune, 0, len(s))
	for _, r := range s {
		if !unicode.IsSpace(r) {
			runes = append(runes, r)
		}
	}
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		if runes[i] != runes[j] {
			return false
		}
	}
	return true
}
