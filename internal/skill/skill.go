package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Discover returns skill paths from backed-up claude skills dirs
// (matching <claudeDir>/skills_backup_*/<entry>) and from agentsSkillsDir.
// Pass an empty agentsSkillsDir to skip agent skill discovery.
func Discover(claudeDir, agentsSkillsDir string) []string {
	var paths []string

	// Scan backup dirs: skills_backup_<timestamp>
	pattern := filepath.Join(claudeDir, "skills_backup_*")
	backupDirs, _ := filepath.Glob(pattern)
	for _, bdir := range backupDirs {
		entries, err := os.ReadDir(bdir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				paths = append(paths, filepath.Join(bdir, e.Name()))
			}
		}
	}

	// Scan agentsSkillsDir
	if agentsSkillsDir != "" {
		entries, err := os.ReadDir(agentsSkillsDir)
		if err == nil {
			for _, e := range entries {
				if e.IsDir() {
					paths = append(paths, filepath.Join(agentsSkillsDir, e.Name()))
				}
			}
		}
	}

	return paths
}

// DiscoverCommands returns .md file paths found in commands_backup_* dirs under claudeDir.
func DiscoverCommands(claudeDir string) []string {
	return discoverMarkdownFiles(claudeDir, "commands_backup_*")
}

// DiscoverAgents returns .md file paths found in agents_backup_* dirs under claudeDir.
func DiscoverAgents(claudeDir string) []string {
	return discoverMarkdownFiles(claudeDir, "agents_backup_*")
}

// discoverMarkdownFiles returns .md files found in dirs matching the given glob pattern under claudeDir.
func discoverMarkdownFiles(claudeDir, pattern string) []string {
	var paths []string
	backupDirs, _ := filepath.Glob(filepath.Join(claudeDir, pattern))
	for _, bdir := range backupDirs {
		entries, err := os.ReadDir(bdir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				paths = append(paths, filepath.Join(bdir, e.Name()))
			}
		}
	}
	return paths
}

// Kind describes how a skill relates to its profile.
type Kind int

const (
	KindOwned     Kind = iota // real dir or external symlink owned by this profile
	KindInherited             // symlink pointing into base/skills/
)

func (k Kind) String() string {
	if k == KindInherited {
		return "inherited"
	}
	return "owned"
}

// Entry is a skill returned by List.
type Entry struct {
	Name string
	Kind Kind
}

// ApplyInheritance reconciles profilesRoot/<profileName>/skills/ so that
// exactly the named base skills are present as inherited symlinks.
// Skills in selected that are missing get a new symlink; inherited symlinks
// for skills not in selected are removed. Owned (non-base) entries are untouched.
func ApplyInheritance(profilesRoot, baseName, profileName string, selected []string) error {
	baseSkills := filepath.Join(profilesRoot, baseName, "skills")
	profileSkills := filepath.Join(profilesRoot, profileName, "skills")

	selectedSet := make(map[string]bool, len(selected))
	for _, s := range selected {
		selectedSet[s] = true
	}

	// Add missing symlinks for selected skills.
	for name := range selectedSet {
		link := filepath.Join(profileSkills, name)
		if _, err := os.Lstat(link); err == nil {
			continue // already exists
		}
		target := filepath.Join(baseSkills, name)
		rel, err := filepath.Rel(profileSkills, target)
		if err != nil {
			return fmt.Errorf("rel path for %s: %w", name, err)
		}
		if err := os.Symlink(rel, link); err != nil {
			return fmt.Errorf("symlink %s: %w", name, err)
		}
	}

	// Remove inherited symlinks for deselected base skills.
	entries, err := os.ReadDir(profileSkills)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, de := range entries {
		if selectedSet[de.Name()] {
			continue
		}
		link := filepath.Join(profileSkills, de.Name())
		fi, err := os.Lstat(link)
		if err != nil {
			continue
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			continue // owned real dir — leave alone
		}
		target, err := os.Readlink(link)
		if err != nil {
			continue
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(profileSkills, target)
		}
		// Only remove symlinks that point into base/skills/
		if isUnderDir(target, baseSkills) {
			if err := os.Remove(link); err != nil {
				return fmt.Errorf("remove %s: %w", de.Name(), err)
			}
		}
	}

	return nil
}

// isUnderDir reports whether path is directly inside dir.
func isUnderDir(path, dir string) bool {
	return strings.HasPrefix(filepath.Clean(path), filepath.Clean(dir)+string(os.PathSeparator))
}

// AddLocal symlinks srcPath into profilesRoot/<profileName>/skills/.
// srcPath must exist.
func AddLocal(profilesRoot, profileName, srcPath string) error {
	if _, err := os.Stat(srcPath); err != nil {
		return fmt.Errorf("skill source %q: %w", srcPath, err)
	}
	skillsDir := filepath.Join(profilesRoot, profileName, "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		return fmt.Errorf("create skills dir: %w", err)
	}
	link := filepath.Join(skillsDir, filepath.Base(srcPath))
	if err := os.Symlink(srcPath, link); err != nil {
		return fmt.Errorf("symlink skill: %w", err)
	}
	return nil
}

// List returns all skills in profilesRoot/<profileName>/skills/, labelled as
// KindInherited if the symlink target is inside base/skills/, else KindOwned.
func List(profilesRoot, profileName string) ([]Entry, error) {
	skillsDir := filepath.Join(profilesRoot, profileName, "skills")
	baseSkills := filepath.Join(profilesRoot, "base", "skills")

	dirEntries, err := os.ReadDir(skillsDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	entries := make([]Entry, 0, len(dirEntries))
	for _, de := range dirEntries {
		e := Entry{Name: de.Name(), Kind: KindOwned}

		fi, err := os.Lstat(filepath.Join(skillsDir, de.Name()))
		if err != nil {
			return nil, err
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(filepath.Join(skillsDir, de.Name()))
			if err != nil {
				return nil, err
			}
			// Resolve relative symlinks against their directory.
			if !filepath.IsAbs(target) {
				target = filepath.Join(skillsDir, target)
			}
			target = filepath.Clean(target)
			if isUnderDir(target, baseSkills) {
				e.Kind = KindInherited
			}
		}
		entries = append(entries, e)
	}
	return entries, nil
}
