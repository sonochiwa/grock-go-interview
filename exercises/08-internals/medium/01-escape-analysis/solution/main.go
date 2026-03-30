package escape_analysis

import (
	"fmt"
	"strconv"
	"testing"
)

// Этот файл — solution для бенчмарков (solution/main_test.go)
// Сам main.go не меняется — задание в написании бенчмарков

// Пара 1: pointer return escapes, value return stays on stack
func newIntHeap() *int { x := 42; return &x } // escapes: return pointer
func newIntStack() int { x := 42; return x }  // stack: value copy

// Пара 2: interface{} causes boxing → heap allocation
func sumInterface(a, b any) any { return a.(int) + b.(int) } // escapes: interface boxing
func sumDirect(a, b int) int    { return a + b }             // stack: no boxing

// Пара 3: closure escaping captures by reference → heap
func closureHeap() func() int { x := 0; return func() int { x++; return x } }    // escapes: closure outlives function
func closureStack() int       { x := 0; add := func() { x++ }; add(); return x } // stack: closure doesn't escape

// Пара 4: slice returned escapes, local slice may stay on stack
func sliceGrow() []int {
	var s []int
	for i := range 100 {
		s = append(s, i)
	}
	return s // escapes: returned
}
func slicePrealloc() int {
	s := make([]int, 0, 100) // may stay on stack (known size, not returned)
	for i := range 100 {
		s = append(s, i)
	}
	sum := 0
	for _, v := range s {
		sum += v
	}
	return sum
}

// Пара 5: fmt.Sprintf uses interface{} → allocations; strconv.Itoa is direct
func formatFmt(n int) string     { return fmt.Sprintf("%d", n) } // 2 allocs (boxing + string)
func formatStrconv(n int) string { return strconv.Itoa(n) }      // 1 alloc (string only)

// Solution benchmarks:

func BenchmarkNewIntHeap(b *testing.B) {
	for b.Loop() {
		_ = newIntHeap()
	}
}
func BenchmarkNewIntStack(b *testing.B) {
	for b.Loop() {
		_ = newIntStack()
	}
}

func BenchmarkSumInterface(b *testing.B) {
	for b.Loop() {
		_ = sumInterface(1, 2)
	}
}
func BenchmarkSumDirect(b *testing.B) {
	for b.Loop() {
		_ = sumDirect(1, 2)
	}
}

func BenchmarkClosureHeap(b *testing.B) {
	for b.Loop() {
		_ = closureHeap()
	}
}
func BenchmarkClosureStack(b *testing.B) {
	for b.Loop() {
		_ = closureStack()
	}
}

func BenchmarkSliceGrow(b *testing.B) {
	for b.Loop() {
		_ = sliceGrow()
	}
}
func BenchmarkSlicePrealloc(b *testing.B) {
	for b.Loop() {
		_ = slicePrealloc()
	}
}

func BenchmarkFormatFmt(b *testing.B) {
	for b.Loop() {
		_ = formatFmt(12345)
	}
}
func BenchmarkFormatStrconv(b *testing.B) {
	for b.Loop() {
		_ = formatStrconv(12345)
	}
}
