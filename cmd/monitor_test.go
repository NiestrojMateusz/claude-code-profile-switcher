package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	tuimonitor "github.com/matis/ccp/tui/monitor"
)

// stubPicker returns a picker that always returns the given choices.
func stubPicker(choices []tuimonitor.Choice) MonitorPicker {
	return func(_, _ []string) ([]tuimonitor.Choice, error) {
		return choices, nil
	}
}

// cancelPicker simulates the user quitting the TUI (returns nil).
var cancelPicker MonitorPicker = func(_, _ []string) ([]tuimonitor.Choice, error) {
	return nil, nil
}

func TestMonitorCheckSkipsPickerWhenNoProfilesExist(t *testing.T) {
	profilesRoot := t.TempDir()
	agentsDir := t.TempDir()
	// No initBase — no profiles in config.

	if err := os.MkdirAll(filepath.Join(agentsDir, "shiny-skill"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	called := false
	neverCalled := MonitorPicker(func(_, _ []string) ([]tuimonitor.Choice, error) {
		called = true
		return nil, nil
	})

	if err := runMonitorCheck(profilesRoot, agentsDir, neverCalled); err != nil {
		t.Fatalf("runMonitorCheck: %v", err)
	}

	if called {
		t.Error("picker must not be called when no profiles exist")
	}
}

func TestMonitorOnCommandSkipsInit(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	agentsDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	if err := os.MkdirAll(filepath.Join(agentsDir, "shiny-skill"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	called := false
	neverCalled := MonitorPicker(func(_, _ []string) ([]tuimonitor.Choice, error) {
		called = true
		return nil, nil
	})

	if err := runMonitorOnCommand("init", profilesRoot, agentsDir, neverCalled); err != nil {
		t.Fatalf("runMonitorOnCommand: %v", err)
	}

	if called {
		t.Error("picker must not be called when command is init")
	}
}

func TestMonitorOnCommandSkipsClean(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	agentsDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	if err := os.MkdirAll(filepath.Join(agentsDir, "shiny-skill"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	called := false
	neverCalled := MonitorPicker(func(_, _ []string) ([]tuimonitor.Choice, error) {
		called = true
		return nil, nil
	})

	if err := runMonitorOnCommand("clean", profilesRoot, agentsDir, neverCalled); err != nil {
		t.Fatalf("runMonitorOnCommand: %v", err)
	}

	if called {
		t.Error("picker must not be called when command is clean")
	}
}

func TestMonitorOnCommandFiresOnlyForSwitch(t *testing.T) {
	silentCommands := []string{"status", "profile", "skill", "list", "create", "delete", "edit"}

	for _, cmdName := range silentCommands {
		t.Run(cmdName, func(t *testing.T) {
			profilesRoot := t.TempDir()
			claudeDir := t.TempDir()
			agentsDir := t.TempDir()
			initBase(t, profilesRoot, claudeDir)

			if err := os.MkdirAll(filepath.Join(agentsDir, "shiny-skill"), 0o755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}

			called := false
			neverCalled := MonitorPicker(func(_, _ []string) ([]tuimonitor.Choice, error) {
				called = true
				return nil, nil
			})

			if err := runMonitorOnCommand(cmdName, profilesRoot, agentsDir, neverCalled); err != nil {
				t.Fatalf("runMonitorOnCommand(%s): %v", cmdName, err)
			}

			if called {
				t.Errorf("picker must not be called for %q", cmdName)
			}
		})
	}
}

func TestMonitorOnCommandFiresForSwitch(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	agentsDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	if err := os.MkdirAll(filepath.Join(agentsDir, "shiny-skill"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	called := false
	capturePicker := MonitorPicker(func(_, _ []string) ([]tuimonitor.Choice, error) {
		called = true
		return nil, nil
	})

	if err := runMonitorOnCommand("switch", profilesRoot, agentsDir, capturePicker); err != nil {
		t.Fatalf("runMonitorOnCommand: %v", err)
	}

	if !called {
		t.Error("picker must be called for switch when new skills exist")
	}
}

func TestMonitorOnCommandPropagatesPickerError(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	agentsDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	if err := os.MkdirAll(filepath.Join(agentsDir, "shiny-skill"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	boom := errors.New("picker exploded")
	errPicker := MonitorPicker(func(_, _ []string) ([]tuimonitor.Choice, error) {
		return nil, boom
	})

	err := runMonitorOnCommand("switch", profilesRoot, agentsDir, errPicker)
	if !errors.Is(err, boom) {
		t.Errorf("want picker error propagated, got %v", err)
	}
}

func TestMonitorCheckExcludesSkillsAlreadyInBase(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	agentsDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	// Skill exists in agentsDir AND is already symlinked into base/skills/.
	skillSrc := filepath.Join(agentsDir, "existing-skill")
	if err := os.MkdirAll(skillSrc, 0o755); err != nil {
		t.Fatalf("mkdir agentsDir skill: %v", err)
	}
	baseLink := filepath.Join(profilesRoot, "base", "skills", "existing-skill")
	if err := os.Symlink(skillSrc, baseLink); err != nil {
		t.Fatalf("symlink into base: %v", err)
	}

	var pickerGotSkills []string
	capturePicker := MonitorPicker(func(newSkills, _ []string) ([]tuimonitor.Choice, error) {
		pickerGotSkills = newSkills
		return nil, nil
	})

	if err := runMonitorCheck(profilesRoot, agentsDir, capturePicker); err != nil {
		t.Fatalf("runMonitorCheck: %v", err)
	}

	for _, s := range pickerGotSkills {
		if s == "existing-skill" {
			t.Error("picker must not receive skills already present in base profile")
		}
	}
}

func TestMonitorCheckCallsPickerWithNewSkills(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	agentsDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	if err := os.MkdirAll(filepath.Join(agentsDir, "shiny-skill"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	var pickerGotSkills []string
	capturePicker := MonitorPicker(func(newSkills, _ []string) ([]tuimonitor.Choice, error) {
		pickerGotSkills = newSkills
		return nil, nil
	})

	if err := runMonitorCheck(profilesRoot, agentsDir, capturePicker); err != nil {
		t.Fatalf("runMonitorCheck: %v", err)
	}

	if len(pickerGotSkills) != 1 || pickerGotSkills[0] != "shiny-skill" {
		t.Errorf("picker got %v, want [shiny-skill]", pickerGotSkills)
	}
}

func TestMonitorCheckSilentWhenNoNewSkills(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	agentsDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	called := false
	neverCalled := MonitorPicker(func(_, _ []string) ([]tuimonitor.Choice, error) {
		called = true
		return nil, nil
	})

	if err := runMonitorCheck(profilesRoot, agentsDir, neverCalled); err != nil {
		t.Fatalf("runMonitorCheck: %v", err)
	}

	if called {
		t.Error("picker should not be called when no new skills")
	}
}

func TestMonitorCheckSilentWhenAgentsDirMissing(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	called := false
	neverCalled := MonitorPicker(func(_, _ []string) ([]tuimonitor.Choice, error) {
		called = true
		return nil, nil
	})

	if err := runMonitorCheck(profilesRoot, "/does/not/exist", neverCalled); err != nil {
		t.Fatalf("runMonitorCheck: %v", err)
	}

	if called {
		t.Error("picker should not be called when agents dir missing")
	}
}

func TestMonitorCheckAppliesChoicesFromPicker(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	agentsDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	skillSrc := filepath.Join(agentsDir, "new-skill")
	if err := os.MkdirAll(skillSrc, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	picker := stubPicker([]tuimonitor.Choice{
		{Skill: "new-skill", Action: tuimonitor.ActionAdd, Profile: "base"},
	})

	if err := runMonitorCheck(profilesRoot, agentsDir, picker); err != nil {
		t.Fatalf("runMonitorCheck: %v", err)
	}

	link := filepath.Join(profilesRoot, "base", "skills", "new-skill")
	if fi, err := os.Lstat(link); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected symlink at %s after ActionAdd", link)
	}
}

func TestMonitorCheckCancelDoesNotApply(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	agentsDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	if err := os.MkdirAll(filepath.Join(agentsDir, "new-skill"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := runMonitorCheck(profilesRoot, agentsDir, cancelPicker); err != nil {
		t.Fatalf("runMonitorCheck: %v", err)
	}

	link := filepath.Join(profilesRoot, "base", "skills", "new-skill")
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Error("skill should not be added when picker returns nil (cancel)")
	}
}
