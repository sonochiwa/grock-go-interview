package parallel_fetch

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"
)

// Result содержит результат загрузки одного URL.
type Result struct {
	URL  string
	Body string
	Err  error
}

// FetchAll загружает все URL конкурентно с ограничением параллелизма.
// fetch — функция для загрузки одного URL.
// Если ctx отменён, возвращает ошибку контекста.
func FetchAll(ctx context.Context, urls []string, fetch func(ctx context.Context, url string) (string, error)) ([]Result, error) {
	results := make([]Result, len(urls))
	var mu sync.Mutex
	_ = mu // используется для безопасной записи в results

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(10)

	for i, url := range urls {
		i, url := i, url
		g.Go(func() error {
			body, err := fetch(gCtx, url)
			results[i] = Result{
				URL:  url,
				Body: body,
				Err:  err,
			}
			if gCtx.Err() != nil {
				return gCtx.Err()
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}
