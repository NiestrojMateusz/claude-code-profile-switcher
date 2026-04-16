package config

import (
	"testing"
)

func TestSetActiveAndActiveProfile(t *testing.T) {
	root := t.TempDir()

	cfg := Config{Profiles: []string{"base", "work"}}
	cfg = SetActive(cfg, "work")

	if ActiveProfile(cfg) != "work" {
		t.Errorf("ActiveProfile: want %q, got %q", "work", ActiveProfile(cfg))
	}

	// persist and reload to confirm it survives a Save+Load cycle
	if err := Save(root, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if ActiveProfile(loaded) != "work" {
		t.Errorf("after reload ActiveProfile: want %q, got %q", "work", ActiveProfile(loaded))
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	root := t.TempDir()

	original := Config{
		ActiveProfile: "work",
		Profiles:      []string{"base", "work"},
		KnownSkills:   []string{"git-helper"},
	}

	if err := Save(root, original); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	got, err := Load(root)
	if err != nil {
		t.Fatalf("Load after Save failed: %v", err)
	}

	if got.ActiveProfile != original.ActiveProfile {
		t.Errorf("ActiveProfile: want %q, got %q", original.ActiveProfile, got.ActiveProfile)
	}
	if len(got.Profiles) != len(original.Profiles) {
		t.Errorf("Profiles length: want %d, got %d", len(original.Profiles), len(got.Profiles))
	}
	if len(got.KnownSkills) != len(original.KnownSkills) {
		t.Errorf("KnownSkills length: want %d, got %d", len(original.KnownSkills), len(got.KnownSkills))
	}
}

func TestLoadMissingFile(t *testing.T) {
	root := t.TempDir()

	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load on missing file returned error: %v", err)
	}
	if cfg.ActiveProfile != "" {
		t.Errorf("expected empty ActiveProfile, got %q", cfg.ActiveProfile)
	}
	if len(cfg.Profiles) != 0 {
		t.Errorf("expected empty Profiles, got %v", cfg.Profiles)
	}
}
