package pipeline

// TODO: отправь все nums в канал и закрой его
func Generate(nums ...int) <-chan int {
	out := make(chan int)
	// TODO
	return out
}

// TODO: читай из in, возводи в квадрат, отправь в out
func Square(in <-chan int) <-chan int {
	out := make(chan int)
	// TODO
	return out
}

// TODO: читай из in, отправь в out только если pred(n) == true
func Filter(in <-chan int, pred func(int) bool) <-chan int {
	out := make(chan int)
	// TODO
	return out
}
