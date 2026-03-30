package optimize_me

import (
	"strings"
	"testing"
)

func generateRecords(n int) []Record {
	records := make([]Record, n)
	for i := range n {
		records[i] = Record{ID: i, Name: "user", Score: float64(i % 100)}
	}
	return records
}

func TestProcessRecords(t *testing.T) {
	records := []Record{
		{1, "Alice", 95.5},
		{2, "Bob", 82.0},
	}
	result := ProcessRecords(records)
	if !strings.Contains(result, "Alice") || !strings.Contains(result, "Bob") {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestFilterHighScores(t *testing.T) {
	records := []Record{
		{1, "Alice", 95.5},
		{2, "Bob", 50.0},
		{3, "Charlie", 75.0},
	}
	got := FilterHighScores(records, 70.0)
	if len(got) != 2 {
		t.Errorf("FilterHighScores: got %d records, want 2", len(got))
	}
}

func BenchmarkProcessRecords(b *testing.B) {
	records := generateRecords(10000)
	b.ResetTimer()
	for b.Loop() {
		ProcessRecords(records)
	}
}

func BenchmarkFilterHighScores(b *testing.B) {
	records := generateRecords(10000)
	b.ResetTimer()
	for b.Loop() {
		FilterHighScores(records, 50.0)
	}
}
