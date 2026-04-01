package worker_pool

import "sync"

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
	wg      sync.WaitGroup
}

func NewPool[T, R any](workers int, handler func(T) (R, error)) *Pool[T, R] {
	p := &Pool[T, R]{
		handler: handler,
		tasks:   make(chan task[T, R], workers*2),
	}
	p.wg.Add(workers)
	for range workers {
		go p.worker()
	}
	return p
}

func (p *Pool[T, R]) worker() {
	defer p.wg.Done()
	for t := range p.tasks {
		v, err := p.handler(t.input)
		t.result <- Result[R]{Value: v, Err: err}
	}
}

func (p *Pool[T, R]) Submit(input T) <-chan Result[R] {
	ch := make(chan Result[R], 1)
	p.tasks <- task[T, R]{input: input, result: ch}
	return ch
}

func (p *Pool[T, R]) Close() {
	close(p.tasks)
	p.wg.Wait()
}
