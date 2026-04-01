# Pipeline

Построй pipeline из 3 стадий:

1. `Generate(nums ...int) <-chan int` — отправляет числа в канал, закрывает
2. `Square(in <-chan int) <-chan int` — возводит в квадрат
3. `Filter(in <-chan int, pred func(int) bool) <-chan int` — оставляет только элементы, прошедшие предикат

Каждая стадия закрывает выходной канал после завершения.
