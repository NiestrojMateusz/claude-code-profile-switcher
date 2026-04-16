package cmd

import (
	"path/filepath"

	"github.com/matis/ccp/internal/config"
	"github.com/matis/ccp/internal/monitor"
	"github.com/matis/ccp/internal/skill"
	tuimonitor "github.com/matis/ccp/tui/monitor"
)

// MonitorPicker receives (newSkills, profiles) and returns the user's choices.
// Tests stub it; production passes tui/monitor.Run.
type MonitorPicker func(newSkills, profiles []string) ([]tuimonitor.Choice, error)

// runMonitorOnCommand calls runMonitorCheck unless cmdName is "init".
// Skipping init prevents two Bubbletea TUI programs from running back-to-back
// in the same terminal session, which corrupts terminal state and leaves the
// init skill-selector TUI unable to receive keyboard input.
// allowMonitorCommands lists the only commands where the monitor TUI may run.
// Using an allowlist ensures new commands never accidentally surface the monitor.
// "switch" is the natural moment to discover new skills — the user is actively
// choosing a profile context.
var allowMonitorCommands = map[string]bool{
	"switch": true,
}

func runMonitorOnCommand(cmdName, profilesRoot, agentsDir string, picker MonitorPicker) error {
	if !allowMonitorCommands[cmdName] {
		return nil
	}
	return runMonitorCheck(profilesRoot, agentsDir, picker)
}

// runMonitorCheck scans agentsDir for new skills not in KnownSkills.
// If any are found, calls picker so the user can decide what to do with each.
// Silent when agentsDir is missing or no new skills are detected.
func runMonitorCheck(profilesRoot, agentsDir string, picker MonitorPicker) error {
	cfg, err := config.Load(profilesRoot)
	if err != nil {
		return err
	}

	if len(cfg.Profiles) == 0 {
		return nil
	}

	// Exclude skills already present in the base profile — they need no action.
	excluded := cfg.KnownSkills
	if baseEntries, err := skill.List(profilesRoot, "base"); err == nil {
		for _, e := range baseEntries {
			excluded = append(excluded, e.Name)
		}
	}

	newSkills := monitor.Scan(agentsDir, excluded)
	if len(newSkills) == 0 {
		return nil
	}

	choices, err := picker(newSkills, cfg.Profiles)
	if err != nil {
		return err
	}
	if choices == nil {
		return nil // user cancelled
	}

	return applyChoices(profilesRoot, agentsDir, choices)
}

// applyChoices executes the user's decisions from the monitor TUI.
// ActionAdd: symlinks the skill from agentsDir into the named profile's skills/.
// ActionNever: records the skill in KnownSkills so it is not surfaced again.
// ActionSkip: no-op.
func applyChoices(profilesRoot, agentsDir string, choices []tuimonitor.Choice) error {
	cfg, err := config.Load(profilesRoot)
	if err != nil {
		return err
	}

	var toMarkKnown []string
	for _, c := range choices {
		switch c.Action {
		case tuimonitor.ActionAdd:
			src := filepath.Join(agentsDir, c.Skill)
			if err := skill.AddLocal(profilesRoot, c.Profile, src); err != nil {
				return err
			}
			toMarkKnown = append(toMarkKnown, c.Skill)
		case tuimonitor.ActionNever:
			toMarkKnown = append(toMarkKnown, c.Skill)
		// ActionSkip: do nothing
		}
	}

	if len(toMarkKnown) == 0 {
		return nil
	}
	cfg.KnownSkills = monitor.MarkKnown(cfg.KnownSkills, toMarkKnown)
	return config.Save(profilesRoot, cfg)
}
