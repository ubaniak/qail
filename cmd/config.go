package cmd

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/ubaniak/qail/internal/actions"
	"github.com/ubaniak/qail/internal/config"
	"github.com/ubaniak/qail/internal/forms"
)

var (
	configConvertCmd = &cobra.Command{
		Use:       "convert",
		ValidArgs: []string{"new", "restore"},
		Args:      cobra.OnlyValidArgs,
		Run: func(cmd *cobra.Command, args []string) {
			a := args[0]
			if a == "new" {
				config.BackUpConfig()
				config.ConvertOldToNew()
			}

			if a == "restore" {
				config.RestoreConfig()
			}
		},
	}
	configLsCmd = &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := mustStore().Read()
			if err != nil {
				log.Fatalln(err)
			}
			forms.DisplayConfig(cfg)
		},
	}
	configEditorCmd = &cobra.Command{
		Use:  "editor",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := actions.SetEditor(mustStore(), args[0]); err != nil {
				log.Fatalln(err)
			}
		},
	}
	configRootCmd = &cobra.Command{
		Use:  "root",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := actions.SetRoot(mustStore(), args[0]); err != nil {
				log.Fatalln(err)
			}
		},
	}
	configCmd = &cobra.Command{
		Use:   "config",
		Short: "Manage the qail config",
	}
)

func init() {
	configCmd.AddCommand(configRootCmd, configEditorCmd, configLsCmd, configConvertCmd)
}
