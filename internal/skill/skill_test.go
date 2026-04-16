package skill

import (
	"os"
	"path/filepath"
	"testing"
)

// helpers

func makeDir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll %s: %v", path, err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile %s: %v", path, err)
	}
}

func setupProfiles(t *testing.T) (profilesRoot string) {
	t.Helper()
	root := t.TempDir()
	for _, sub := range []string{"skills", "commands", "agents"} {
		makeDir(t, filepath.Join(root, "base", sub))
	}
	return root
}

// --- Discover ---

func TestDiscoverFindsSkillsInBackupDir(t *testing.T) {
	claudeDir := t.TempDir()

	// Simulate a backed-up skills dir with two skills
	backupSkills := filepath.Join(claudeDir, "skills_backup_1234567890")
	makeDir(t, filepath.Join(backupSkills, "skill-a"))
	makeDir(t, filepath.Join(backupSkills, "skill-b"))

	found := Discover(claudeDir, "")

	want := map[string]bool{
		filepath.Join(backupSkills, "skill-a"): true,
		filepath.Join(backupSkills, "skill-b"): true,
	}
	if len(found) != len(want) {
		t.Fatalf("want %d skills, got %d: %v", len(want), len(found), found)
	}
	for _, p := range found {
		if !want[p] {
			t.Errorf("unexpected skill path: %q", p)
		}
	}
}

func TestDiscoverReturnsNilWhenNoBackup(t *testing.T) {
	claudeDir := t.TempDir()
	found := Discover(claudeDir, "")
	if len(found) != 0 {
		t.Errorf("want empty, got %v", found)
	}
}

// --- DiscoverCommands ---

func TestDiscoverCommandsFindsMarkdownInCommandsBackup(t *testing.T) {
	claudeDir := t.TempDir()
	backupDir := filepath.Join(claudeDir, "commands_backup_1234567890")
	makeDir(t, backupDir)
	writeFile(t, filepath.Join(backupDir, "deploy.md"), "# deploy")
	writeFile(t, filepath.Join(backupDir, "lint.md"), "# lint")

	found := DiscoverCommands(claudeDir)

	want := map[string]bool{
		filepath.Join(backupDir, "deploy.md"): true,
		filepath.Join(backupDir, "lint.md"):   true,
	}
	if len(found) != len(want) {
		t.Fatalf("want %d commands, got %d: %v", len(want), len(found), found)
	}
	for _, p := range found {
		if !want[p] {
			t.Errorf("unexpected command path: %q", p)
		}
	}
}

func TestDiscoverCommandsReturnsEmptyWhenNoBackup(t *testing.T) {
	claudeDir := t.TempDir()
	found := DiscoverCommands(claudeDir)
	if len(found) != 0 {
		t.Errorf("want empty, got %v", found)
	}
}

func TestDiscoverCommandsIgnoresNonMarkdownFiles(t *testing.T) {
	claudeDir := t.TempDir()
	backupDir := filepath.Join(claudeDir, "commands_backup_1234567890")
	makeDir(t, backupDir)
	writeFile(t, filepath.Join(backupDir, "deploy.md"), "# deploy")
	writeFile(t, filepath.Join(backupDir, "README.txt"), "not markdown")

	found := DiscoverCommands(claudeDir)

	if len(found) != 1 || filepath.Base(found[0]) != "deploy.md" {
		t.Errorf("want only deploy.md, got %v", found)
	}
}

// --- DiscoverAgents ---

func TestDiscoverAgentsFindsMarkdownInAgentsBackup(t *testing.T) {
	claudeDir := t.TempDir()
	backupDir := filepath.Join(claudeDir, "agents_backup_1234567890")
	makeDir(t, backupDir)
	writeFile(t, filepath.Join(backupDir, "reviewer.md"), "# reviewer")

	found := DiscoverAgents(claudeDir)

	if len(found) != 1 || found[0] != filepath.Join(backupDir, "reviewer.md") {
		t.Errorf("want [reviewer.md path], got %v", found)
	}
}

func TestDiscoverAgentsReturnsEmptyWhenNoBackup(t *testing.T) {
	claudeDir := t.TempDir()
	found := DiscoverAgents(claudeDir)
	if len(found) != 0 {
		t.Errorf("want empty, got %v", found)
	}
}

// --- ApplyInheritance ---

func TestApplyInheritanceAddsSymlinkForSelected(t *testing.T) {
	profilesRoot := setupProfiles(t)
	makeDir(t, filepath.Join(profilesRoot, "work", "skills"))

	// base has a skill
	makeDir(t, filepath.Join(profilesRoot, "base", "skills", "cool-skill"))

	if err := ApplyInheritance(profilesRoot, "base", "work", []string{"cool-skill"}); err != nil {
		t.Fatalf("ApplyInheritance: %v", err)
	}

	link := filepath.Join(profilesRoot, "work", "skills", "cool-skill")
	fi, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("Lstat %s: %v", link, err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected symlink at %s", link)
	}
}

