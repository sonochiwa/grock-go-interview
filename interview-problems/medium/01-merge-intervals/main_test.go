package merge_intervals

import (
	"reflect"
	"testing"
)

func TestMergeIntervals(t *testing.T) {
	tests := []struct {
		name string
		in   []Interval
		want []Interval
	}{
		{
			"overlapping",
			[]Interval{{1, 3}, {2, 6}, {8, 10}, {15, 18}},
			[]Interval{{1, 6}, {8, 10}, {15, 18}},
		},
		{
			"adjacent",
			[]Interval{{1, 4}, {4, 5}},
			[]Interval{{1, 5}},
		},
		{
			"no overlap",
			[]Interval{{1, 2}, {5, 6}},
			[]Interval{{1, 2}, {5, 6}},
		},
		{
			"all overlap",
			[]Interval{{1, 10}, {2, 5}, {3, 7}},
			[]Interval{{1, 10}},
		},
		{
			"single",
			[]Interval{{1, 5}},
			[]Interval{{1, 5}},
		},
		{
			"empty",
			nil,
			nil,
		},
		{
			"unsorted",
			[]Interval{{8, 10}, {1, 3}, {2, 6}, {15, 18}},
			[]Interval{{1, 6}, {8, 10}, {15, 18}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeIntervals(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeIntervals(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
