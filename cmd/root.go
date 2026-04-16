package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	tuimonitor "github.com/matis/ccp/tui/monitor"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ccp",
	Short: "Claude Code Profile manager",
	PersistentPreRun: func(cmd *cobra.Command, _ []string) {
		agentsDir := filepath.Join(os.Getenv("HOME"), ".agents", "skills")
		if err := runMonitorOnCommand(cmd.Name(), defaultProfilesRoot, agentsDir, tuimonitor.Run); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "monitor: %v\n", err)
		}
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&defaultProfilesRoot, "profiles-root", defaultProfilesRoot, "override profiles directory (default ~/.claude-profiles)")
	rootCmd.PersistentFlags().StringVar(&defaultClaudeDir, "claude-dir", defaultClaudeDir, "override claude directory (default ~/.claude)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

