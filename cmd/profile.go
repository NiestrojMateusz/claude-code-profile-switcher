package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/matis/ccp/internal/config"
	"github.com/matis/ccp/internal/profile"
	"github.com/matis/ccp/internal/skill"
	"github.com/matis/ccp/tui/selector"
	"github.com/spf13/cobra"
)

// tuiEditorPicker is the real picker — opens the Bubbletea TUI with pre-selection and step header.
func tuiEditorPicker(items, preSelected []string, step, totalSteps int, title string) ([]string, error) {
	m := selector.NewWithSelected(items, preSelected)
	m.Step = step
	m.TotalSteps = totalSteps
	m.Title = title
	return selector.RunModel(m)
}

// SkillEditorPicker receives available items, pre-selected items, and step info, returns the new selection.
// Tests stub it; production passes a step-aware TUI runner.
type SkillEditorPicker func(items, preSelected []string, step, totalSteps int, title string) ([]string, error)

func init() {
	rootCmd.AddCommand(profileCmd)
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileCreateCmd)
	profileCmd.AddCommand(profileDeleteCmd)
	profileCmd.AddCommand(profileEditCmd)
}

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage profiles",
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runProfileList(defaultProfilesRoot, cmd.OutOrStdout())
	},
}

var profileCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new profile inheriting base skills",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runProfileCreate(defaultProfilesRoot, args[0], defaultClaudeDir, tuiEditorPicker, cmd.OutOrStdout())
	},
}

var profileDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runProfileDelete(defaultProfilesRoot, args[0], cmd.OutOrStdout())
	},
}

func runProfileList(profilesRoot string, w io.Writer) error {
	cfg, err := config.Load(profilesRoot)
	if err != nil {
		return err
	}
	if len(cfg.Profiles) == 0 {
		fmt.Fprintln(w, "no profiles found; run 'ccp init' first")
		return nil
	}
	for _, p := range cfg.Profiles {
		marker := " "
		if p == cfg.ActiveProfile {
			marker = "*"
		}
		fmt.Fprintf(w, "%s %s\n", marker, p)
	}
	return nil
}

func runProfileCreate(profilesRoot, name, claudeDir string, picker SkillEditorPicker, w io.Writer) error {
	// Collect base skill names (pre-selected by default).
	baseEntries, err := skill.List(profilesRoot, "base")
	if err != nil {
		return fmt.Errorf("list base skills: %w", err)
	}
	baseNames := make(map[string]bool, len(baseEntries))
	for _, e := range baseEntries {
		baseNames[e.Name] = true
	}

	// Discover external skills (agents dir + backups) not already in base.
	// Build a name→srcPath map for external skills.
	externalPaths := discoverSkills(claudeDir)
	extByName := make(map[string]string)
	for _, p := range externalPaths {
		n := filepath.Base(p)
		if !baseNames[n] {
			extByName[n] = p
		}
	}

	// Combined list: base names first, then external names.
	allBase := make([]string, 0, len(baseNames))
	for n := range baseNames {
		allBase = append(allBase, n)
	}
	allItems := append(allBase, sortedKeys(extByName)...)

	// Step 1: Skills — base skills pre-selected; external skills not.
	selectedSkills, err := picker(allItems, allBase, 1, 3, "Skills")
	if err != nil {
		return err
	}
	if selectedSkills == nil {
		fmt.Fprintln(w, "create cancelled")
		return nil
	}

	// Step 2: Commands — discover available, no pre-selection for new profile.
	selectedCommands, err := picker(discoverCommands(claudeDir), nil, 2, 3, "Commands")
	if err != nil {
		return err
	}

	// Step 3: Agents — discover available, no pre-selection for new profile.
	selectedAgents, err := picker(discoverAgents(claudeDir), nil, 3, 3, "Agents")
	if err != nil {
		return err
	}

	if err := profile.CreateChild(profilesRoot, "base", name); err != nil {
		return err
	}

	// Apply skills: base (inherit) vs external (AddLocal).
	var selectedBase []string
	for _, s := range selectedSkills {
		if baseNames[s] {
			selectedBase = append(selectedBase, s)
		}
	}
	if err := skill.ApplyInheritance(profilesRoot, "base", name, selectedBase); err != nil {
		return fmt.Errorf("apply inheritance: %w", err)
	}
	for _, s := range selectedSkills {
		if src, ok := extByName[s]; ok {
			if err := skill.AddLocal(profilesRoot, name, src); err != nil {
				return fmt.Errorf("add local skill %s: %w", s, err)
			}
		}
	}

	// Install selected commands and agents.
	if err := installItems(profilesRoot, name, "commands", selectedCommands); err != nil {
		return err
	}
	if err := installItems(profilesRoot, name, "agents", selectedAgents); err != nil {
		return err
	}

	fmt.Fprintf(w, "created profile: %s\n", name)
	return nil
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func runProfileDelete(profilesRoot, name string, w io.Writer) error {
	if err := profile.Delete(profilesRoot, name); err != nil {
		return err
	}
	fmt.Fprintf(w, "deleted profile: %s\n", name)
	return nil
}

var profileEditCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Edit inherited skills for a profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runProfileEdit(defaultProfilesRoot, args[0], defaultClaudeDir, tuiEditorPicker, cmd.OutOrStdout())
	},
}

