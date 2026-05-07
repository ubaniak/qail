package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"

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
			if err := scripts.Default().Cd(); err != nil {
				log.Fatalln(err)
			}
		},
	}
	addScriptCmd = &cobra.Command{
		Use:     "add",
		Aliases: []string{"a"},
		Run: func(cmd *cobra.Command, args []string) {
			name, err := forms.NewScript()
			if err != nil {
				log.Fatalln(err)
			}
			if err := scripts.Default().CreateBashScript(name); err != nil {
				log.Fatalln(err)
			}
		},
	}
	lsScriptCmd = &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Run: func(cmd *cobra.Command, args []string) {
			scriptList, err := scripts.Default().ListScripts()
			if err != nil {
				log.Fatalln(err)
			}
			forms.DisplayScripts(scriptList)
		},
	}
	openScriptCmd = &cobra.Command{
		Use:     "open",
		Aliases: []string{"o"},
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := mustStore().Read()
			if err != nil {
				log.Fatalln(err)
			}
			sc := scripts.Default()
			allScripts, err := sc.ListScripts()
			if err != nil {
				log.Fatalln(err)
			}
			script, err := forms.SelectScript(allScripts)
			if err != nil {
				log.Fatalln(err)
			}
			if err := sc.Open(cfg.Editor, script); err != nil {
				log.Fatalln(err)
			}
		},
	}
	removeScriptCmd = &cobra.Command{
		Use:     "remove",
		Aliases: []string{"rm"},
		Run: func(cmd *cobra.Command, args []string) {
			sc := scripts.Default()
			allScripts, err := sc.ListScripts()
			if err != nil {
				log.Fatalln(err)
			}
			script, err := forms.SelectScript(allScripts)
			if err != nil {
				log.Fatalln(err)
			}

			confirm, err := forms.Confirm("This will remove the selected script. Do you want to continue?")
			if err != nil {
				log.Fatalln(err)
			}
			if !confirm {
				fmt.Println("Aborting")
				return
			}
			fmt.Printf("Removing %s\n", script)

			if err := sc.RemoveScript(script); err != nil {
				log.Fatalln(err)
			}
		},
	}
)

func init() {
	scriptsCmd.AddCommand(addScriptCmd, lsScriptCmd, openScriptCmd, removeScriptCmd, cdScriptCmd)
}
