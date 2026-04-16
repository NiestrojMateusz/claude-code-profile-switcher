package symlink

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInspectRealDir(t *testing.T) {
	claudeDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(claudeDir, "skills"), 0o755)

	info, err := Inspect(claudeDir, Skills)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if info.Kind != KindReal {
		t.Errorf("Kind: want KindReal, got %v", info.Kind)
	}
}

func TestInspectSymlink(t *testing.T) {
	claudeDir := t.TempDir()
	target := t.TempDir()

	skillsPath := filepath.Join(claudeDir, "skills")
	if err := os.Symlink(target, skillsPath); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	info, err := Inspect(claudeDir, Skills)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if info.Kind != KindSymlink {
		t.Errorf("Kind: want KindSymlink, got %v", info.Kind)
	}
	if info.Target != target {
		t.Errorf("Target: want %q, got %q", target, info.Target)
	}
}

func TestInspectAbsent(t *testing.T) {
	claudeDir := t.TempDir()
	// no "skills" dir created inside

	info, err := Inspect(claudeDir, Skills)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if info.Kind != KindAbsent {
		t.Errorf("Kind: want KindAbsent, got %v", info.Kind)
	}
}

func TestSwitchUpdatesAllSymlinks(t *testing.T) {
	claudeDir := t.TempDir()
	profileA := t.TempDir()
	profileB := t.TempDir()

	// Create profile subdirs for both profiles.
	for _, sub := range []string{"skills", "commands", "agents"} {
		if err := os.MkdirAll(filepath.Join(profileA, sub), 0o755); err != nil {
			t.Fatalf("MkdirAll profileA/%s: %v", sub, err)
		}
		if err := os.MkdirAll(filepath.Join(profileB, sub), 0o755); err != nil {
			t.Fatalf("MkdirAll profileB/%s: %v", sub, err)
		}
	}

	// Pre-wire claudeDir → profileA.
	for _, sub := range []string{"skills", "commands", "agents"} {
		link := filepath.Join(claudeDir, sub)
		if err := os.Symlink(filepath.Join(profileA, sub), link); err != nil {
			t.Fatalf("Symlink to A: %v", err)
		}
	}

	// Switch to profileB.
	if err := Switch(claudeDir, profileB); err != nil {
		t.Fatalf("Switch: %v", err)
	}

	// All three symlinks must now point at profileB subdirs.
	for _, sub := range []string{"skills", "commands", "agents"} {
		info, err := Inspect(claudeDir, Target(sub))
		if err != nil {
			t.Fatalf("Inspect %s: %v", sub, err)
		}
		if info.Kind != KindSymlink {
			t.Errorf("%s: want KindSymlink, got %v", sub, info.Kind)
		}
		want := filepath.Join(profileB, sub)
		if info.Target != want {
			t.Errorf("%s target: want %q, got %q", sub, want, info.Target)
		}
	}
}

func TestSwitchMissingTargetSubdir(t *testing.T) {
	claudeDir := t.TempDir()
	profileA := t.TempDir()
	profileB := t.TempDir() // no subdirs inside

	for _, sub := range []string{"skills", "commands", "agents"} {
		if err := os.MkdirAll(filepath.Join(profileA, sub), 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		link := filepath.Join(claudeDir, sub)
		if err := os.Symlink(filepath.Join(profileA, sub), link); err != nil {
			t.Fatalf("Symlink: %v", err)
		}
	}

	// profileB has no subdirs — Switch should return an error.
	if err := Switch(claudeDir, profileB); err == nil {
		t.Error("Switch to profile with missing subdirs: want error, got nil")
	}
}
