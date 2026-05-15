package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"

	"github.com/ubaniak/qail/internal/actions"
	"github.com/ubaniak/qail/internal/color"
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
			cdCmd, err := scripts.Default().CdCommand()
			if err != nil {
				log.Fatalln(err)
			}
			fmt.Printf("%s copied %s to clipboard\n\n", color.Yellow(">>>"), color.Green(cdCmd))
			clipboard.WriteAll(cdCmd)
		},
	}
	addScriptCmd = &cobra.Command{
		Use:     "add [name]",
		Aliases: []string{"a"},
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := firstArg(args)
			if name == "" {
				if err := requireTTY("script name"); err != nil {
					log.Fatalln(err)
				}
				n, err := forms.NewScript()
				if err != nil {
					log.Fatalln(err)
				}
				name = n
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
		Use:     "open [name]",
		Aliases: []string{"o"},
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			editor, err := actions.DefaultEditorCommand(mustStore())
			if err != nil {
				log.Fatalln(err)
			}
			if editor == "" {
				log.Fatalln("no default editor configured; run `qail config editor add` first")
			}
			sc := scripts.Default()

			if name := firstArg(args); name != "" {
				ok, err := sc.Has(name)
				if err != nil {
					log.Fatalln(err)
				}
				if !ok {
					log.Fatalf("script %q not found\n", name)
				}
				if err := sc.Open(context.Background(), editor, name); err != nil {
					log.Fatalln(err)
				}
				return
			}

			if err := requireTTY("script name"); err != nil {
				log.Fatalln(err)
			}
			allScripts, err := sc.ListScripts()
			if err != nil {
				log.Fatalln(err)
			}
			script, err := forms.SelectScript(allScripts)
			if err != nil {
				log.Fatalln(err)
			}
			if err := sc.Open(context.Background(), editor, script); err != nil {
				log.Fatalln(err)
			}
		},
	}
	removeScriptCmd = &cobra.Command{
		Use:     "remove [name]",
		Aliases: []string{"rm"},
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			sc := scripts.Default()

			if name := firstArg(args); name != "" {
				ok, err := sc.Has(name)
				if err != nil {
					log.Fatalln(err)
				}
				if !ok {
					log.Fatalf("script %q not found\n", name)
				}
				confirmed, err := confirmOrSkip(fmt.Sprintf("Remove script %q?", name), "--yes for non-interactive remove")
				if err != nil {
					log.Fatalln(err)
				}
				if !confirmed {
					fmt.Fprintln(os.Stdout, "Aborting")
					return
				}
				fmt.Printf("Removing %s\n", name)
				if err := sc.RemoveScript(name); err != nil {
					log.Fatalln(err)
				}
				return
			}

			if err := requireTTY("script name"); err != nil {
				log.Fatalln(err)
			}
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
