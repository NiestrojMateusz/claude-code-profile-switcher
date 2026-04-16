package cmd

import (
	"fmt"
	"io"
	"path/filepath"
	"slices"

	"github.com/matis/ccp/internal/config"
	"github.com/matis/ccp/internal/process"
	"github.com/matis/ccp/internal/symlink"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(switchCmd)
}

var switchCmd = &cobra.Command{
	Use:   "switch <profile>",
	Short: "Switch the active profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSwitch(defaultProfilesRoot, defaultClaudeDir, args[0], process.Scan, terminalConfirm, cmd.OutOrStdout())
	},
}

// terminalConfirm asks the user via stdin whether to proceed despite running processes.
func terminalConfirm(pids []int) (bool, error) {
	fmt.Printf("claude is running (PIDs: %v). Switch anyway? [y/N] ", pids)
	var resp string
	_, err := fmt.Scanln(&resp)
	if err != nil {
		return false, nil
	}
	return resp == "y" || resp == "Y", nil
}

func runSwitch(
	profilesRoot, claudeDir, name string,
	scanProcesses func() ([]int, error),
	confirm func([]int) (bool, error),
	w io.Writer,
) error {
	cfg, err := config.Load(profilesRoot)
	if err != nil {
		return err
	}

	if !slices.Contains(cfg.Profiles, name) {
		return fmt.Errorf("profile %q not found", name)
	}

	// Warn if claude processes are running.
	pids, err := scanProcesses()
	if err != nil {
		return fmt.Errorf("scan processes: %w", err)
	}
	if len(pids) > 0 {
		fmt.Fprintf(w, "warning: claude is running (PIDs: %v)\n", pids)
		ok, err := confirm(pids)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(w, "switch aborted")
			return nil
		}
	}

	// Update symlinks.
	profileRoot := filepath.Join(profilesRoot, name)
	if err := symlink.Switch(claudeDir, profileRoot); err != nil {
		return fmt.Errorf("switch symlinks: %w", err)
	}

	// Persist new active profile.
	updated := config.SetActive(cfg, name)
	if err := config.Save(profilesRoot, updated); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(w, "switched to profile: %s\n", name)
	return nil
}
