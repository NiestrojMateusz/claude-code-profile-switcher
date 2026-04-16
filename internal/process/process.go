package process

import (
	"os/exec"
	"strconv"
	"strings"
)

// entry is an internal representation of one process row.
type entry struct {
	pid  int
	name string
}

// scanEntries filters entries whose name matches "claude".
func scanEntries(entries []entry) []int {
	var pids []int
	for _, e := range entries {
		if e.name == "claude" {
			pids = append(pids, e.pid)
		}
	}
	return pids
}

// Scan returns PIDs of all running processes named "claude".
// It uses `ps` on macOS/Linux.
func Scan() ([]int, error) {
	out, err := exec.Command("ps", "-eo", "pid,comm").Output()
	if err != nil {
		return nil, err
	}
	var entries []entry
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		// fields[1] may be a full path on some platforms; use base name.
		name := fields[1]
		if idx := strings.LastIndex(name, "/"); idx >= 0 {
			name = name[idx+1:]
		}
		entries = append(entries, entry{pid: pid, name: name})
	}
	return scanEntries(entries), nil
}
