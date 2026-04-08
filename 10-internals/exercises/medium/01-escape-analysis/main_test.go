package escape_analysis

import "testing"

// TODO: напиши бенчмарки для каждой пары
// Сравни heap vs stack версию
// Запусти: go test -bench=. -benchmem

func BenchmarkNewIntHeap(b *testing.B) {
	for b.Loop() {
		_ = newIntHeap()
	}
}

// TODO: BenchmarkNewIntStack

// TODO: BenchmarkSumInterface vs BenchmarkSumDirect

// TODO: BenchmarkClosureHeap vs BenchmarkClosureStack

// TODO: BenchmarkSliceGrow vs BenchmarkSlicePrealloc

// TODO: BenchmarkFormatFmt vs BenchmarkFormatStrconv
