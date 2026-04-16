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

// noopPicker is a stub picker for tests — selects nothing.
func noopPicker(_ []string, _, _ int, _ string) ([]string, error) { return nil, nil }

func TestInitSucceedsWhenClaudeDirDoesNotExist(t *testing.T) {
	profilesRoot := t.TempDir()
	// claudeDir intentionally not created — simulates fresh system or custom --claude-dir
	claudeDir := filepath.Join(t.TempDir(), "nonexistent-claude")

	var buf bytes.Buffer
	if err := runInit(profilesRoot, claudeDir, noopPicker, &buf); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	if !strings.Contains(buf.String(), "initialised") {
		t.Errorf("expected initialised message, got: %q", buf.String())
	}
}

func TestInitCreatesBaseProfile(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()

	// simulate an existing claude setup with real directories
	for _, d := range []string{"skills", "commands", "agents"} {
		if err := os.MkdirAll(filepath.Join(claudeDir, d), 0o755); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}

	var buf bytes.Buffer
	if err := runInit(profilesRoot, claudeDir, noopPicker, &buf); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	// config.json records base as active profile
	cfg, err := config.Load(profilesRoot)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if config.ActiveProfile(cfg) != "base" {
		t.Errorf("ActiveProfile: want %q, got %q", "base", config.ActiveProfile(cfg))
	}

	// each managed path is now a symlink pointing into base/
	for _, target := range symlink.AllTargets {
		info, err := symlink.Inspect(claudeDir, target)
		if err != nil {
			t.Fatalf("Inspect %s: %v", target, err)
		}
		if info.Kind != symlink.KindSymlink {
			t.Errorf("%s: want KindSymlink, got %v", target, info.Kind)
		}
		want := filepath.Join(profilesRoot, "base", string(target))
		if info.Target != want {
			t.Errorf("%s target: want %q, got %q", target, want, info.Target)
		}
	}
}

func TestInitRunsThreeStepPickerAndSymlinksCommands(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()

	// Seed a backed-up commands dir with a .md file.
	commandsBackup := filepath.Join(claudeDir, "commands_backup_1234567890")
	if err := os.MkdirAll(commandsBackup, 0o755); err != nil {
		t.Fatalf("mkdir commands backup: %v", err)
	}
	commandFile := filepath.Join(commandsBackup, "deploy.md")
	if err := os.WriteFile(commandFile, []byte("# deploy"), 0o644); err != nil {
		t.Fatalf("write command file: %v", err)
	}

	// Spy picker: records step calls, selects all items.
	type stepCall struct{ step, total int; title string }
	var calls []stepCall
	spyPicker := func(available []string, step, total int, title string) ([]string, error) {
		calls = append(calls, stepCall{step, total, title})
		return available, nil
	}

	var buf bytes.Buffer
	if err := runInit(profilesRoot, claudeDir, spyPicker, &buf); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	// Picker must be called 3 times with the right step headers.
	if len(calls) != 3 {
		t.Fatalf("want 3 picker calls, got %d: %v", len(calls), calls)
	}
	if calls[0] != (stepCall{1, 3, "Skills"}) {
		t.Errorf("step 1: want {1 3 Skills}, got %+v", calls[0])
	}
	if calls[1] != (stepCall{2, 3, "Commands"}) {
		t.Errorf("step 2: want {2 3 Commands}, got %+v", calls[1])
	}
	if calls[2] != (stepCall{3, 3, "Agents"}) {
		t.Errorf("step 3: want {3 3 Agents}, got %+v", calls[2])
	}

	// Selected command must be symlinked into base/commands/.
	link := filepath.Join(profilesRoot, "base", "commands", "deploy.md")
	if _, err := os.Lstat(link); err != nil {
		t.Errorf("expected symlink at %s, got: %v", link, err)
	}
}

func TestInitAlreadyInitialised(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()

	// first init
	if err := runInit(profilesRoot, claudeDir, noopPicker, &bytes.Buffer{}); err != nil {
		t.Fatalf("first runInit: %v", err)
	}

	// second init must exit cleanly without error
	var buf bytes.Buffer
	if err := runInit(profilesRoot, claudeDir, noopPicker, &buf); err != nil {
		t.Fatalf("second runInit: %v", err)
	}
	if got := buf.String(); !strings.Contains(got, "already initialised") {
		t.Errorf("expected 'already initialised' message, got: %q", got)
	}
}
