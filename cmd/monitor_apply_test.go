package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matis/ccp/internal/config"
	tuimonitor "github.com/matis/ccp/tui/monitor"
)

func TestApplyChoicesAddsSkillToProfile(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	agentsDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	// Create a skill in agentsDir
	skillSrc := filepath.Join(agentsDir, "cool-skill")
	if err := os.MkdirAll(skillSrc, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	choices := []tuimonitor.Choice{
		{Skill: "cool-skill", Action: tuimonitor.ActionAdd, Profile: "base"},
	}

	if err := applyChoices(profilesRoot, agentsDir, choices); err != nil {
		t.Fatalf("applyChoices: %v", err)
	}

	link := filepath.Join(profilesRoot, "base", "skills", "cool-skill")
	if fi, err := os.Lstat(link); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected symlink at %s", link)
	}
}

func TestApplyChoicesMarksNeverAsKnown(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	agentsDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	choices := []tuimonitor.Choice{
		{Skill: "boring-skill", Action: tuimonitor.ActionNever},
	}

	if err := applyChoices(profilesRoot, agentsDir, choices); err != nil {
		t.Fatalf("applyChoices: %v", err)
	}

	cfg, err := config.Load(profilesRoot)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	found := false
	for _, k := range cfg.KnownSkills {
		if k == "boring-skill" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'boring-skill' in KnownSkills, got %v", cfg.KnownSkills)
	}
}

func TestApplyChoicesSkipDoesNothing(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	choices := []tuimonitor.Choice{
		{Skill: "meh-skill", Action: tuimonitor.ActionSkip},
	}

	if err := applyChoices(profilesRoot, "", choices); err != nil {
		t.Fatalf("applyChoices: %v", err)
	}

	cfg, _ := config.Load(profilesRoot)
	for _, k := range cfg.KnownSkills {
		if k == "meh-skill" {
			t.Error("Skip should not add to KnownSkills")
		}
	}
}
