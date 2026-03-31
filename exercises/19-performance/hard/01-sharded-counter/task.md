# Sharded Counter (False Sharing)

Реализуй 3 варианта concurrent counter и сравни производительность:

1. `MutexCounter` — sync.Mutex
2. `AtomicCounter` — atomic.Int64 (single)
3. `ShardedCounter` — N шардов с padding (избежание false sharing)

`ShardedCounter`: каждая горутина пишет в свой шард (по goroutine ID или round-robin).
Padding между шардами = cache line (64 bytes).

Напиши бенчмарки с GOMAXPROCS=8 и 8 горутинами, докажи что ShardedCounter быстрее.
