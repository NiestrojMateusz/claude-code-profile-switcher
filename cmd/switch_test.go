package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matis/ccp/internal/config"
	"github.com/matis/ccp/internal/symlink"
)

// makeProfile creates a profile directory tree and registers it in config.
func makeProfile(t *testing.T, profilesRoot, name string) {
	t.Helper()
	for _, sub := range []string{"skills", "commands", "agents"} {
		if err := os.MkdirAll(filepath.Join(profilesRoot, name, sub), 0o755); err != nil {
			t.Fatalf("MkdirAll %s/%s: %v", name, sub, err)
		}
	}
	cfg, _ := config.Load(profilesRoot)
	cfg.Profiles = append(cfg.Profiles, name)
	if err := config.Save(profilesRoot, cfg); err != nil {
		t.Fatalf("Save config: %v", err)
	}
}

// wireSymlinks creates claudeDir symlinks pointing at a given profile.
func wireSymlinks(t *testing.T, claudeDir, profilesRoot, name string) {
	t.Helper()
	for _, sub := range []string{"skills", "commands", "agents"} {
		link := filepath.Join(claudeDir, sub)
		_ = os.Remove(link)
		target := filepath.Join(profilesRoot, name, sub)
		if err := os.Symlink(target, link); err != nil {
			t.Fatalf("Symlink %s: %v", sub, err)
		}
	}
}

func noProcesses() ([]int, error)       { return nil, nil }
func twoProcesses() ([]int, error)      { return []int{111, 222}, nil }
func confirmYes(_ []int) (bool, error)  { return true, nil }
func confirmNo(_ []int) (bool, error)   { return false, nil }

func TestRunSwitchProfileNotFound(t *testing.T) {
	root := t.TempDir()
	claude := t.TempDir()

	makeProfile(t, root, "base")
	wireSymlinks(t, claude, root, "base")

	var buf bytes.Buffer
	err := runSwitch(root, claude, "nonexistent", noProcesses, confirmYes, &buf)
	if err == nil {
		t.Fatal("want error for unknown profile, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention profile name: %v", err)
	}
}

func TestRunSwitchSuccess(t *testing.T) {
	root := t.TempDir()
	claude := t.TempDir()

	makeProfile(t, root, "base")
	makeProfile(t, root, "work")
	wireSymlinks(t, claude, root, "base")

	// Set base as active in config.
	cfg, _ := config.Load(root)
	cfg = config.SetActive(cfg, "base")
	_ = config.Save(root, cfg)

	var buf bytes.Buffer
	if err := runSwitch(root, claude, "work", noProcesses, confirmYes, &buf); err != nil {
		t.Fatalf("runSwitch: %v", err)
	}

	// Symlinks must point at work.
	for _, sub := range []string{"skills", "commands", "agents"} {
		info, err := symlink.Inspect(claude, symlink.Target(sub))
		if err != nil {
			t.Fatalf("Inspect %s: %v", sub, err)
		}
		want := filepath.Join(root, "work", sub)
		if info.Target != want {
			t.Errorf("%s target: want %q, got %q", sub, want, info.Target)
		}
	}

	// Config must record work as active.
	cfg2, _ := config.Load(root)
	if config.ActiveProfile(cfg2) != "work" {
		t.Errorf("active profile: want work, got %q", config.ActiveProfile(cfg2))
	}
}

func TestRunSwitchWarnsAndAbortsWhenProcessesRunning(t *testing.T) {
	root := t.TempDir()
	claude := t.TempDir()

	makeProfile(t, root, "base")
	makeProfile(t, root, "work")
	wireSymlinks(t, claude, root, "base")

	cfg, _ := config.Load(root)
	cfg = config.SetActive(cfg, "base")
	_ = config.Save(root, cfg)

	var buf bytes.Buffer
	err := runSwitch(root, claude, "work", twoProcesses, confirmNo, &buf)
	if err != nil {
		t.Fatalf("runSwitch with abort: want nil error, got %v", err)
	}

	// Output must mention PIDs.
	out := buf.String()
	if !strings.Contains(out, "111") || !strings.Contains(out, "222") {
		t.Errorf("expected PIDs in output, got: %q", out)
	}

	// Active profile must remain base (not switched).
	cfg2, _ := config.Load(root)
	if config.ActiveProfile(cfg2) != "base" {
		t.Errorf("should remain base, got %q", config.ActiveProfile(cfg2))
	}
}

func TestRunSwitchProceedsWhenUserConfirms(t *testing.T) {
	root := t.TempDir()
	claude := t.TempDir()

	makeProfile(t, root, "base")
	makeProfile(t, root, "work")
	wireSymlinks(t, claude, root, "base")

	cfg, _ := config.Load(root)
	cfg = config.SetActive(cfg, "base")
	_ = config.Save(root, cfg)

	var buf bytes.Buffer
	if err := runSwitch(root, claude, "work", twoProcesses, confirmYes, &buf); err != nil {
		t.Fatalf("runSwitch with confirm: %v", err)
	}

	cfg2, _ := config.Load(root)
	if config.ActiveProfile(cfg2) != "work" {
		t.Errorf("want work, got %q", config.ActiveProfile(cfg2))
	}
}