func runProfileEdit(profilesRoot, profileName, claudeDir string, picker SkillEditorPicker, w io.Writer) error {
	// --- Skills (step 1 of 3) ---
	baseEntries, err := skill.List(profilesRoot, "base")
	if err != nil {
		return fmt.Errorf("list base skills: %w", err)
	}
	allBase := make([]string, len(baseEntries))
	for i, e := range baseEntries {
		allBase[i] = e.Name
	}
	profileEntries, err := skill.List(profilesRoot, profileName)
	if err != nil {
		return fmt.Errorf("list profile skills: %w", err)
	}
	var currentSkills []string
	for _, e := range profileEntries {
		if e.Kind == skill.KindInherited {
			currentSkills = append(currentSkills, e.Name)
		}
	}
	selectedSkills, err := picker(allBase, currentSkills, 1, 3, "Skills")
	if err != nil {
		return err
	}
	if selectedSkills == nil {
		fmt.Fprintln(w, "edit cancelled")
		return nil
	}

	// --- Commands (step 2 of 3) ---
	commandPaths := discoverCommands(claudeDir)
	commandSrcMap := pathsByBase(commandPaths)
	currentCommands := listProfileCategory(profilesRoot, profileName, "commands")
	selectedCommands, err := picker(sortedKeys(commandSrcMap), currentCommands, 2, 3, "Commands")
	if err != nil {
		return err
	}

	// --- Agents (step 3 of 3) ---
	agentPaths := discoverAgents(claudeDir)
	agentSrcMap := pathsByBase(agentPaths)
	currentAgents := listProfileCategory(profilesRoot, profileName, "agents")
	selectedAgents, err := picker(sortedKeys(agentSrcMap), currentAgents, 3, 3, "Agents")
	if err != nil {
		return err
	}

	// Apply all changes.
	if err := skill.ApplyInheritance(profilesRoot, "base", profileName, selectedSkills); err != nil {
		return fmt.Errorf("apply inheritance: %w", err)
	}
	if err := applyCategory(profilesRoot, profileName, "commands", selectedCommands, commandSrcMap); err != nil {
		return err
	}
	if err := applyCategory(profilesRoot, profileName, "agents", selectedAgents, agentSrcMap); err != nil {
		return err
	}

	fmt.Fprintf(w, "updated profile: %s\n", profileName)
	return nil
}

// listProfileCategory returns the base names of files in profilesRoot/<profile>/<category>/.
func listProfileCategory(profilesRoot, profileName, category string) []string {
	dir := filepath.Join(profilesRoot, profileName, category)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// pathsByBase returns a map of base filename → full path for a list of paths.
func pathsByBase(paths []string) map[string]string {
	m := make(map[string]string, len(paths))
	for _, p := range paths {
		m[filepath.Base(p)] = p
	}
	return m
}

// applyCategory reconciles the category dir so exactly selectedNames are symlinked.
// Newly selected items are linked via srcMap; deselected items' symlinks are removed.
func applyCategory(profilesRoot, profileName, category string, selectedNames []string, srcMap map[string]string) error {
	dir := filepath.Join(profilesRoot, profileName, category)
	selectedSet := make(map[string]bool, len(selectedNames))
	for _, n := range selectedNames {
		selectedSet[n] = true
	}

	// Add missing symlinks for selected items.
	for name := range selectedSet {
		link := filepath.Join(dir, name)
		if _, err := os.Lstat(link); err == nil {
			continue // already exists
		}
		src, ok := srcMap[name]
		if !ok {
			continue
		}
		if err := os.Symlink(src, link); err != nil && !os.IsExist(err) {
			return fmt.Errorf("symlink %s/%s: %w", category, name, err)
		}
	}

	// Remove symlinks for deselected items.
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, de := range entries {
		if selectedSet[de.Name()] {
			continue
		}
		link := filepath.Join(dir, de.Name())
		fi, err := os.Lstat(link)
		if err != nil {
			continue
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(link); err != nil {
				return fmt.Errorf("remove %s/%s: %w", category, de.Name(), err)
			}
		}
	}
	return nil
}
