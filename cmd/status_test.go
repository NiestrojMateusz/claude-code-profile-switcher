package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootFlagsOverrideDefaultPaths(t *testing.T) {
	customRoot := t.TempDir()
	customClaude := t.TempDir()

	// Restore globals after test so other tests are not affected.
	origRoot, origClaude := defaultProfilesRoot, defaultClaudeDir
	t.Cleanup(func() { defaultProfilesRoot, defaultClaudeDir = origRoot, origClaude })

	rootCmd.SetArgs([]string{
		"--profiles-root", customRoot,
		"--claude-dir", customClaude,
		"status",
	})
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if defaultProfilesRoot != customRoot {
		t.Errorf("defaultProfilesRoot: want %q, got %q", customRoot, defaultProfilesRoot)
	}
	if defaultClaudeDir != customClaude {
		t.Errorf("defaultClaudeDir: want %q, got %q", customClaude, defaultClaudeDir)
	}
	if !strings.Contains(out.String(), "no active profile") {
		t.Errorf("expected status output, got: %q", out.String())
	}
}

func TestStatusNoConfig(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()

	var buf bytes.Buffer
	err := runStatus(profilesRoot, claudeDir, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	want := "no active profile"
	if !strings.Contains(got, want) {
		t.Errorf("expected output to contain %q, got: %q", want, got)
	}
}
