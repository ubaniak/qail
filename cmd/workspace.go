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
	"github.com/ubaniak/qail/internal/config"
	"github.com/ubaniak/qail/internal/forms"
	"github.com/ubaniak/qail/internal/runner"
	"github.com/ubaniak/qail/internal/scripts"
	"github.com/ubaniak/qail/internal/workspace"
)

// flag-backed inputs for non-interactive workspace commands.
var (
	wsAddRepos    []string
	wsEditRepos   []string
	wsCloneRepos  []string
	wsPIScripts   []string
	wsPIClear     bool
	wsOpenEditor  string
	wsEditorUnset bool
)

// resolveWorkspaceName returns a workspace name. Validates against the
// registry when `name` is set; otherwise opens the interactive picker
// (--no-tty turns the missing input into a hard error). Disk-existence and
// LastUsed touch happen inside the matching action, not here.
func resolveWorkspaceName(s config.Store, name string) (string, error) {
	ctx, err := actions.ReadWorkspaceContext(s)
	if err != nil {
		return "", err
	}
	if name != "" {
		if _, ok := ctx.Workspaces[name]; !ok {
			return "", fmt.Errorf("workspace %q not found", name)
		}
		return name, nil
	}
	if err := requireTTY("workspace name"); err != nil {
		return "", err
	}
	r, err := forms.FindWorkspace(ctx.Workspaces)
	if err != nil {
		return "", err
	}
	return r.Name, nil
}

// firstArg returns args[0] when present, "" otherwise.
func firstArg(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return ""
}

