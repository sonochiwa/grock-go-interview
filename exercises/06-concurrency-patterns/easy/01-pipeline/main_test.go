package pipeline

import "testing"

func collect(ch <-chan int) []int {
	var res []int
	for v := range ch {
		res = append(res, v)
	}
	return res
}

func TestGenerate(t *testing.T) {
	got := collect(Generate(1, 2, 3))
	want := []int{1, 2, 3}
	if len(got) != len(want) {
		t.Fatalf("Generate: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Generate[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestSquare(t *testing.T) {
	got := collect(Square(Generate(1, 2, 3, 4)))
	want := []int{1, 4, 9, 16}
	if len(got) != len(want) {
		t.Fatalf("Square: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Square[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestPipeline(t *testing.T) {
	// Generate → Square → Filter(> 10)
	out := Filter(
		Square(Generate(1, 2, 3, 4, 5)),
		func(n int) bool { return n > 10 },
	)
	got := collect(out)
	want := []int{16, 25}
	if len(got) != len(want) {
		t.Fatalf("Pipeline: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Pipeline[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestEmpty(t *testing.T) {
	got := collect(Square(Generate()))
	if len(got) != 0 {
		t.Errorf("empty pipeline got %v", got)
	}
}
