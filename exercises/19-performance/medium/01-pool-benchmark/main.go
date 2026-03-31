package pool_benchmark

import (
	"bytes"
	"encoding/json"
	"sync"
)

var bufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

// TODO: без pool — создаёт bytes.Buffer каждый раз
func ProcessWithAlloc(data []byte) []byte {
	return nil
}

// TODO: с pool — берёт bytes.Buffer из sync.Pool, возвращает после использования
func ProcessWithPool(data []byte) []byte {
	_ = bufPool
	return nil
}

// Helper: общая логика обработки
func process(data []byte, buf *bytes.Buffer) []byte {
	buf.Reset()
	var m map[string]any
	json.Unmarshal(data, &m)
	m["processed"] = true
	json.NewEncoder(buf).Encode(m)
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result
}
