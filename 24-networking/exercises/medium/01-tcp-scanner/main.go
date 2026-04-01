package tcp_scanner

import (
	"net"
	"time"
)

// TODO: проверить один порт (TCP connect)
func ScanPort(host string, port int, timeout time.Duration) bool {
	_ = net.DialTimeout
	return false
}

// TODO: сканировать диапазон портов с worker pool
// workers горутин параллельно сканируют порты
// вернуть отсортированный []int открытых портов
func ScanRange(host string, startPort, endPort int, workers int, timeout time.Duration) []int {
	return nil
}
