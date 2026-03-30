# Fan-Out / Fan-In

- `FanOut(in <-chan int, n int) []<-chan int` — раздаёт значения из одного канала в n каналов (round-robin)
- `FanIn(channels ...<-chan int) <-chan int` — объединяет все каналы в один

Все каналы должны закрываться корректно.
