package pool_benchmark

import (
	"bytes"
	"encoding/json"
	"sync"
)

var bufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

func ProcessWithAlloc(data []byte) []byte {
	buf := new(bytes.Buffer)
	return process(data, buf)
}

func ProcessWithPool(data []byte) []byte {
	buf := bufPool.Get().(*bytes.Buffer)
	defer bufPool.Put(buf)
	return process(data, buf)
}

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
