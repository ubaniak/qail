package cmd

import (
	"fmt"
	"log"
	"slices"

	"github.com/spf13/cobra"

	"github.com/ubaniak/qail/internal/forms"
	"github.com/ubaniak/qail/internal/scripts"
)

// scriptExists reports whether name appears in the scripts dir. Used by
// non-interactive paths so a typo fails loudly before we touch the
// filesystem.
func scriptExists(sc *scripts.Scripts, name string) (bool, error) {
	all, err := sc.ListScripts()
	if err != nil {
		return false, err
	}
	return slices.Contains(all, name), nil
}

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
			cfg, err := mustStore().Read()
			if err != nil {
				log.Fatalln(err)
			}
			sc := scripts.Default()

			if name := firstArg(args); name != "" {
				ok, err := scriptExists(sc, name)
				if err != nil {
					log.Fatalln(err)
				}
				if !ok {
					log.Fatalf("script %q not found\n", name)
				}
				if err := sc.Open(cfg.Editor, name); err != nil {
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
			if err := sc.Open(cfg.Editor, script); err != nil {
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
				ok, err := scriptExists(sc, name)
				if err != nil {
					log.Fatalln(err)
				}
				if !ok {
					log.Fatalf("script %q not found\n", name)
				}
				if !flagYes {
					if err := requireTTY("--yes for non-interactive remove"); err != nil {
						log.Fatalln(err)
					}
					confirm, err := forms.Confirm(fmt.Sprintf("Remove script %q?", name))
					if err != nil {
						log.Fatalln(err)
					}
					if !confirm {
						fmt.Println("Aborting")
						return
					}
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
