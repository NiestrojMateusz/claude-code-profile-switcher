package symlink

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Target names the three directories managed by ccp under ~/.claude.
type Target string

const (
	Skills   Target = "skills"
	Commands Target = "commands"
	Agents   Target = "agents"

	// AllTargets lists all managed targets in display order.
)

var AllTargets = []Target{Skills, Commands, Agents}

// Kind describes what exists at a managed path.
type Kind int

const (
	KindAbsent  Kind = iota // path does not exist
	KindReal                // path is a plain directory (not a symlink)
	KindSymlink             // path is a symlink
)

// LinkInfo holds the inspection result for one managed path.
type LinkInfo struct {
	Path   string
	Kind   Kind
	Target string // non-empty when Kind == KindSymlink
}

// Inspect reports the state of a single target under claudeDir.
// It uses os.Lstat so it sees the symlink itself, not its destination.
func Inspect(claudeDir string, t Target) (LinkInfo, error) {
	path := filepath.Join(claudeDir, string(t))
	info := LinkInfo{Path: path}

	fi, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		info.Kind = KindAbsent
		return info, nil
	}
	if err != nil {
		return LinkInfo{}, err
	}

	if fi.Mode()&os.ModeSymlink != 0 {
		dest, err := os.Readlink(path)
		if err != nil {
			return LinkInfo{}, err
		}
		info.Kind = KindSymlink
		info.Target = dest
		return info, nil
	}

	info.Kind = KindReal
	return info, nil
}

// Switch atomically replaces all three managed symlinks under claudeDir to
// point at the corresponding subdirectory inside profileRoot.
// It requires that profileRoot/skills, profileRoot/commands and
// profileRoot/agents all exist before the switch begins.
func Switch(claudeDir, profileRoot string) error {
	for _, t := range AllTargets {
		dest := filepath.Join(profileRoot, string(t))
		if _, err := os.Stat(dest); err != nil {
			return fmt.Errorf("profile subdir %q: %w", dest, err)
		}
	}
	for _, t := range AllTargets {
		link := filepath.Join(claudeDir, string(t))
		dest := filepath.Join(profileRoot, string(t))
		// Remove existing entry (symlink or absent — real dir left alone).
		if err := os.Remove(link); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove %q: %w", link, err)
		}
		if err := os.Symlink(dest, link); err != nil {
			return fmt.Errorf("symlink %q → %q: %w", link, dest, err)
		}
	}
	return nil
}

// InspectAll returns LinkInfo for all three managed targets.
func InspectAll(claudeDir string) ([]LinkInfo, error) {
	var result []LinkInfo
	for _, t := range AllTargets {
		info, err := Inspect(claudeDir, t)
		if err != nil {
			return nil, err
		}
		result = append(result, info)
	}
	return result, nil
}
