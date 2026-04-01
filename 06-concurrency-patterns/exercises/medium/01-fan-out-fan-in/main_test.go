package fan_out_fan_in

import (
	"sort"
	"testing"
)

func gen(nums ...int) <-chan int {
	ch := make(chan int)
	go func() {
		defer close(ch)
		for _, n := range nums {
			ch <- n
		}
	}()
	return ch
}

func collect(ch <-chan int) []int {
	var r []int
	for v := range ch {
		r = append(r, v)
	}
	return r
}

func TestFanOut(t *testing.T) {
	in := gen(1, 2, 3, 4, 5, 6)
	outs := FanOut(in, 3)
	if len(outs) != 3 {
		t.Fatalf("FanOut returned %d channels, want 3", len(outs))
	}

	var all []int
	for _, ch := range outs {
		all = append(all, collect(ch)...)
	}
	sort.Ints(all)
	want := []int{1, 2, 3, 4, 5, 6}
	if len(all) != len(want) {
		t.Fatalf("got %v, want %v", all, want)
	}
	for i := range want {
		if all[i] != want[i] {
			t.Errorf("[%d] = %d, want %d", i, all[i], want[i])
		}
	}
}

func TestFanIn(t *testing.T) {
	ch1 := gen(1, 3, 5)
	ch2 := gen(2, 4, 6)
	merged := FanIn(ch1, ch2)
	all := collect(merged)
	sort.Ints(all)
	want := []int{1, 2, 3, 4, 5, 6}
	if len(all) != len(want) {
		t.Fatalf("got %v, want %v", all, want)
	}
	for i := range want {
		if all[i] != want[i] {
			t.Errorf("[%d] = %d, want %d", i, all[i], want[i])
		}
	}
}

func TestRoundTrip(t *testing.T) {
	in := gen(10, 20, 30, 40)
	outs := FanOut(in, 2)
	merged := FanIn(outs...)
	all := collect(merged)
	sort.Ints(all)
	want := []int{10, 20, 30, 40}
	if len(all) != len(want) {
		t.Fatalf("got %v, want %v", all, want)
	}
}
