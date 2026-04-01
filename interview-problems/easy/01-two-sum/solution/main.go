package two_sum

func TwoSum(nums []int, target int) (int, int) {
	seen := make(map[int]int, len(nums))
	for i, n := range nums {
		complement := target - n
		if j, ok := seen[complement]; ok {
			if j < i {
				return j, i
			}
			return i, j
		}
		seen[n] = i
	}
	return -1, -1
}
