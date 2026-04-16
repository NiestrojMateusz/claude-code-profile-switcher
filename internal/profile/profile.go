package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/matis/ccp/internal/config"
)

var subdirs = []string{"skills", "commands", "agents"}

// Create makes the profile directory structure under profilesRoot/<name>/ and
// records the profile as active in config.json. It does NOT create symlinks —
// that is the responsibility of the caller (runInit or ccp switch).
func Create(profilesRoot, name string) error {
	for _, sub := range subdirs {
		dir := filepath.Join(profilesRoot, name, sub)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	cfg, err := config.Load(profilesRoot)
	if err != nil {
		return err
	}

	cfg = config.SetActive(cfg, name)
	cfg.Profiles = appendIfMissing(cfg.Profiles, name)

	return config.Save(profilesRoot, cfg)
}

// CreateChild creates a new profile directory structure under profilesRoot/<childName>/
// and symlinks every skill entry in base/skills/ into <childName>/skills/.
// It records the new profile in config.json but does NOT change the active profile.
func CreateChild(profilesRoot, baseName, childName string) error {
	for _, sub := range subdirs {
		if err := os.MkdirAll(filepath.Join(profilesRoot, childName, sub), 0o755); err != nil {
			return err
		}
	}

	// Inherit base skills as relative symlinks.
	baseSkills := filepath.Join(profilesRoot, baseName, "skills")
	childSkills := filepath.Join(profilesRoot, childName, "skills")
	entries, err := os.ReadDir(baseSkills)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read base skills: %w", err)
	}
	for _, e := range entries {
		target := filepath.Join(baseSkills, e.Name())
		link := filepath.Join(childSkills, e.Name())
		// Relative path from link's dir to target.
		rel, err := filepath.Rel(childSkills, target)
		if err != nil {
			return fmt.Errorf("rel path: %w", err)
		}
		if err := os.Symlink(rel, link); err != nil {
			return fmt.Errorf("symlink %s: %w", e.Name(), err)
		}
	}

	cfg, err := config.Load(profilesRoot)
	if err != nil {
		return err
	}
	cfg.Profiles = appendIfMissing(cfg.Profiles, childName)
	return config.Save(profilesRoot, cfg)
}

// Delete removes the profile directory and its entry from config.json.
// Fails if name is the currently active profile, or if name is "base" and
// other profiles still exist.
func Delete(profilesRoot, name string) error {
	cfg, err := config.Load(profilesRoot)
	if err != nil {
		return err
	}

	if cfg.ActiveProfile == name {
		return fmt.Errorf("cannot delete active profile %q; switch to another profile first", name)
	}

	if name == "base" {
		others := make([]string, 0, len(cfg.Profiles))
		for _, p := range cfg.Profiles {
			if p != "base" {
				others = append(others, p)
			}
		}
		if len(others) > 0 {
			return fmt.Errorf("cannot delete base profile while other profiles exist: %v", others)
		}
	}

	if err := os.RemoveAll(filepath.Join(profilesRoot, name)); err != nil {
		return err
	}

	cfg.Profiles = removeFromSlice(cfg.Profiles, name)
	return config.Save(profilesRoot, cfg)
}

func appendIfMissing(slice []string, s string) []string {
	if slices.Contains(slice, s) {
		return slice
	}
	return append(slice, s)
}

func removeFromSlice(slice []string, s string) []string {
	result := slice[:0:0]
	for _, v := range slice {
		if v != s {
			result = append(result, v)
		}
	}
	return result
}