func TestApplyInheritanceRemovesSymlinkForDeselected(t *testing.T) {
	profilesRoot := setupProfiles(t)
	makeDir(t, filepath.Join(profilesRoot, "work", "skills"))

	baseSkill := filepath.Join(profilesRoot, "base", "skills", "old-skill")
	makeDir(t, baseSkill)

	// Pre-create the inherited symlink
	link := filepath.Join(profilesRoot, "work", "skills", "old-skill")
	rel, _ := filepath.Rel(filepath.Join(profilesRoot, "work", "skills"), baseSkill)
	if err := os.Symlink(rel, link); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	// Deselect: pass empty selected list
	if err := ApplyInheritance(profilesRoot, "base", "work", []string{}); err != nil {
		t.Fatalf("ApplyInheritance: %v", err)
	}

	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Error("expected symlink removed")
	}
}

func TestApplyInheritanceIsIdempotent(t *testing.T) {
	profilesRoot := setupProfiles(t)
	makeDir(t, filepath.Join(profilesRoot, "work", "skills"))
	makeDir(t, filepath.Join(profilesRoot, "base", "skills", "skill-x"))

	if err := ApplyInheritance(profilesRoot, "base", "work", []string{"skill-x"}); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	if err := ApplyInheritance(profilesRoot, "base", "work", []string{"skill-x"}); err != nil {
		t.Fatalf("second apply: %v", err)
	}

	link := filepath.Join(profilesRoot, "work", "skills", "skill-x")
	if _, err := os.Lstat(link); err != nil {
		t.Errorf("symlink missing after idempotent apply: %v", err)
	}
}

// --- AddLocal ---

func TestAddLocalCreatesSymlinkInProfileSkills(t *testing.T) {
	profilesRoot := setupProfiles(t)
	src := t.TempDir() // fake skill dir

	if err := AddLocal(profilesRoot, "base", src); err != nil {
		t.Fatalf("AddLocal: %v", err)
	}

	link := filepath.Join(profilesRoot, "base", "skills", filepath.Base(src))
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
	if target != src {
		t.Errorf("symlink target: want %q, got %q", src, target)
	}
}

func TestAddLocalCreatesSkillsDirWhenMissing(t *testing.T) {
	profilesRoot := t.TempDir()
	// Profile dir exists but skills/ subdir does not.
	makeDir(t, filepath.Join(profilesRoot, "work"))
	src := t.TempDir()

	if err := AddLocal(profilesRoot, "work", src); err != nil {
		t.Fatalf("AddLocal: %v", err)
	}

	link := filepath.Join(profilesRoot, "work", "skills", filepath.Base(src))
	fi, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("Lstat %s: %v", link, err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected symlink at %s", link)
	}
}

func TestAddLocalErrorsOnNonExistentSrc(t *testing.T) {
	profilesRoot := setupProfiles(t)

	err := AddLocal(profilesRoot, "base", "/nonexistent/skill")
	if err == nil {
		t.Fatal("expected error for non-existent src, got nil")
	}
}

// --- List ---

func TestListReturnsOwnedSkill(t *testing.T) {
	profilesRoot := setupProfiles(t)

	// Create a real dir in base/skills (owned)
	makeDir(t, filepath.Join(profilesRoot, "base", "skills", "owned-skill"))

	entries, err := List(profilesRoot, "base")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "owned-skill" {
		t.Errorf("name: want %q, got %q", "owned-skill", entries[0].Name)
	}
	if entries[0].Kind != KindOwned {
		t.Errorf("kind: want KindOwned, got %v", entries[0].Kind)
	}
}

func TestListReturnsInheritedSkill(t *testing.T) {
	profilesRoot := setupProfiles(t)
	// Set up work profile
	makeDir(t, filepath.Join(profilesRoot, "work", "skills"))

	// Add a skill to base
	baseSkill := filepath.Join(profilesRoot, "base", "skills", "shared-skill")
	makeDir(t, baseSkill)

	// Symlink it into work (as CreateChild would do)
	link := filepath.Join(profilesRoot, "work", "skills", "shared-skill")
	rel, _ := filepath.Rel(filepath.Join(profilesRoot, "work", "skills"), baseSkill)
	if err := os.Symlink(rel, link); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	entries, err := List(profilesRoot, "work")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	if entries[0].Kind != KindInherited {
		t.Errorf("kind: want KindInherited, got %v", entries[0].Kind)
	}
}

func TestListEmptyProfileReturnsNil(t *testing.T) {
	profilesRoot := setupProfiles(t)

	entries, err := List(profilesRoot, "base")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("want 0 entries, got %d", len(entries))
	}
}
