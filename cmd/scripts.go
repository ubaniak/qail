package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ubaniak/qail/internal/config"
	"github.com/ubaniak/qail/internal/forms"
	"github.com/ubaniak/qail/internal/scripts"
)

var (
	scriptsCmd = &cobra.Command{
		Use:     "scripts",
		Short:   "manage pre|post install scripts",
		Aliases: []string{"s"},
	}
	cdScriptCmd = &cobra.Command{
		Use: "cd",
		Run: func(cmd *cobra.Command, args []string) {
			fn := func(cfg *config.Config) error {
				return scripts.Cd()
			}

			HandleConfig(fn)
		},
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
				scriptList, err := scripts.ListScripts()
				if err != nil {
					return err
				}
				forms.DisplayScripts(scriptList)

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

func init() {
	scriptsCmd.AddCommand(addScriptCmd, lsScriptCmd, openScriptCmd, removeScriptCmd, cdScriptCmd)
}
