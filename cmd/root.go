package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// flagYes skips interactive confirmation prompts. Required when running
	// destructive non-interactive commands in scripts/CI.
	flagYes bool
	// flagNoTTY disables every interactive fallback. Any missing input
	// becomes a hard error instead of opening a huh form.
	flagNoTTY bool

	rootCmd = &cobra.Command{
		Use:   "qail",
		Short: "A workspace manager",
		Long:  "Manage your workspace in style with qail",
	}
)

// requireTTY returns nil when interactive prompts are allowed, or a
// descriptive error when --no-tty is set. `what` names the missing input
// so the error tells the caller which flag to pass next time.
func requireTTY(what string) error {
	if flagNoTTY {
		return fmt.Errorf("--no-tty set but %s not provided", what)
	}
	return nil
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&flagYes, "yes", "y", false, "skip confirmation prompts")
	rootCmd.PersistentFlags().BoolVar(&flagNoTTY, "no-tty", false, "fail instead of falling back to interactive prompts")

	rootCmd.AddCommand(scriptsCmd)
	rootCmd.AddCommand(tmuxCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(wsCmd)
	rootCmd.AddCommand(repoCmd)
	rootCmd.AddCommand(initCmd)
}
