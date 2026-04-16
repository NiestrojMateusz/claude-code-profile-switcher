package process

import (
	"testing"
)

func TestScanWithFakeList(t *testing.T) {
	// Scan accepts an injectable lister so tests don't shell out.
	// Fake lister returns a predictable set of process names → PIDs.
	entries := []entry{
		{pid: 100, name: "bash"},
		{pid: 200, name: "claude"},
		{pid: 300, name: "claude"},
		{pid: 400, name: "vim"},
	}
	pids := scanEntries(entries)
	if len(pids) != 2 {
		t.Fatalf("want 2 claude PIDs, got %d: %v", len(pids), pids)
	}
	if pids[0] != 200 || pids[1] != 300 {
		t.Errorf("want [200 300], got %v", pids)
	}
}

func TestScanNoClaudeProcesses(t *testing.T) {
	entries := []entry{
		{pid: 1, name: "launchd"},
		{pid: 2, name: "kernel_task"},
	}
	pids := scanEntries(entries)
	if len(pids) != 0 {
		t.Errorf("want empty, got %v", pids)
	}
}
