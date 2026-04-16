package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helpers reuse initBase from profile_test.go (same package)

// --- skill add ---

func TestSkillAddCreatesSymlinkInActiveProfile(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	src := t.TempDir() // fake skill directory

	var buf bytes.Buffer
	if err := runSkillAdd(profilesRoot, "base", src, &buf); err != nil {
		t.Fatalf("runSkillAdd: %v", err)
	}

	link := filepath.Join(profilesRoot, "base", "skills", filepath.Base(src))
	fi, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("Lstat %s: %v", link, err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected symlink at %s", link)
	}
}

func TestSkillAddErrorsOnNonExistentSrc(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	var buf bytes.Buffer
	err := runSkillAdd(profilesRoot, "base", "/does/not/exist", &buf)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- skill list ---

func TestSkillListShowsOwnedSkill(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	// Add a real skill dir to base/skills
	if err := os.MkdirAll(filepath.Join(profilesRoot, "base", "skills", "my-skill"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	var buf bytes.Buffer
	if err := runSkillList(profilesRoot, "base", &buf); err != nil {
		t.Fatalf("runSkillList: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "my-skill") {
		t.Errorf("expected 'my-skill' in output, got: %q", out)
	}
	if !strings.Contains(out, "owned") {
		t.Errorf("expected 'owned' label in output, got: %q", out)
	}
}

func TestSkillListShowsInheritedSkill(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	// Create base skill + work profile that inherits it
	baseSkill := filepath.Join(profilesRoot, "base", "skills", "shared")
	if err := os.MkdirAll(baseSkill, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := runProfileCreate(profilesRoot, "work", claudeDir, noEditorPicker, &bytes.Buffer{}); err != nil {
		t.Fatalf("runProfileCreate: %v", err)
	}

	var buf bytes.Buffer
	if err := runSkillList(profilesRoot, "work", &buf); err != nil {
		t.Fatalf("runSkillList: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "inherited") {
		t.Errorf("expected 'inherited' label in output, got: %q", out)
	}
}

func TestSkillListEmptyProfile(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	var buf bytes.Buffer
	if err := runSkillList(profilesRoot, "base", &buf); err != nil {
		t.Fatalf("runSkillList: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "no skills") {
		t.Errorf("expected 'no skills' message, got: %q", out)
	}
}
