package tcp_scanner

import (
	"net"
	"testing"
	"time"
)

func TestScanPortOpen(t *testing.T) {
	// Start a test server
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	if !ScanPort("127.0.0.1", port, time.Second) {
		t.Errorf("port %d should be open", port)
	}
}

func TestScanPortClosed(t *testing.T) {
	if ScanPort("127.0.0.1", 1, 100*time.Millisecond) {
		t.Error("port 1 should be closed")
	}
}

func TestScanRange(t *testing.T) {
	// Start 3 test servers
	var listeners []net.Listener
	var openPorts []int
	for range 3 {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		listeners = append(listeners, ln)
		openPorts = append(openPorts, ln.Addr().(*net.TCPAddr).Port)
	}
	defer func() {
		for _, ln := range listeners {
			ln.Close()
		}
	}()

	// Scan range covering all open ports
	minPort, maxPort := openPorts[0], openPorts[0]
	for _, p := range openPorts {
		if p < minPort {
			minPort = p
		}
		if p > maxPort {
			maxPort = p
		}
	}

	result := ScanRange("127.0.0.1", minPort, maxPort, 10, time.Second)
	if len(result) < 3 {
		t.Errorf("expected at least 3 open ports, got %d", len(result))
	}

	// Check sorted
	for i := 1; i < len(result); i++ {
		if result[i] < result[i-1] {
			t.Error("result should be sorted")
		}
	}
}
