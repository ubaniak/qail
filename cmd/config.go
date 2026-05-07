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

// HandleConfig runs fn against the loaded config and writes any mutation
// back, logging fatally on error.
//
// Deprecated: prefer the actions package, which exposes one function per
// user-facing operation backed by a config.Store. New cmd handlers should
// not introduce HandleConfig calls.
func HandleConfig(fn func(*config.Config) error) {
	err := config.WithConfig(fn)
	if err != nil {
		log.Fatalln(err)
	}
}
