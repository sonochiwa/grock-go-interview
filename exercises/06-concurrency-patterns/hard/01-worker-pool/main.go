package worker_pool

type Result[R any] struct {
	Value R
	Err   error
}

type task[T, R any] struct {
	input  T
	result chan Result[R]
}

type Pool[T, R any] struct {
	handler func(T) (R, error)
	tasks   chan task[T, R]
}

// TODO: создай pool с N воркерами
func NewPool[T, R any](workers int, handler func(T) (R, error)) *Pool[T, R] {
	return nil
}

// TODO: отправь задачу, верни канал для получения результата
func (p *Pool[T, R]) Submit(input T) <-chan Result[R] {
	return nil
}

// TODO: закрой канал задач, дождись завершения воркеров
func (p *Pool[T, R]) Close() {
}