var (
	wsCmd = &cobra.Command{
		Use:     "workspace",
		Aliases: []string{"ws"},
		Short:   "Manage your workspaces",
	}
	exploreCmd = &cobra.Command{
		Use:     "explore [name]",
		Aliases: []string{"exp"},
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			s := mustStore()
			name, err := resolveWorkspaceName(s, firstArg(args))
			if err != nil {
				log.Fatalln(err)
			}
			if err := actions.ExploreWorkspace(context.Background(), s, runner.NewOS(), name); err != nil {
				log.Fatalln(err)
			}
		},
	}
	openWsCmd = &cobra.Command{
		Use:     "open [name]",
		Aliases: []string{"o"},
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			s := mustStore()
			name, err := resolveWorkspaceName(s, firstArg(args))
			if err != nil {
				log.Fatalln(err)
			}
			if err := actions.OpenWorkspaceWith(context.Background(), s, runner.NewOS(), name, wsOpenEditor); err != nil {
				log.Fatalln(err)
			}
		},
	}
	editorWsCmd = &cobra.Command{
		Use:   "editor <workspace> [name]",
		Short: "Set the workspace's preferred editor (or --unset to inherit global)",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			s := mustStore()
			ws := args[0]
			var editorName string
			if wsEditorUnset {
				editorName = ""
			} else if len(args) == 2 {
				editorName = args[1]
			} else {
				log.Fatalln("editor name required (or pass --unset)")
			}
			if err := actions.SetWorkspaceEditor(s, ws, editorName); err != nil {
				log.Fatalln(err)
			}
		},
	}
	cdWsCmd = &cobra.Command{
		Use:  "cd [name]",
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			s := mustStore()
			name, err := resolveWorkspaceName(s, firstArg(args))
			if err != nil {
				log.Fatalln(err)
			}
			wsPath, err := actions.CdWorkspace(s, name)
			if err != nil {
				log.Fatalln(err)
			}
			cdCmd := fmt.Sprintf("cd %s", wsPath)
			fmt.Printf("%s copied %s to clipboard\n\n", color.Yellow(">>>"), color.Green(cdCmd))
			clipboard.WriteAll(cdCmd)
		},
	}
	tmuxWsCmd = &cobra.Command{
		Use:     "mux [name]",
		Aliases: []string{"m"},
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			s := mustStore()
			name, err := resolveWorkspaceName(s, firstArg(args))
			if err != nil {
				log.Fatalln(err)
			}
			attachCmd, err := actions.MuxWorkspace(context.Background(), s, name)
			if err != nil {
				log.Fatalln(err)
			}
			fmt.Printf("%s copied %s to clipboard\n\n", color.Yellow(">>>"), color.Green(attachCmd))
			clipboard.WriteAll(attachCmd)
		},
	}
	removeWsCmd = &cobra.Command{
		Use:     "remove [name]",
		Aliases: []string{"rm"},
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			s := mustStore()

			// Non-interactive: name positional, --yes skips confirm.
			if name := firstArg(args); name != "" {
				ctx, err := actions.ReadWorkspaceContext(s)
				if err != nil {
					log.Fatalln(err)
				}
				if _, ok := ctx.Workspaces[name]; !ok {
					log.Fatalf("workspace %q not found\n", name)
				}
				ok, err := confirmOrSkip(fmt.Sprintf("Remove workspace %q?", name), "--yes for non-interactive remove")
				if err != nil {
					log.Fatalln(err)
				}
				if !ok {
					return
				}
				if err := actions.RemoveWorkspace(s, name); err != nil {
					log.Fatalln(err)
				}
				return
			}

			if err := requireTTY("workspace name"); err != nil {
				log.Fatalln(err)
			}
			ctx, err := actions.ReadWorkspaceContext(s)
			if err != nil {
				log.Fatalln(err)
			}
			name, err := forms.SelectWorkspaceName(ctx.Workspaces)
			if err != nil {
				log.Fatalln(err)
			}
			confirmed, err := forms.Confirm(fmt.Sprintf("Remove workspace %q?", name))
			if err != nil {
				log.Fatalln(err)
			}
			if !confirmed {
				return
			}
			if err := actions.RemoveWorkspace(s, name); err != nil {
				log.Fatalln(err)
			}
		},
	}
	listWsCmd = &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			ws, postInstall, err := actions.ListWorkspaces(mustStore())
			if err != nil {
				log.Fatalln(err)
			}
			forms.DisplayWorkspaces(ws, postInstall)
		},
	}
	createWsCmd = &cobra.Command{
		Use:     "create [name]",
		Aliases: []string{"c"},
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			s := mustStore()

			if name := firstArg(args); name != "" {
				ctx, err := actions.ReadWorkspaceContext(s)
				if err != nil {
					log.Fatalln(err)
				}
				if _, ok := ctx.Workspaces[name]; !ok {
					log.Fatalf("workspace %q not found in registry; use `ws add` first\n", name)
				}
				if err := actions.CreateWorkspaceOnDisk(context.Background(), s, name, nil); err != nil {
					log.Fatalln(err)
				}
				return
			}

			if err := requireTTY("workspace name"); err != nil {
				log.Fatalln(err)
			}
			ctx, err := actions.ReadWorkspaceContext(s)
			if err != nil {
				log.Fatalln(err)
			}
			r, err := forms.FindWorkspace(ctx.Workspaces)
			if err != nil {
				log.Fatalln(err)
			}
			if err := actions.CreateWorkspaceOnDisk(context.Background(), s, r.Name, nil); err != nil {
				log.Fatalln(err)
			}
		},
	}
	cloneWsCmd = &cobra.Command{
		Use:  "clone [src] [dst]",
		Args: cobra.MaximumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			s := mustStore()

			// Non-interactive: src + dst positional. --repo overrides src's
			// package list; without it the dst inherits src's repos.
			if len(args) == 2 {
				src, dst := args[0], args[1]
				ctx, err := actions.ReadWorkspaceContext(s)
				if err != nil {
					log.Fatalln(err)
				}
				srcProfile, ok := ctx.Workspaces[src]
				if !ok {
					log.Fatalf("source workspace %q not found\n", src)
				}
				if _, exists := ctx.Workspaces[dst]; exists {
					log.Fatalf("destination workspace %q already exists\n", dst)
				}
				pkgs := srcProfile.Repos
				if len(wsCloneRepos) > 0 {
					pkgs = wsCloneRepos
				}
				if err := actions.CloneWorkspace(context.Background(), s, dst, pkgs, nil); err != nil {
					log.Fatalln(err)
				}
				return
			}

			if err := requireTTY("src + dst workspace names"); err != nil {
				log.Fatalln(err)
			}
			ctx, err := actions.ReadWorkspaceContext(s)
			if err != nil {
				log.Fatalln(err)
			}
			f, err := forms.FindWorkspace(ctx.Workspaces)
			if err != nil {
				log.Fatalln(err)
			}
			c, err := forms.CloneWorkspace(f.Name, f.Packages)
			if err != nil {
				log.Fatalln(err)
			}
			if err := actions.CloneWorkspace(context.Background(), s, c.Name, c.Packages, nil); err != nil {
				log.Fatalln(err)
			}
		},
	}
	addWsCmd = &cobra.Command{
		Use:     "add [name]",
		Aliases: []string{"a"},
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			s := mustStore()

			// Non-interactive: name positional. --repo optional (empty
			// list creates an empty workspace, matching what the TUI
			// allows when no packages are selected).
			if name := firstArg(args); name != "" {
				if err := actions.AddWorkspace(context.Background(), s, name, wsAddRepos, nil); err != nil {
					log.Fatalln(err)
				}
				return
			}

			if err := requireTTY("workspace name"); err != nil {
				log.Fatalln(err)
			}
			cfg, err := s.Read()
			if err != nil {
				log.Fatalln(err)
			}
			r, err := forms.NewWorkspace(cfg.Repos)
			if err != nil {
				log.Fatalln(err)
			}
			if err := actions.AddWorkspace(context.Background(), s, r.Name, r.Packages, nil); err != nil {
				log.Fatalln(err)
			}
		},
	}
	editWsCmd = &cobra.Command{
		Use:     "edit [name]",
		Aliases: []string{"e"},
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			s := mustStore()

			// Non-interactive: name positional + --repo (full replacement).
			// --repo with zero values clears the package list — same shape
			// as the TUI letting you deselect everything.
			if name := firstArg(args); name != "" && cmd.Flags().Changed("repo") {
				if err := actions.EditWorkspace(context.Background(), s, name, wsEditRepos, nil); err != nil {
					log.Fatalln(err)
				}
				return
			}

			if err := requireTTY("workspace name + --repo flags"); err != nil {
				log.Fatalln(err)
			}
			cfg, err := s.Read()
			if err != nil {
				log.Fatalln(err)
			}
			r, err := forms.FindWorkspace(cfg.Workspaces)
			if err != nil {
				log.Fatalln(err)
			}
			e, err := forms.EditWorkspace(r.Name, r.Packages, cfg.Repos)
			if err != nil {
				log.Fatalln(err)
			}
			if err := actions.EditWorkspace(context.Background(), s, e.Name, e.Packages, nil); err != nil {
				log.Fatalln(err)
			}
		},
	}
	cleanWSCmd = &cobra.Command{
		Use: "clean",
		Run: func(cmd *cobra.Command, args []string) {
			s := mustStore()
			ctx, err := actions.ReadWorkspaceContext(s)
			if err != nil {
				log.Fatalln(err)
			}
			ok, err := confirmOrSkip("This will delete untracked files in your workspace. Are you sure?", "--yes for non-interactive clean")
			if err != nil {
				log.Fatalln(err)
			}
			if !ok {
				return
			}
			if err := workspace.Clean(ctx.Root, ctx.Workspaces, os.Stdout); err != nil {
				log.Fatalln(err)
			}
		},
	}
	postInstallScriptWsCmd = &cobra.Command{
		Use:     "post-install-script [name]",
		Aliases: []string{"p"},
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			s := mustStore()
			wsMap, postInstall, err := actions.ListWorkspaces(s)
			if err != nil {
				log.Fatalln(err)
			}

			// Non-interactive: ws name + --script (repeatable) or --clear.
			if name := firstArg(args); name != "" && (len(wsPIScripts) > 0 || wsPIClear) {
				if _, ok := wsMap[name]; !ok {
					log.Fatalf("workspace %q not found\n", name)
				}
				out := wsPIScripts
				if wsPIClear {
					out = nil
				}
				if err := actions.SetWorkspacePostInstall(s, name, out); err != nil {
					log.Fatalln(err)
				}
				return
			}

			if err := requireTTY("workspace name + --script/--clear flags"); err != nil {
				log.Fatalln(err)
			}

			var name string
			if n := firstArg(args); n != "" {
				if _, ok := wsMap[n]; !ok {
					log.Fatalf("workspace %q not found\n", n)
				}
				name = n
			} else {
				ws, err := forms.FindWorkspace(wsMap)
				if err != nil {
					log.Fatalln(err)
				}
				name = ws.Name
			}
			selected := postInstall[name]

			scriptList, err := scripts.Default().ListScripts()
			if err != nil {
				log.Fatalln(err)
			}
			updated, err := forms.SelectScripts(scriptList, selected)
			if err != nil {
				log.Fatalln(err)
			}
			if err := actions.SetWorkspacePostInstall(s, name, updated); err != nil {
				log.Fatalln(err)
			}
		},
	}
)

