package optimize_me

import (
	"strconv"
	"strings"
)

type Record struct {
	ID    int
	Name  string
	Score float64
}

func ProcessRecords(records []Record) string {
	var b strings.Builder
	b.Grow(len(records) * 40) // estimate

	for _, r := range records {
		b.WriteString("ID:")
		b.WriteString(strconv.Itoa(r.ID))
		b.WriteString(" Name:")
		b.WriteString(r.Name)
		b.WriteString(" Score:")
		b.WriteString(strconv.FormatFloat(r.Score, 'f', 1, 64))
		b.WriteByte('\n')
	}
	return b.String()
}

func FilterHighScores(records []Record, threshold float64) []Record {
	result := make([]Record, 0, len(records)/2) // estimate
	for _, r := range records {
		if r.Score > threshold {
			result = append(result, r)
		}
	}
	return result
}
