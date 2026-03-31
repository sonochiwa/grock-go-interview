# sync.Pool Benchmark

Реализуй два варианта обработки JSON запросов и сравни через бенчмарки:

1. **Без pool**: каждый раз создаёт новый `bytes.Buffer`
2. **С pool**: переиспользует `bytes.Buffer` через `sync.Pool`

Напиши `ProcessWithAlloc(data []byte) []byte` и `ProcessWithPool(data []byte) []byte`.
Оба делают одно и то же: декодируют JSON, добавляют поле `"processed": true`, кодируют обратно.

Напиши бенчмарки, докажи разницу в аллокациях.
