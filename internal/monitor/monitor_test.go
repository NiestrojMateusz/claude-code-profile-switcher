package monitor

import (
	"os"
	"path/filepath"
	"testing"
)

func mkSkillDir(t *testing.T, root, name string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
}

func TestScanReturnsNewSkills(t *testing.T) {
	agentsDir := t.TempDir()
	mkSkillDir(t, agentsDir, "skill-new")
	mkSkillDir(t, agentsDir, "skill-old")

	known := []string{"skill-old"}
	got := Scan(agentsDir, known)

	if len(got) != 1 || got[0] != "skill-new" {
		t.Errorf("Scan: want [skill-new], got %v", got)
	}
}

func TestScanReturnsNilWhenNoNewSkills(t *testing.T) {
	agentsDir := t.TempDir()
	mkSkillDir(t, agentsDir, "skill-a")

	known := []string{"skill-a"}
	got := Scan(agentsDir, known)

	if len(got) != 0 {
		t.Errorf("Scan: want empty, got %v", got)
	}
}

func TestScanIgnoresFiles(t *testing.T) {
	agentsDir := t.TempDir()
	// a regular file — not a skill
	if err := os.WriteFile(filepath.Join(agentsDir, "README.md"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	mkSkillDir(t, agentsDir, "real-skill")

	got := Scan(agentsDir, nil)

	if len(got) != 1 || got[0] != "real-skill" {
		t.Errorf("Scan: want [real-skill], got %v", got)
	}
}

func TestScanEmptyDirReturnsNil(t *testing.T) {
	agentsDir := t.TempDir()
	got := Scan(agentsDir, nil)
	if len(got) != 0 {
		t.Errorf("Scan: want empty, got %v", got)
	}
}

func TestScanMissingDirReturnsNil(t *testing.T) {
	got := Scan("/does/not/exist", nil)
	if len(got) != 0 {
		t.Errorf("Scan: want empty on missing dir, got %v", got)
	}
}

// --- MarkKnown ---

func TestMarkKnownAddsToSlice(t *testing.T) {
	known := []string{"skill-a"}
	got := MarkKnown(known, []string{"skill-b", "skill-c"})

	if len(got) != 3 {
		t.Fatalf("MarkKnown: want 3 items, got %v", got)
	}
}

func TestMarkKnownIsIdempotent(t *testing.T) {
	known := []string{"skill-a"}
	got := MarkKnown(known, []string{"skill-a"})

	if len(got) != 1 {
		t.Errorf("MarkKnown: duplicated entry, got %v", got)
	}
}
