package escape_analysis

import (
	"fmt"
	"strconv"
)

// --- Пара 1: возврат указателя ---

func newIntHeap() *int {
	x := 42
	return &x
}

func newIntStack() int {
	x := 42
	return x
}

// --- Пара 2: interface boxing ---

func sumInterface(a, b any) any {
	return a.(int) + b.(int)
}

func sumDirect(a, b int) int {
	return a + b
}

// --- Пара 3: closure capture ---

func closureHeap() func() int {
	x := 0
	return func() int {
		x++
		return x
	}
}

func closureStack() int {
	x := 0
	add := func() { x++ }
	add()
	return x
}

// --- Пара 4: slice ---

func sliceGrow() []int {
	var s []int
	for i := range 100 {
		s = append(s, i)
	}
	return s
}

func slicePrealloc() int {
	s := make([]int, 0, 100)
	for i := range 100 {
		s = append(s, i)
	}
	sum := 0
	for _, v := range s {
		sum += v
	}
	return sum
}

// --- Пара 5: fmt.Sprintf vs strconv ---

func formatFmt(n int) string {
	return fmt.Sprintf("%d", n)
}

func formatStrconv(n int) string {
	return strconv.Itoa(n)
}
