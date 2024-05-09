package cmd

import (
	"log"
	"qail/internal/config"
	"qail/internal/forms"

	"github.com/spf13/cobra"
)

var (
	configLsCmd = &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.ReadFromFile()
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
			cfg, err := config.ReadFromFile()
			if err != nil {
				log.Fatalln(err)
			}

			cfg.Editor = args[0]

			err = config.WriteToFile(cfg)
			if err != nil {
				log.Fatalln(err)
			}
		},
	}
	configRootCmd = &cobra.Command{
		Use:  "root",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.ReadFromFile()
			if err != nil {
				log.Fatalln(err)
			}

			cfg.Root = args[0]

			err = config.WriteToFile(cfg)
			if err != nil {
				log.Fatalln(err)
			}
		},
	}
	configCmd = &cobra.Command{
		Use:   "config",
		Short: "Manage the qail config",
		Run:   runCacheCmd(),
	}
)

func runCacheCmd() cobraReturnType {
	return func(cmd *cobra.Command, args []string) {
		for _, arg := range args {
			switch arg {
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
	configCmd.AddCommand(configRootCmd, configEditorCmd, configLsCmd)
}
