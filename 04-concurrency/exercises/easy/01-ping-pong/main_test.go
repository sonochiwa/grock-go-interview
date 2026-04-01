package ping_pong

import (
	"slices"
	"testing"
)

func TestPingPong(t *testing.T) {
	tests := []struct {
		name     string
		n        int
		expected []string
	}{
		{
			name:     "n=1 returns only ping",
			n:        1,
			expected: []string{"ping"},
		},
		{
			name:     "n=2 returns ping pong",
			n:        2,
			expected: []string{"ping", "pong"},
		},
		{
			name:     "n=5 returns alternating messages",
			n:        5,
			expected: []string{"ping", "pong", "ping", "pong", "ping"},
		},
		{
			name:     "n=10 returns 10 messages",
			n:        10,
			expected: []string{"ping", "pong", "ping", "pong", "ping", "pong", "ping", "pong", "ping", "pong"},
		},
		{
			name:     "n=0 returns empty slice",
			n:        0,
			expected: []string{},
		},
		{
			name:     "negative n returns empty slice",
			n:        -1,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PingPong(tt.n)
			if !slices.Equal(result, tt.expected) {
				t.Errorf("PingPong(%d) = %v, want %v", tt.n, result, tt.expected)
			}
		})
	}
}
