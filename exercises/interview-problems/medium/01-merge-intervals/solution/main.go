package merge_intervals

import "sort"

type Interval struct {
	Start int
	End   int
}

func MergeIntervals(intervals []Interval) []Interval {
	if len(intervals) <= 1 {
		return intervals
	}

	sort.Slice(intervals, func(i, j int) bool {
		return intervals[i].Start < intervals[j].Start
	})

	result := []Interval{intervals[0]}
	for _, curr := range intervals[1:] {
		last := &result[len(result)-1]
		if curr.Start <= last.End {
			if curr.End > last.End {
				last.End = curr.End
			}
		} else {
			result = append(result, curr)
		}
	}
	return result
}
