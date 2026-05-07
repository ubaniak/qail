package cmd

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"

	"github.com/ubaniak/qail/internal/actions"
	"github.com/ubaniak/qail/internal/color"
	"github.com/ubaniak/qail/internal/config"
	"github.com/ubaniak/qail/internal/forms"
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
)

// resolveWorkspaceName returns a workspace name + its on-disk path.
// If `name` is non-empty it is validated against the registry; otherwise
// a TUI selector runs (unless --no-tty, which becomes a hard error).
func resolveWorkspaceName(s config.Store, name string) (string, string, []string, error) {
	ctx, err := actions.ReadWorkspaceContext(s)
	if err != nil {
		return "", "", nil, err
	}

	if name != "" {
		profile, ok := ctx.Workspaces[name]
		if !ok {
			return "", "", nil, fmt.Errorf("workspace %q not found", name)
		}
		wsPath := path.Join(ctx.Root, name)
		if _, err := os.Stat(wsPath); os.IsNotExist(err) {
			return "", "", nil, fmt.Errorf("workspace %q does not exist on disk. Please run qail ws create", wsPath)
		}
		return name, wsPath, profile.Repos, nil
	}

	if err := requireTTY("workspace name"); err != nil {
		return "", "", nil, err
	}
	r, err := forms.FindWorkspace(ctx.Workspaces)
	if err != nil {
		return "", "", nil, err
	}
	wsPath := path.Join(ctx.Root, r.Name)
	if _, err := os.Stat(wsPath); os.IsNotExist(err) {
		return "", "", nil, fmt.Errorf("workspace %q does not exist. Please run qail ws create", wsPath)
	}
	return r.Name, wsPath, r.Packages, nil
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
			name, wsPath, _, err := resolveWorkspaceName(s, firstArg(args))
			if err != nil {
				log.Fatalln(err)
			}
			_ = name
			workspace.Explore(wsPath)
		},
	}
	openWsCmd = &cobra.Command{
		Use:     "open [name]",
		Aliases: []string{"o"},
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			s := mustStore()
			name, wsPath, _, err := resolveWorkspaceName(s, firstArg(args))
			if err != nil {
				log.Fatalln(err)
			}
			if _, err := actions.TouchWorkspace(s, name); err != nil {
				log.Fatalln(err)
			}
			ctx, err := actions.ReadWorkspaceContext(s)
			if err != nil {
				log.Fatalln(err)
			}
			workspace.Open(ctx.Editor, wsPath)
		},
	}
	cdWsCmd = &cobra.Command{
		Use:  "cd [name]",
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			s := mustStore()
			name, wsPath, _, err := resolveWorkspaceName(s, firstArg(args))
			if err != nil {
				log.Fatalln(err)
			}
			if _, err := actions.TouchWorkspace(s, name); err != nil {
				log.Fatalln(err)
			}
			workspace.Cd(wsPath)
		},
	}
	tmuxWsCmd = &cobra.Command{
		Use:     "mux [name]",
		Aliases: []string{"m"},
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			s := mustStore()
			name, wsPath, _, err := resolveWorkspaceName(s, firstArg(args))
			if err != nil {
				log.Fatalln(err)
			}
			if _, err := actions.TouchWorkspace(s, name); err != nil {
				log.Fatalln(err)
			}
			attachCmd, err := workspace.Tmux(wsPath)
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
				if !flagYes {
					if err := requireTTY("--yes for non-interactive remove"); err != nil {
						log.Fatalln(err)
					}
					ok, err := forms.Confirm(fmt.Sprintf("Remove workspace %q?", name))
					if err != nil {
						log.Fatalln(err)
					}
					if !ok {
						return
					}
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
			name, confirmed, err := forms.SelectWorkspaceToRemove(ctx.Workspaces)
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
				if err := actions.CreateWorkspaceOnDisk(s, name); err != nil {
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
			if err := actions.CreateWorkspaceOnDisk(s, r.Name); err != nil {
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
				if err := actions.CloneWorkspace(s, dst, pkgs); err != nil {
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
			if err := actions.CloneWorkspace(s, c.Name, c.Packages); err != nil {
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
				if err := actions.AddWorkspace(s, name, wsAddRepos); err != nil {
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
			if err := actions.AddWorkspace(s, r.Name, r.Packages); err != nil {
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
				if err := actions.EditWorkspace(s, name, wsEditRepos); err != nil {
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
			if err := actions.EditWorkspace(s, e.Name, e.Packages); err != nil {
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
			if !flagYes {
				if err := requireTTY("--yes for non-interactive clean"); err != nil {
					log.Fatalln(err)
				}
				ok, err := forms.CleanWorkspace()
				if err != nil {
					log.Fatalln(err)
				}
				if !ok {
					return
				}
			}
			if err := workspace.Clean(ctx.Root, ctx.Workspaces); err != nil {
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

	wsCmd.AddCommand(addWsCmd, listWsCmd, createWsCmd, cloneWsCmd, editWsCmd, removeWsCmd, cdWsCmd, openWsCmd, cleanWSCmd, tmuxWsCmd, postInstallScriptWsCmd, exploreCmd)
}
