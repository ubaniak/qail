package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"qail/internal/config"
	"qail/internal/forms"
	"qail/internal/scripts"
)

var (
	scriptsCmd = &cobra.Command{
		Use:     "scripts",
		Short:   "manage pre|post install scripts",
		Aliases: []string{"s"},
		Run:     runScriptsCmd(),
	}
	addScriptCmd = &cobra.Command{
		Use:     "add",
		Aliases: []string{"a"},
		Run: func(cmd *cobra.Command, args []string) {
			fn := func(cfg *config.Config) error {
				s, err := forms.NewScript()
				scripts.CreateBashScript(s)

				return err
			}

			HandleConfig(fn)
		},
	}
	lsScriptCmd = &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Run: func(cmd *cobra.Command, args []string) {
			fn := func(cfg *config.Config) error {
				scripts, err := scripts.ListScripts()
				if err != nil {
					return err
				}
				forms.DisplayScripts(scripts)

				return nil
			}

			HandleConfig(fn)
		},
	}
	openScriptCmd = &cobra.Command{
		Use:     "open",
		Aliases: []string{"o"},
		Run: func(cmd *cobra.Command, args []string) {
			fn := func(cfg *config.Config) error {
				allScripts, err := scripts.ListScripts()
				if err != nil {
					return err
				}
				script, err := forms.SelectScript(allScripts)
				if err != nil {
					return err
				}

				return scripts.Open(cfg.Editor, script)
			}

			HandleConfig(fn)
		},
	}
	removeScriptCmd = &cobra.Command{
		Use:     "remove",
		Aliases: []string{"rm"},
		Run: func(cmd *cobra.Command, args []string) {
			fn := func(cfg *config.Config) error {
				allScripts, err := scripts.ListScripts()
				if err != nil {
					return err
				}
				script, err := forms.SelectScript(allScripts)
				if err != nil {
					return err
				}

				confirm, err := forms.Confirm("This will remove the selected script. Do you want to continue?")
				if err != nil {
					return err
				}

				if !confirm {
					fmt.Println("Aborting")
					return nil
				}
				fmt.Printf("Removing %s\n", script)

				return scripts.RemoveScript(script)
			}

			HandleConfig(fn)
		},
	}
)

func runScriptsCmd() cobraReturnType {
	return func(cmd *cobra.Command, args []string) {
		for _, arg := range args {
			switch arg {
			case "add":
				addScriptCmd.Execute()
			case "list":
				lsScriptCmd.Execute()
			case "remove":
				removeScriptCmd.Execute()
			case "open":
				openScriptCmd.Execute()
			}
		}
	}
}

func init() {
	scriptsCmd.AddCommand(addScriptCmd, lsScriptCmd, openScriptCmd, removeScriptCmd)
}
