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

// resolveWorkspacePath reads a snapshot, lets the user pick a workspace,
// and returns the on-disk path plus the picked profile. Used by every
// flow that needs to "select an existing workspace and act on it" before
// the action wraps the LastUsed update.
func resolveWorkspacePath(s config.Store) (string, string, []string, error) {
	ctx, err := actions.ReadWorkspaceContext(s)
	if err != nil {
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

var (
	wsCmd = &cobra.Command{
		Use:     "workspace",
		Aliases: []string{"ws"},
		Short:   "Manage your workspaces",
	}
	exploreCmd = &cobra.Command{
		Use:     "explore",
		Aliases: []string{"exp"},
		Run: func(cmd *cobra.Command, args []string) {
			s := mustStore()
			ctx, err := actions.ReadWorkspaceContext(s)
			if err != nil {
				log.Fatalln(err)
			}
			r, err := forms.FindWorkspace(ctx.Workspaces)
			if err != nil {
				log.Fatalln(err)
			}
			ws := path.Join(ctx.Root, r.Name)
			workspace.Explore(ws)
		},
	}
	openWsCmd = &cobra.Command{
		Use:     "open",
		Aliases: []string{"o"},
		Run: func(cmd *cobra.Command, args []string) {
			s := mustStore()
			name, wsPath, _, err := resolveWorkspacePath(s)
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
		Use: "cd",
		Run: func(cmd *cobra.Command, args []string) {
			s := mustStore()
			name, wsPath, _, err := resolveWorkspacePath(s)
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
		Use:     "mux",
		Aliases: []string{"m"},
		Run: func(cmd *cobra.Command, args []string) {
			s := mustStore()
			name, wsPath, _, err := resolveWorkspacePath(s)
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
		Use:     "remove",
		Aliases: []string{"rm"},
		Run: func(cmd *cobra.Command, args []string) {
			s := mustStore()
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
		Use:     "create",
		Aliases: []string{"c"},
		Run: func(cmd *cobra.Command, args []string) {
			s := mustStore()
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
		Use: "clone",
		Run: func(cmd *cobra.Command, args []string) {
			s := mustStore()
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
		Use:     "add",
		Aliases: []string{"a"},
		Run: func(cmd *cobra.Command, args []string) {
			s := mustStore()
			// forms.NewWorkspace needs the repo map; ReadWorkspaceContext
			// only carries Root/Editor/Workspaces, so read the snapshot
			// directly here for the repo list.
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
		Use:     "edit",
		Aliases: []string{"e"},
		Run: func(cmd *cobra.Command, args []string) {
			s := mustStore()
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
			ok, err := forms.CleanWorkspace()
			if err != nil {
				log.Fatalln(err)
			}
			if !ok {
				return
			}
			if err := workspace.Clean(ctx.Root, ctx.Workspaces); err != nil {
				log.Fatalln(err)
			}
		},
	}
	postInstallScriptWsCmd = &cobra.Command{
		Use:     "post-install-script",
		Aliases: []string{"p"},
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			s := mustStore()
			wsMap, postInstall, err := actions.ListWorkspaces(s)
			if err != nil {
				log.Fatalln(err)
			}
			ws, err := forms.FindWorkspace(wsMap)
			if err != nil {
				log.Fatalln(err)
			}
			selected := postInstall[ws.Name]

			scriptList, err := scripts.Default().ListScripts()
			if err != nil {
				log.Fatalln(err)
			}
			updated, err := forms.SelectScripts(scriptList, selected)
			if err != nil {
				log.Fatalln(err)
			}
			if err := actions.SetWorkspacePostInstall(s, ws.Name, updated); err != nil {
				log.Fatalln(err)
			}
		},
	}
)

func init() {
	wsCmd.AddCommand(addWsCmd, listWsCmd, createWsCmd, cloneWsCmd, editWsCmd, removeWsCmd, cdWsCmd, openWsCmd, cleanWSCmd, tmuxWsCmd, postInstallScriptWsCmd, exploreCmd)
}
