package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/matis/ccp/internal/symlink"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(cleanCmd)
}

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove profiles directory and claude dir symlinks",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runClean(defaultProfilesRoot, defaultClaudeDir, cmd.OutOrStdout())
	},
}

func runClean(profilesRoot, claudeDir string, w io.Writer) error {
	// Remove symlinks ccp created in claudeDir.
	for _, t := range symlink.AllTargets {
		link := filepath.Join(claudeDir, string(t))
		fi, err := os.Lstat(link)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("lstat %s: %w", link, err)
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(link); err != nil {
				return fmt.Errorf("remove symlink %s: %w", link, err)
			}
		}
	}

	// Remove profiles directory.
	if err := os.RemoveAll(profilesRoot); err != nil {
		return fmt.Errorf("remove profiles root: %w", err)
	}

	fmt.Fprintf(w, "cleaned: removed %s and symlinks from %s\n", profilesRoot, claudeDir)
	return nil
}
