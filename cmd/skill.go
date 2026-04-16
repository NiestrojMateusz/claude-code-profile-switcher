package cmd

import (
	"fmt"
	"io"

	"github.com/matis/ccp/internal/config"
	"github.com/matis/ccp/internal/skill"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(skillCmd)
	skillCmd.AddCommand(skillAddCmd)
	skillCmd.AddCommand(skillListCmd)

	skillAddCmd.Flags().StringP("profile", "p", "", "target profile (defaults to active)")
	skillListCmd.Flags().StringP("profile", "p", "", "profile to list (defaults to active)")
}

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Manage skills",
}

var skillAddCmd = &cobra.Command{
	Use:   "add <path>",
	Short: "Add a skill to a profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName, err := resolveProfile(cmd, defaultProfilesRoot)
		if err != nil {
			return err
		}
		return runSkillAdd(defaultProfilesRoot, profileName, args[0], cmd.OutOrStdout())
	},
}

var skillListCmd = &cobra.Command{
	Use:   "list",
	Short: "List skills in a profile",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		profileName, err := resolveProfile(cmd, defaultProfilesRoot)
		if err != nil {
			return err
		}
		return runSkillList(defaultProfilesRoot, profileName, cmd.OutOrStdout())
	},
}

// resolveProfile returns the --profile flag value, or the active profile from config.
func resolveProfile(cmd *cobra.Command, profilesRoot string) (string, error) {
	if name, _ := cmd.Flags().GetString("profile"); name != "" {
		return name, nil
	}
	cfg, err := config.Load(profilesRoot)
	if err != nil {
		return "", err
	}
	if cfg.ActiveProfile == "" {
		return "", fmt.Errorf("no active profile; run 'ccp init' first")
	}
	return cfg.ActiveProfile, nil
}

func runSkillAdd(profilesRoot, profileName, srcPath string, w io.Writer) error {
	if err := skill.AddLocal(profilesRoot, profileName, srcPath); err != nil {
		return err
	}
	fmt.Fprintf(w, "added skill %q to profile %q\n", srcPath, profileName)
	return nil
}

func runSkillList(profilesRoot, profileName string, w io.Writer) error {
	entries, err := skill.List(profilesRoot, profileName)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Fprintln(w, "no skills in profile")
		return nil
	}
	for _, e := range entries {
		fmt.Fprintf(w, "  %s (%s)\n", e.Name, e.Kind)
	}
	return nil
}
