package process_monitor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

type ProcessInfo struct {
	PID      int
	Name     string
	MemoryKB int64
	State    string
}

func GetProcessInfo(pid int) (*ProcessInfo, error) {
	path := fmt.Sprintf("/proc/%d/status", pid)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info := &ProcessInfo{PID: pid}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "Name":
			info.Name = val
		case "State":
			if len(val) > 0 {
				info.State = string(val[0])
			}
		case "VmRSS":
			val = strings.TrimSuffix(val, " kB")
			info.MemoryKB, _ = strconv.ParseInt(strings.TrimSpace(val), 10, 64)
		}
	}
	return info, nil
}

func ListProcesses() ([]ProcessInfo, error) {
	entries, err := filepath.Glob("/proc/[0-9]*")
	if err != nil {
		return nil, err
	}

	var procs []ProcessInfo
	for _, entry := range entries {
		pidStr := filepath.Base(entry)
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}
		info, err := GetProcessInfo(pid)
		if err != nil {
			continue
		}
		procs = append(procs, *info)
	}
	return procs, nil
}

func TopByMemory(n int) ([]ProcessInfo, error) {
	procs, err := ListProcesses()
	if err != nil {
		return nil, err
	}
	slices.SortFunc(procs, func(a, b ProcessInfo) int {
		return int(b.MemoryKB - a.MemoryKB) // descending
	})
	if len(procs) > n {
		procs = procs[:n]
	}
	return procs, nil
}
