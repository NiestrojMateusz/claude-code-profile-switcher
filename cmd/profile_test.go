package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matis/ccp/internal/config"
	"github.com/matis/ccp/internal/profile"
)

// helpers

// noEditorPicker selects all items (simulates user confirming the full pre-selection).
func noEditorPicker(items, _ []string, _, _ int, _ string) ([]string, error) { return items, nil }

func initBase(t *testing.T, profilesRoot, claudeDir string) {
	t.Helper()
	for _, d := range []string{"skills", "commands", "agents"} {
		if err := os.MkdirAll(filepath.Join(claudeDir, d), 0o755); err != nil {
			t.Fatalf("setup claude dir: %v", err)
		}
	}
	if err := runInit(profilesRoot, claudeDir, noopPicker, &bytes.Buffer{}); err != nil {
		t.Fatalf("runInit: %v", err)
	}
}

// --- profile list ---

func TestProfileListShowsActiveMarker(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	var buf bytes.Buffer
	if err := runProfileList(profilesRoot, &buf); err != nil {
		t.Fatalf("runProfileList: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "*") {
		t.Error("expected active marker '*' in output")
	}
	if !strings.Contains(out, "base") {
		t.Error("expected 'base' in output")
	}
}

func TestProfileListShowsAllProfiles(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	// create a second profile
	if err := profile.CreateChild(profilesRoot, "base", "work"); err != nil {
		t.Fatalf("CreateChild: %v", err)
	}

	var buf bytes.Buffer
	if err := runProfileList(profilesRoot, &buf); err != nil {
		t.Fatalf("runProfileList: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "work") {
		t.Errorf("expected 'work' in output, got: %q", out)
	}
}

// --- profile create ---

func TestProfileCreateRunsThreeStepPickerAndSymlinksCommands(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	// Seed a backed-up commands dir.
	commandsBackup := filepath.Join(claudeDir, "commands_backup_1234567890")
	if err := os.MkdirAll(commandsBackup, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	commandFile := filepath.Join(commandsBackup, "release.md")
	if err := os.WriteFile(commandFile, []byte("# release"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	type stepCall struct{ step, total int; title string }
	var calls []stepCall
	spyPicker := func(available, _ []string, step, total int, title string) ([]string, error) {
		calls = append(calls, stepCall{step, total, title})
		return available, nil // select all
	}

	var buf bytes.Buffer
	if err := runProfileCreate(profilesRoot, "work", claudeDir, spyPicker, &buf); err != nil {
		t.Fatalf("runProfileCreate: %v", err)
	}

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

	// Selected command must be symlinked into work/commands/.
	link := filepath.Join(profilesRoot, "work", "commands", "release.md")
	if _, err := os.Lstat(link); err != nil {
		t.Errorf("expected symlink at %s, got: %v", link, err)
	}
}

func TestProfileCreateMakesDirs(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	var buf bytes.Buffer
	if err := runProfileCreate(profilesRoot, "work", claudeDir, noEditorPicker, &buf); err != nil {
		t.Fatalf("runProfileCreate: %v", err)
	}

	for _, sub := range []string{"skills", "commands", "agents"} {
		dir := filepath.Join(profilesRoot, "work", sub)
		if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
			t.Errorf("expected dir %s", dir)
		}
	}
}

func TestProfileCreateOffersExternalSkillsFromAgentsDir(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	// External skill in a backup dir (how discoverSkills finds original skills).
	backupDir := filepath.Join(claudeDir, "skills_backup_999")
	if err := os.MkdirAll(filepath.Join(backupDir, "external-skill"), 0o755); err != nil {
		t.Fatalf("mkdir backup skill: %v", err)
	}

	var pickerGotItems []string
	capturePicker := func(items, _ []string, _, _ int, _ string) ([]string, error) {
		pickerGotItems = items
		return nil, nil // cancel — we only care what was offered (step 1)
	}

	var buf bytes.Buffer
	if err := runProfileCreate(profilesRoot, "work", claudeDir, capturePicker, &buf); err != nil {
		t.Fatalf("runProfileCreate: %v", err)
	}

	found := false
	for _, item := range pickerGotItems {
		if item == "external-skill" {
			found = true
		}
	}
	if !found {
		t.Errorf("picker must receive external skills; got %v", pickerGotItems)
	}
}

func TestProfileCreateAddsExternalSkillViaAddLocal(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	backupDir := filepath.Join(claudeDir, "skills_backup_999")
	extSkill := filepath.Join(backupDir, "external-skill")
	if err := os.MkdirAll(extSkill, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// picker selects the external skill
	selectExternal := func(_, _ []string, _, _ int, _ string) ([]string, error) {
		return []string{"external-skill"}, nil
	}

	var buf bytes.Buffer
	if err := runProfileCreate(profilesRoot, "work", claudeDir, selectExternal, &buf); err != nil {
		t.Fatalf("runProfileCreate: %v", err)
	}

	link := filepath.Join(profilesRoot, "work", "skills", "external-skill")
	fi, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("Lstat: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected symlink for external-skill at %s", link)
	}
}

func TestProfileCreateInheritsOnlyPickedSkills(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	for _, s := range []string{"skill-a", "skill-b"} {
		if err := os.MkdirAll(filepath.Join(profilesRoot, "base", "skills", s), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	// picker selects only skill-a
	onlyA := func(_, _ []string, _, _ int, _ string) ([]string, error) { return []string{"skill-a"}, nil }

	var buf bytes.Buffer
	if err := runProfileCreate(profilesRoot, "work", claudeDir, onlyA, &buf); err != nil {
		t.Fatalf("runProfileCreate: %v", err)
	}

	linkA := filepath.Join(profilesRoot, "work", "skills", "skill-a")
	if fi, err := os.Lstat(linkA); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected symlink for skill-a at %s", linkA)
	}

	linkB := filepath.Join(profilesRoot, "work", "skills", "skill-b")
	if _, err := os.Lstat(linkB); !os.IsNotExist(err) {
		t.Errorf("skill-b must not be inherited when not picked")
	}
}

func TestProfileCreateDoesNotChangeActiveProfile(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	var buf bytes.Buffer
	if err := runProfileCreate(profilesRoot, "work", claudeDir, noEditorPicker, &buf); err != nil {
		t.Fatalf("runProfileCreate: %v", err)
	}

	cfg, err := config.Load(profilesRoot)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if cfg.ActiveProfile != "base" {
		t.Errorf("ActiveProfile changed: want %q, got %q", "base", cfg.ActiveProfile)
	}
}

// --- profile edit ---

// stubEditor returns a picker that always confirms the given selection.
func stubEditor(returnSelected []string) func([]string, []string, int, int, string) ([]string, error) {
	return func(_, _ []string, _, _ int, _ string) ([]string, error) { return returnSelected, nil }
}

// cancelEditor simulates the user pressing Esc (returns nil).
func cancelEditor(_, _ []string, _, _ int, _ string) ([]string, error) { return nil, nil }

func TestProfileEditRunsThreeStepPickerWithPreSelectedCommands(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	// Seed a backed-up commands dir.
	commandsBackup := filepath.Join(claudeDir, "commands_backup_1234567890")
	if err := os.MkdirAll(commandsBackup, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	commandFile := filepath.Join(commandsBackup, "deploy.md")
	if err := os.WriteFile(commandFile, []byte("# deploy"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Create work profile (no commands selected initially).
	noCommandsPicker := func(_, _ []string, _, _ int, title string) ([]string, error) {
		if title == "Skills" {
			return []string{}, nil // no skills, but confirmed
		}
		return []string{}, nil // no commands/agents either
	}
	if err := runProfileCreate(profilesRoot, "work", claudeDir, noCommandsPicker, &bytes.Buffer{}); err != nil {
		t.Fatalf("runProfileCreate: %v", err)
	}

	// Manually add the command symlink to work/commands/ (simulates a prior edit that selected it).
	commandsDir := filepath.Join(profilesRoot, "work", "commands")
	link := filepath.Join(commandsDir, "deploy.md")
	if err := os.Symlink(commandFile, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	// Spy picker: record pre-selections per step.
	type stepCall struct {
		preSelected []string
		title       string
	}
	var calls []stepCall
	spyPicker := func(_, preSelected []string, _, _ int, title string) ([]string, error) {
		calls = append(calls, stepCall{preSelected, title})
		if preSelected == nil {
			return []string{}, nil // confirm with empty selection (non-nil = confirmed, nil = cancelled)
		}
		return preSelected, nil
	}

	var buf bytes.Buffer
	if err := runProfileEdit(profilesRoot, "work", claudeDir, spyPicker, &buf); err != nil {
		t.Fatalf("runProfileEdit: %v", err)
	}

	if len(calls) != 3 {
		t.Fatalf("want 3 picker calls, got %d", len(calls))
	}

	// Step 2 (Commands) must pre-select "deploy.md".
	commandsPreSelected := calls[1].preSelected
	found := false
	for _, name := range commandsPreSelected {
		if name == "deploy.md" {
			found = true
		}
	}
	if !found {
		t.Errorf("step 2 (Commands) must pre-select deploy.md; got %v", commandsPreSelected)
	}
}

func TestProfileEditDeselectingCommandRemovesSymlink(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	commandsBackup := filepath.Join(claudeDir, "commands_backup_1234567890")
	if err := os.MkdirAll(commandsBackup, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	commandFile := filepath.Join(commandsBackup, "deploy.md")
	if err := os.WriteFile(commandFile, []byte("# deploy"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	noCommandsPicker := func(_, _ []string, _, _ int, _ string) ([]string, error) {
		return []string{}, nil // confirm with nothing selected
	}
	if err := runProfileCreate(profilesRoot, "work", claudeDir, noCommandsPicker, &bytes.Buffer{}); err != nil {
		t.Fatalf("runProfileCreate: %v", err)
	}

	// Pre-install command symlink.
	link := filepath.Join(profilesRoot, "work", "commands", "deploy.md")
	if err := os.Symlink(commandFile, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	// Edit: deselect command (return nil = empty selection for commands step).
	deselect := func(_, _ []string, _, _ int, _ string) ([]string, error) {
		return []string{}, nil // confirm with empty selection for all steps
	}

	var buf bytes.Buffer
	if err := runProfileEdit(profilesRoot, "work", claudeDir, deselect, &buf); err != nil {
		t.Fatalf("runProfileEdit: %v", err)
	}

	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Error("deploy.md symlink should have been removed after deselection")
	}
}

func TestProfileEditAppliesSelection(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	// Add two skills to base
	for _, s := range []string{"skill-a", "skill-b"} {
		if err := os.MkdirAll(filepath.Join(profilesRoot, "base", "skills", s), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	// Create work profile (inherits both)
	if err := runProfileCreate(profilesRoot, "work", claudeDir, noEditorPicker, &bytes.Buffer{}); err != nil {
		t.Fatalf("runProfileCreate: %v", err)
	}

	// Edit: keep only skill-a
	var buf bytes.Buffer
	if err := runProfileEdit(profilesRoot, "work", claudeDir, stubEditor([]string{"skill-a"}), &buf); err != nil {
		t.Fatalf("runProfileEdit: %v", err)
	}

	// skill-a should still be symlinked
	if _, err := os.Lstat(filepath.Join(profilesRoot, "work", "skills", "skill-a")); err != nil {
		t.Error("skill-a should still be present")
	}
	// skill-b should be removed
	if _, err := os.Lstat(filepath.Join(profilesRoot, "work", "skills", "skill-b")); !os.IsNotExist(err) {
		t.Error("skill-b should have been removed")
	}
}

func TestProfileEditCancelDoesNotApply(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir)

	// Add a skill to base and inherit it in work
	if err := os.MkdirAll(filepath.Join(profilesRoot, "base", "skills", "skill-x"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := runProfileCreate(profilesRoot, "work", claudeDir, noEditorPicker, &bytes.Buffer{}); err != nil {
		t.Fatalf("runProfileCreate: %v", err)
	}

	// Cancel the edit
	var buf bytes.Buffer
	if err := runProfileEdit(profilesRoot, "work", claudeDir, cancelEditor, &buf); err != nil {
		t.Fatalf("runProfileEdit: %v", err)
	}

	// skill-x should still be there (cancel = no change)
	if _, err := os.Lstat(filepath.Join(profilesRoot, "work", "skills", "skill-x")); err != nil {
		t.Error("skill-x should still be present after cancel")
	}
}

// --- profile delete ---

func TestProfileDeleteRemovesProfile(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir) // active = base

	if err := profile.CreateChild(profilesRoot, "base", "work"); err != nil {
		t.Fatalf("CreateChild: %v", err)
	}

	var buf bytes.Buffer
	if err := runProfileDelete(profilesRoot, "work", &buf); err != nil {
		t.Fatalf("runProfileDelete: %v", err)
	}

	if _, err := os.Stat(filepath.Join(profilesRoot, "work")); !os.IsNotExist(err) {
		t.Error("expected work dir removed")
	}
}

func TestProfileDeleteRefusesActiveProfile(t *testing.T) {
	profilesRoot := t.TempDir()
	claudeDir := t.TempDir()
	initBase(t, profilesRoot, claudeDir) // active = base

	var buf bytes.Buffer
	err := runProfileDelete(profilesRoot, "base", &buf)
	if err == nil {
		t.Fatal("expected error deleting active profile")
	}
}
