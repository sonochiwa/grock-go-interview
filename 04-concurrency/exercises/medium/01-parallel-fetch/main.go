package parallel_fetch

import "context"

// Result содержит результат загрузки одного URL.
type Result struct {
	URL  string
	Body string
	Err  error
}

// FetchAll загружает все URL конкурентно с ограничением параллелизма.
// fetch — функция для загрузки одного URL.
// Если ctx отменён, возвращает ошибку контекста.
// TODO: реализуй функцию
func FetchAll(ctx context.Context, urls []string, fetch func(ctx context.Context, url string) (string, error)) ([]Result, error) {
	return nil, nil
}
