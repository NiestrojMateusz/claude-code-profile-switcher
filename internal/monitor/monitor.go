package monitor

import (
	"os"
	"slices"
)

// Scan reads agentsDir and returns the names of skill directories that are not
// present in knownSkills. Files (non-directories) are ignored. Returns nil when
// agentsDir does not exist or is empty.
func Scan(agentsDir string, knownSkills []string) []string {
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return nil
	}
	var newSkills []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if !slices.Contains(knownSkills, e.Name()) {
			newSkills = append(newSkills, e.Name())
		}
	}
	return newSkills
}

// MarkKnown appends each name in toAdd to knownSkills, skipping duplicates.
// Returns the updated slice. Does not mutate the input.
func MarkKnown(knownSkills, toAdd []string) []string {
	result := append([]string(nil), knownSkills...)
	for _, name := range toAdd {
		if !slices.Contains(result, name) {
			result = append(result, name)
		}
	}
	return result
}
