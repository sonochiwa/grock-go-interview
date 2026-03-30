package fan_out_fan_in

import "sync"

func FanOut(in <-chan int, n int) []<-chan int {
	outs := make([]chan int, n)
	for i := range n {
		outs[i] = make(chan int)
	}
	go func() {
		i := 0
		for v := range in {
			outs[i%n] <- v
			i++
		}
		for _, ch := range outs {
			close(ch)
		}
	}()

	result := make([]<-chan int, n)
	for i, ch := range outs {
		result[i] = ch
	}
	return result
}

func FanIn(channels ...<-chan int) <-chan int {
	out := make(chan int)
	var wg sync.WaitGroup
	for _, ch := range channels {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for v := range ch {
				out <- v
			}
		}()
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}
