package process_monitor

import (
	"os"
	"runtime"
	"testing"
)

func TestGetProcessInfo(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux only")
	}
	info, err := GetProcessInfo(os.Getpid())
	if err != nil {
		t.Fatalf("GetProcessInfo error: %v", err)
	}
	if info.PID != os.Getpid() {
		t.Errorf("PID = %d, want %d", info.PID, os.Getpid())
	}
	if info.Name == "" {
		t.Error("Name is empty")
	}
	if info.MemoryKB <= 0 {
		t.Error("MemoryKB should be positive")
	}
	if info.State == "" {
		t.Error("State is empty")
	}
}

func TestListProcesses(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux only")
	}
	procs, err := ListProcesses()
	if err != nil {
		t.Fatalf("ListProcesses error: %v", err)
	}
	if len(procs) < 2 {
		t.Errorf("expected at least 2 processes, got %d", len(procs))
	}
}

func TestTopByMemory(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux only")
	}
	top, err := TopByMemory(5)
	if err != nil {
		t.Fatalf("TopByMemory error: %v", err)
	}
	if len(top) > 5 {
		t.Errorf("expected max 5, got %d", len(top))
	}
	// Check sorted descending
	for i := 1; i < len(top); i++ {
		if top[i].MemoryKB > top[i-1].MemoryKB {
			t.Error("should be sorted descending by memory")
		}
	}
}
