# Strategy

## В Go

Strategy позволяет подменять алгоритм через интерфейс. В Go — это просто интерфейс + разные реализации.

```go
type Compressor interface {
    Compress(data []byte) ([]byte, error)
}

type GzipCompressor struct{}
func (g *GzipCompressor) Compress(data []byte) ([]byte, error) { ... }

type ZstdCompressor struct{}
func (z *ZstdCompressor) Compress(data []byte) ([]byte, error) { ... }

type NoopCompressor struct{}
func (n *NoopCompressor) Compress(data []byte) ([]byte, error) { return data, nil }

// Использование
type FileProcessor struct {
    compressor Compressor
}

func (fp *FileProcessor) Process(data []byte) ([]byte, error) {
    return fp.compressor.Compress(data)
}

// Подмена стратегии
p := &FileProcessor{compressor: &GzipCompressor{}}
p := &FileProcessor{compressor: &ZstdCompressor{}}
```

### Strategy через функциональный тип

```go
type CompressFn func([]byte) ([]byte, error)

type FileProcessor struct {
    compress CompressFn
}

fp := &FileProcessor{
    compress: gzipCompress, // просто функция
}
```

В Go strategy через функциональный тип часто проще, чем через интерфейс (для стратегий с одним методом).
