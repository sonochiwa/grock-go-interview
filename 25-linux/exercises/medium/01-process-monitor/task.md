# Process Monitor

Реализуй мониторинг процессов через `/proc` (Linux) или `os/exec` (кроссплатформенно):

- `ProcessInfo{PID int, Name string, MemoryKB int64, State string}`
- `GetProcessInfo(pid int) (*ProcessInfo, error)` — инфо о процессе
- `ListProcesses() ([]ProcessInfo, error)` — все процессы
- `TopByMemory(n int) ([]ProcessInfo, error)` — топ-N по памяти

Парси `/proc/<pid>/status` для Linux.
