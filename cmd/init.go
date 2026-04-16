package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/matis/ccp/internal/backup"
	"github.com/matis/ccp/internal/config"
	"github.com/matis/ccp/internal/profile"
	"github.com/matis/ccp/internal/skill"
	"github.com/matis/ccp/internal/symlink"
	"github.com/matis/ccp/tui/selector"
	"github.com/spf13/cobra"
)

// SkillPicker receives available item names and step info, returns which ones to include.
// Tests pass a stub; production passes the Bubbletea TUI.
type SkillPicker func(available []string, step, totalSteps int, title string) ([]string, error)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialise ccp and create the base profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInit(defaultProfilesRoot, defaultClaudeDir, tuiSkillPicker, cmd.OutOrStdout())
	},
}

// tuiSkillPicker is the real picker — opens the Bubbletea TUI with step header.
func tuiSkillPicker(available []string, step, totalSteps int, title string) ([]string, error) {
	m := selector.NewWithStep(available, step, totalSteps, title)
	return selector.RunModel(m)
}

func runInit(profilesRoot, claudeDir string, picker SkillPicker, w io.Writer) error {
	// Guard: already initialised when base profile exists in config.
	cfg, err := config.Load(profilesRoot)
	if err != nil {
		return err
	}
	if config.ActiveProfile(cfg) != "" {
		fmt.Fprintln(w, "already initialised")
		return nil
	}

	// 1. Ensure claudeDir exists (user may pass a custom --claude-dir that doesn't exist yet).
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		return fmt.Errorf("create claude dir: %w", err)
	}

	// 2. Back up existing real directories.
	for _, t := range symlink.AllTargets {
		dir := filepath.Join(claudeDir, string(t))
		if _, err := backup.Backup(dir); err != nil {
			return fmt.Errorf("backup %s: %w", t, err)
		}
	}

	// 3. Run 3-step picker: skills → commands → agents.
	selectedSkills, err := picker(discoverSkills(claudeDir), 1, 3, "Skills")
	if err != nil {
		return err
	}
	selectedCommands, err := picker(discoverCommands(claudeDir), 2, 3, "Commands")
	if err != nil {
		return err
	}
	selectedAgents, err := picker(discoverAgents(claudeDir), 3, 3, "Agents")
	if err != nil {
		return err
	}

	// 4. Create base profile dirs and write config.
	if err := profile.Create(profilesRoot, "base"); err != nil {
		return fmt.Errorf("create base profile: %w", err)
	}

	// 5. Install selected items into base/{skills,commands,agents}/.
	if err := installItems(profilesRoot, "base", "skills", selectedSkills); err != nil {
		return err
	}
	if err := installItems(profilesRoot, "base", "commands", selectedCommands); err != nil {
		return err
	}
	if err := installItems(profilesRoot, "base", "agents", selectedAgents); err != nil {
		return err
	}

	// 5. Replace ~/.claude/skills|commands|agents with symlinks to base/.
	for _, t := range symlink.AllTargets {
		link := filepath.Join(claudeDir, string(t))
		target := filepath.Join(profilesRoot, "base", string(t))
		if err := os.Symlink(target, link); err != nil {
			return fmt.Errorf("symlink %s: %w", t, err)
		}
	}

	fmt.Fprintln(w, "initialised: base profile active")
	return nil
}

// discoverSkills collects skill paths from backed-up claude dir and ~/.agents/skills/.
func discoverSkills(claudeDir string) []string {
	home, _ := os.UserHomeDir()
	agentsSkillsDir := filepath.Join(home, ".agents", "skills")
	return skill.Discover(claudeDir, agentsSkillsDir)
}

func discoverCommands(claudeDir string) []string {
	return skill.DiscoverCommands(claudeDir)
}

func discoverAgents(claudeDir string) []string {
	return skill.DiscoverAgents(claudeDir)
}

// installItems symlinks each src path into profilesRoot/<profile>/<category>/.
func installItems(profilesRoot, profileName, category string, srcs []string) error {
	dir := filepath.Join(profilesRoot, profileName, category)
	for _, src := range srcs {
		dest := filepath.Join(dir, filepath.Base(src))
		if err := os.Symlink(src, dest); err != nil && !os.IsExist(err) {
			return fmt.Errorf("link %s/%s: %w", category, filepath.Base(src), err)
		}
	}
	return nil
}
