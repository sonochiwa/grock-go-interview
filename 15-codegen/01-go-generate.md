# go generate

## Обзор

`go generate` запускает произвольные команды, указанные в комментариях `//go:generate`. Используется для генерации кода перед компиляцией.

```go
// В исходном файле:
//go:generate stringer -type=Color
//go:generate mockgen -source=interface.go -destination=mock_interface.go

type Color int
const (
    Red Color = iota
    Green
    Blue
)
```

```bash
# Запуск
go generate ./...          # все пакеты
go generate ./pkg/models/  # конкретный пакет
```

### Популярные генераторы

| Инструмент | Что делает |
|---|---|
| stringer | String() для iota констант |
| mockgen | Моки для интерфейсов |
| easyjson | Быстрый JSON маршалинг (без reflect) |
| protoc-gen-go | gRPC/protobuf код |
| sqlc | Type-safe SQL запросы |
| enumer | Расширенный stringer + validation |
| wire | Dependency injection |

### Best practices

1. Коммить сгенерированный код в репозиторий
2. Добавляй `// Code generated ... DO NOT EDIT.` в начало
3. `go generate` — НЕ часть `go build` (запускай явно)
4. В CI: проверяй что `go generate` не изменяет файлы
