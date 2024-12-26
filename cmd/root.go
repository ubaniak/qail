package cmd

import (
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "qail",
		Short: "A workspace manager",
		Long:  "Manage your repos in style with qail",
	}
)

type cobraReturnType = func(*cobra.Command, []string)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(scriptsCmd)
	rootCmd.AddCommand(tmuxCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(wsCmd)
	rootCmd.AddCommand(repoCmd)
	rootCmd.AddCommand(initCmd)
}
