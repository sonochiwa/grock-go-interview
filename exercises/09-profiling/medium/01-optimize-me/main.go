package optimize_me

import "fmt"

type Record struct {
	ID    int
	Name  string
	Score float64
}

// TODO: оптимизируй эту функцию (минимум 3x ускорение)
func ProcessRecords(records []Record) string {
	// Проблема 1: string конкатенация в цикле
	result := ""
	for _, r := range records {
		// Проблема 2: fmt.Sprintf для простых конверсий
		line := fmt.Sprintf("ID:%d Name:%s Score:%.1f\n", r.ID, r.Name, r.Score)
		result += line // O(n²) copying!
	}
	return result
}

// TODO: оптимизируй — верни только записи со Score > threshold
func FilterHighScores(records []Record, threshold float64) []Record {
	// Проблема 3: нет предаллокации
	var result []Record
	for _, r := range records {
		if r.Score > threshold {
			// Проблема 4: копирование структуры (в данном случае мелкая, но покажи что знаешь)
			result = append(result, r)
		}
	}
	return result
}
