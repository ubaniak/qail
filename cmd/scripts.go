package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"

	"github.com/ubaniak/qail/internal/color"
	"github.com/ubaniak/qail/internal/forms"
	"github.com/ubaniak/qail/internal/scripts"
)

// scriptScope is the shared flag value bound to every scripts subcommand
// that needs to disambiguate workspace vs repo buckets. Defaults to
// workspace so existing CLI invocations continue to act on the workspace
// post-install scripts.
var scriptScope string

// resolveScope maps the --scope flag to a scripts.Scope, erroring out on
// an unknown value so the CLI never silently writes into the wrong bucket.
func resolveScope() scripts.Scope {
	s := scripts.Scope(scriptScope)
	if !s.Valid() {
		log.Fatalf("invalid --scope %q (want %q or %q)\n", scriptScope, scripts.ScopeWorkspace, scripts.ScopeRepo)
	}
	return s
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
			scope := resolveScope()
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
			if err := scripts.Default().CreateBashScript(name, scope); err != nil {
				log.Fatalln(err)
			}
		},
	}
	lsScriptCmd = &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Run: func(cmd *cobra.Command, args []string) {
			scope := resolveScope()
			scriptList, err := scripts.Default().ListScripts(scope)
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
			scope := resolveScope()
			sc := scripts.Default()

			if name := firstArg(args); name != "" {
				ok, err := sc.Has(scope, name)
				if err != nil {
					log.Fatalln(err)
				}
				if !ok {
					log.Fatalf("script %q not found in %s scope\n", name, scope)
				}
				if err := sc.OpenDefault(context.Background(), scope, name); err != nil {
					log.Fatalln(err)
				}
				return
			}

			if err := requireTTY("script name"); err != nil {
				log.Fatalln(err)
			}
			allScripts, err := sc.ListScripts(scope)
			if err != nil {
				log.Fatalln(err)
			}
			script, err := forms.SelectScript(allScripts)
			if err != nil {
				log.Fatalln(err)
			}
			if err := sc.OpenDefault(context.Background(), scope, script); err != nil {
				log.Fatalln(err)
			}
		},
	}
	removeScriptCmd = &cobra.Command{
		Use:     "remove [name]",
		Aliases: []string{"rm"},
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			scope := resolveScope()
			sc := scripts.Default()

			if name := firstArg(args); name != "" {
				ok, err := sc.Has(scope, name)
				if err != nil {
					log.Fatalln(err)
				}
				if !ok {
					log.Fatalf("script %q not found in %s scope\n", name, scope)
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
				if err := sc.RemoveScript(name, scope); err != nil {
					log.Fatalln(err)
				}
				return
			}

			if err := requireTTY("script name"); err != nil {
				log.Fatalln(err)
			}
			allScripts, err := sc.ListScripts(scope)
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

			if err := sc.RemoveScript(script, scope); err != nil {
				log.Fatalln(err)
			}
		},
	}
)

func init() {
	for _, c := range []*cobra.Command{addScriptCmd, lsScriptCmd, openScriptCmd, removeScriptCmd} {
		c.Flags().StringVar(&scriptScope, "scope", string(scripts.ScopeWorkspace), "script scope: workspace or repo")
	}
	scriptsCmd.AddCommand(addScriptCmd, lsScriptCmd, openScriptCmd, removeScriptCmd, cdScriptCmd)
}
