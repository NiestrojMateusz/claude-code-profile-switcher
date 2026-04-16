package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/matis/ccp/internal/config"
	"github.com/matis/ccp/internal/symlink"
	"github.com/spf13/cobra"
)

var (
	defaultProfilesRoot = filepath.Join(os.Getenv("HOME"), ".claude-profiles")
	defaultClaudeDir    = filepath.Join(os.Getenv("HOME"), ".claude")
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show active profile and symlink state",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStatus(defaultProfilesRoot, defaultClaudeDir, cmd.OutOrStdout())
	},
}

func runStatus(profilesRoot, claudeDir string, w io.Writer) error {
	cfg, err := config.Load(profilesRoot)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	active := config.ActiveProfile(cfg)
	if active == "" {
		fmt.Fprintln(w, "no active profile")
	} else {
		fmt.Fprintf(w, "active profile: %s\n", active)
	}

	links, err := symlink.InspectAll(claudeDir)
	if err != nil {
		return fmt.Errorf("inspect symlinks: %w", err)
	}

	for _, l := range links {
		switch l.Kind {
		case symlink.KindSymlink:
			fmt.Fprintf(w, "  %s -> %s\n", filepath.Base(l.Path), l.Target)
		case symlink.KindReal:
			fmt.Fprintf(w, "  %s [real dir]\n", filepath.Base(l.Path))
		case symlink.KindAbsent:
			fmt.Fprintf(w, "  %s [absent]\n", filepath.Base(l.Path))
		}
	}

	return nil
}
