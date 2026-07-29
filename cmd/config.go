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
			s, jsonPath, err := config.DefaultSQLiteStore()
			if err != nil {
				log.Fatalln(err)
			}
			switch args[0] {
			case "new":
				if err := actions.ConvertConfig(s, jsonPath); err != nil {
					log.Fatalln(err)
				}
			case "restore":
				if err := actions.RestoreConfig(s); err != nil {
					log.Fatalln(err)
				}
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
		Use:   "editor",
		Short: "Manage editors",
	}
	configEditorAddCmd = &cobra.Command{
		Use:   "add <name> <command>",
		Short: "Register an editor (e.g. add vscode code)",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if err := actions.AddEditor(mustStore(), args[0], args[1]); err != nil {
				log.Fatalln(err)
			}
		},
	}
	configEditorRmCmd = &cobra.Command{
		Use:     "remove <name>",
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := actions.RemoveEditor(mustStore(), args[0]); err != nil {
				log.Fatalln(err)
			}
		},
	}
	configEditorDefaultCmd = &cobra.Command{
		Use:   "default <name>",
		Short: "Set the global default editor",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := actions.SetDefaultEditor(mustStore(), args[0]); err != nil {
				log.Fatalln(err)
			}
		},
	}
	configEditorLsCmd = &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Run: func(cmd *cobra.Command, args []string) {
			editors, def, err := actions.ListEditors(mustStore())
			if err != nil {
				log.Fatalln(err)
			}
			forms.DisplayEditors(editors, def)
		},
	}
	configAiCmd = &cobra.Command{
		Use:   "ai",
		Short: "Manage AI tools",
	}
	configAiAddCmd = &cobra.Command{
		Use:   "add <name> <command>",
		Short: "Register an AI tool (e.g. add claude claude)",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if err := actions.AddAI(mustStore(), args[0], args[1]); err != nil {
				log.Fatalln(err)
			}
		},
	}
	configAiRmCmd = &cobra.Command{
		Use:     "remove <name>",
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := actions.RemoveAI(mustStore(), args[0]); err != nil {
				log.Fatalln(err)
			}
		},
	}
	configAiDefaultCmd = &cobra.Command{
		Use:   "default <name>",
		Short: "Set the global default AI tool",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := actions.SetDefaultAI(mustStore(), args[0]); err != nil {
				log.Fatalln(err)
			}
		},
	}
	configAiLsCmd = &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Run: func(cmd *cobra.Command, args []string) {
			ais, def, err := actions.ListAI(mustStore())
			if err != nil {
				log.Fatalln(err)
			}
			forms.DisplayAIs(ais, def)
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
	configEditorCmd.AddCommand(configEditorAddCmd, configEditorRmCmd, configEditorDefaultCmd, configEditorLsCmd)
	configAiCmd.AddCommand(configAiAddCmd, configAiRmCmd, configAiDefaultCmd, configAiLsCmd)
	configCmd.AddCommand(configRootCmd, configEditorCmd, configAiCmd, configLsCmd, configConvertCmd)
}
