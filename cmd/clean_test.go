package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCleanRemovesProfilesRoot(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	var buf bytes.Buffer
	if err := runClean(profilesRoot, claudeDir, &buf); err != nil {
		t.Fatalf("runClean: %v", err)
	}

	if _, err := os.Stat(profilesRoot); !os.IsNotExist(err) {
		t.Errorf("expected profilesRoot %s to be removed", profilesRoot)
	}
}

func TestCleanRemovesSymlinksFromClaudeDir(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	var buf bytes.Buffer
	if err := runClean(profilesRoot, claudeDir, &buf); err != nil {
		t.Fatalf("runClean: %v", err)
	}

	for _, name := range []string{"skills", "commands", "agents"} {
		link := filepath.Join(claudeDir, name)
		if _, err := os.Lstat(link); !os.IsNotExist(err) {
			t.Errorf("expected symlink %s to be removed", link)
		}
	}
}

func TestCleanPrintsConfirmation(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	var buf bytes.Buffer
	if err := runClean(profilesRoot, claudeDir, &buf); err != nil {
		t.Fatalf("runClean: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("expected confirmation output, got nothing")
	}
}
