# TCP Port Scanner

Реализуй конкурентный TCP port scanner:

- `ScanPort(host string, port int, timeout time.Duration) bool` — проверить один порт
- `ScanRange(host string, startPort, endPort int, workers int, timeout time.Duration) []int` — сканировать диапазон с worker pool

Возвращает отсортированный список открытых портов.
Используй `net.DialTimeout`.
