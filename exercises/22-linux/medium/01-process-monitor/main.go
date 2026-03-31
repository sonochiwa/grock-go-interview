package process_monitor

type ProcessInfo struct {
	PID      int
	Name     string
	MemoryKB int64
	State    string
}

// TODO: прочитай /proc/<pid>/status и заполни ProcessInfo
// Name: строка "Name:"
// State: строка "State:" (R/S/D/Z/T)
// MemoryKB: строка "VmRSS:" (в KB)
func GetProcessInfo(pid int) (*ProcessInfo, error) {
	return nil, nil
}

// TODO: пройдись по /proc/[0-9]* и собери инфо обо всех процессах
func ListProcesses() ([]ProcessInfo, error) {
	return nil, nil
}

// TODO: верни топ-N процессов по памяти (от большего к меньшему)
func TopByMemory(n int) ([]ProcessInfo, error) {
	return nil, nil
}
