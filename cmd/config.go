package cmd

import (
	"log"

	"github.com/spf13/cobra"

	"qail/internal/config"
	"qail/internal/forms"
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
			fn := func(cfg *config.Config) error {
				forms.DisplayConfig(*cfg)
				return nil
			}

			HandleConfig(fn)

		},
	}
	configEditorCmd = &cobra.Command{
		Use:  "editor",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fn := func(cfg *config.Config) error {
				cfg.Editor = args[0]
				return nil
			}

			HandleConfig(fn)
		},
	}
	configRootCmd = &cobra.Command{
		Use:  "root",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {

			fn := func(cfg *config.Config) error {
				cfg.Root = args[0]
				return nil
			}

			HandleConfig(fn)
		},
	}
	configCmd = &cobra.Command{
		Use:   "config",
		Short: "Manage the qail config",
		Run:   runConfigCmd(),
	}
)

func runConfigCmd() cobraReturnType {
	return func(cmd *cobra.Command, args []string) {
		for _, arg := range args {
			switch arg {
			case "convert":
				configConvertCmd.Execute()
			case "root":
				configRootCmd.Execute()
			case "editor":
				configEditorCmd.Execute()
			case "list":
				configLsCmd.Execute()
			}
		}
	}
}

func init() {
	configCmd.AddCommand(configRootCmd, configEditorCmd, configLsCmd, configConvertCmd)
}

func HandleConfig(fn func(*config.Config) error) {
	err := config.WithConfig(fn)
	if err != nil {
		log.Fatalln(err)
	}
}
