package profile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matis/ccp/internal/config"
)

func TestCreateMakesSubdirs(t *testing.T) {
	profilesRoot := t.TempDir()

	if err := Create(profilesRoot, "base"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	for _, sub := range []string{"skills", "commands", "agents"} {
		dir := filepath.Join(profilesRoot, "base", sub)
		if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
			t.Errorf("expected dir %s to exist", dir)
		}
	}
}

func TestCreateWritesConfig(t *testing.T) {
	profilesRoot := t.TempDir()

	if err := Create(profilesRoot, "base"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	cfg, err := config.Load(profilesRoot)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	if config.ActiveProfile(cfg) != "base" {
		t.Errorf("ActiveProfile: want %q, got %q", "base", config.ActiveProfile(cfg))
	}

	found := false
	for _, p := range cfg.Profiles {
		if p == "base" {
			found = true
		}
	}
	if !found {
		t.Errorf("'base' not in Profiles list: %v", cfg.Profiles)
	}
}

// --- CreateChild ---

func TestCreateChildMakesSubdirs(t *testing.T) {
	profilesRoot := t.TempDir()
	setupProfile(t, profilesRoot, "base")

	if err := CreateChild(profilesRoot, "base", "work"); err != nil {
		t.Fatalf("CreateChild: %v", err)
	}

	for _, sub := range []string{"skills", "commands", "agents"} {
		dir := filepath.Join(profilesRoot, "work", sub)
		if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
			t.Errorf("expected dir %s", dir)
		}
	}
}

func TestCreateChildDoesNotChangeActiveProfile(t *testing.T) {
	profilesRoot := t.TempDir()
	setupProfile(t, profilesRoot, "base") // active = base

	if err := CreateChild(profilesRoot, "base", "work"); err != nil {
		t.Fatalf("CreateChild: %v", err)
	}

	cfg, err := config.Load(profilesRoot)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if cfg.ActiveProfile != "base" {
		t.Errorf("ActiveProfile changed: want %q, got %q", "base", cfg.ActiveProfile)
	}
}

func TestCreateChildSymlinksBaseSkills(t *testing.T) {
	profilesRoot := t.TempDir()
	setupProfile(t, profilesRoot, "base")

	// Add a skill to base
	skillDir := filepath.Join(profilesRoot, "base", "skills", "my-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}

	if err := CreateChild(profilesRoot, "base", "work"); err != nil {
		t.Fatalf("CreateChild: %v", err)
	}

	// work/skills/my-skill should be a symlink resolving to base/skills/my-skill
	link := filepath.Join(profilesRoot, "work", "skills", "my-skill")
	fi, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("Lstat %s: %v", link, err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected symlink at %s", link)
	}
	target, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("Readlink: %v", err)
	}
	resolved := filepath.Join(filepath.Dir(link), target)
	if resolved != skillDir {
		t.Errorf("symlink resolves to %q; want %q", resolved, skillDir)
	}
}

func TestCreateChildAddsToConfigProfiles(t *testing.T) {
	profilesRoot := t.TempDir()
	setupProfile(t, profilesRoot, "base")

	if err := CreateChild(profilesRoot, "base", "work"); err != nil {
		t.Fatalf("CreateChild: %v", err)
	}

	cfg, err := config.Load(profilesRoot)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	found := false
	for _, p := range cfg.Profiles {
		if p == "work" {
			found = true
		}
	}
	if !found {
		t.Errorf("'work' not in config.Profiles: %v", cfg.Profiles)
	}
}

// --- Delete ---

func setupProfile(t *testing.T, profilesRoot, name string) {
	t.Helper()
	if err := Create(profilesRoot, name); err != nil {
		t.Fatalf("Create(%q): %v", name, err)
	}
}

func TestDeleteRefusesActiveProfile(t *testing.T) {
	profilesRoot := t.TempDir()
	setupProfile(t, profilesRoot, "base")
	// "base" is active after Create

	err := Delete(profilesRoot, "base")
	if err == nil {
		t.Fatal("Delete active profile: want error, got nil")
	}
}

func TestDeleteRefusesBaseWhenDependentsExist(t *testing.T) {
	profilesRoot := t.TempDir()
	setupProfile(t, profilesRoot, "base")
	setupProfile(t, profilesRoot, "work") // sets active to "work"

	// Manually set active to "work" so base is not active
	cfg, _ := config.Load(profilesRoot)
	cfg = config.SetActive(cfg, "work")
	_ = config.Save(profilesRoot, cfg)

	err := Delete(profilesRoot, "base")
	if err == nil {
		t.Fatal("Delete base with dependents: want error, got nil")
	}
}

func TestDeleteRemovesProfileDirAndConfig(t *testing.T) {
	profilesRoot := t.TempDir()
	setupProfile(t, profilesRoot, "base")
	setupProfile(t, profilesRoot, "work") // active is now "work"; switch back to base

	cfg, _ := config.Load(profilesRoot)
	cfg = config.SetActive(cfg, "base")
	_ = config.Save(profilesRoot, cfg)

	if err := Delete(profilesRoot, "work"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// dir should be gone
	if _, err := os.Stat(filepath.Join(profilesRoot, "work")); !os.IsNotExist(err) {
		t.Error("expected work dir to be removed")
	}

	// config should no longer list "work"
	cfg, err := config.Load(profilesRoot)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	for _, p := range cfg.Profiles {
		if p == "work" {
			t.Error("expected 'work' removed from config.Profiles")
		}
	}
}