func init() {
	addWsCmd.Flags().StringSliceVarP(&wsAddRepos, "repo", "r", nil, "repo name to include (repeatable, or comma-separated)")
	editWsCmd.Flags().StringSliceVarP(&wsEditRepos, "repo", "r", nil, "repo names that fully replace the workspace's package list")
	cloneWsCmd.Flags().StringSliceVarP(&wsCloneRepos, "repo", "r", nil, "override repo list for the clone (defaults to source)")
	postInstallScriptWsCmd.Flags().StringSliceVarP(&wsPIScripts, "script", "s", nil, "post-install script name (repeatable, or comma-separated)")
	postInstallScriptWsCmd.Flags().BoolVar(&wsPIClear, "clear", false, "clear all post-install scripts for the workspace")
	openWsCmd.Flags().StringVarP(&wsOpenEditor, "editor", "e", "", "editor name override (defaults to workspace/global)")
	editorWsCmd.Flags().BoolVar(&wsEditorUnset, "unset", false, "clear the workspace's editor override")

	wsCmd.AddCommand(addWsCmd, listWsCmd, createWsCmd, cloneWsCmd, editWsCmd, removeWsCmd, cdWsCmd, openWsCmd, cleanWSCmd, tmuxWsCmd, postInstallScriptWsCmd, exploreCmd, editorWsCmd)
}
