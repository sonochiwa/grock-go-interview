package fan_out_fan_in

// TODO: раздай значения из in в n каналов round-robin. Закрой все каналы.
func FanOut(in <-chan int, n int) []<-chan int {
	return nil
}

// TODO: объедини все каналы в один. Закрой выходной канал когда все входные закрыты.
func FanIn(channels ...<-chan int) <-chan int {
	return nil
}
