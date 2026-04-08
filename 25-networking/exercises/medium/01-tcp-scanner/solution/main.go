package tcp_scanner

import (
	"fmt"
	"net"
	"slices"
	"sync"
	"time"
)

func ScanPort(host string, port int, timeout time.Duration) bool {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func ScanRange(host string, startPort, endPort int, workers int, timeout time.Duration) []int {
	ports := make(chan int, workers)
	var mu sync.Mutex
	var open []int
	var wg sync.WaitGroup

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for port := range ports {
				if ScanPort(host, port, timeout) {
					mu.Lock()
					open = append(open, port)
					mu.Unlock()
				}
			}
		}()
	}

	for p := startPort; p <= endPort; p++ {
		ports <- p
	}
	close(ports)
	wg.Wait()

	slices.Sort(open)
	return open
}
