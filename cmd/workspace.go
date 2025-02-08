package cmd

import (
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/spf13/cobra"

	"qail/internal/config"
	forms "qail/internal/forms"
	"qail/internal/scripts"
	"qail/internal/workspace"
)

var (
	wsCmd = &cobra.Command{
		Use:     "workspace",
		Aliases: []string{"ws"},
		Short:   "Manage your workspaces",
		Run:     runWsCmd(),
	}
	exploreCmd = &cobra.Command{
		Use:     "explore",
		Aliases: []string{"exp"},
		Run: func(cmd *cobra.Command, args []string) {
			fn := func(cfg *config.Config) error {
				r, err := forms.FindWorkspace(cfg.Workspaces)
				if err != nil {
					return err
				}

				ws := path.Join(cfg.Root, r.Name)

				workspace.Explore(ws)
				return nil
			}
			HandleConfig(fn)
		},
	}
	openWsCmd = &cobra.Command{
		Use:     "open",
		Aliases: []string{"o"},
		Run: func(cmd *cobra.Command, args []string) {
			fn := func(cfg *config.Config) error {
				r, err := forms.FindWorkspace(cfg.Workspaces)
				if err != nil {
					return err
				}

				ws := path.Join(cfg.Root, r.Name)

				if _, err := os.Stat(ws); os.IsNotExist(err) {
					return fmt.Errorf("workspace \"%s\" does not exist. Please run qail ws create", ws)
				}

				cfg.Workspaces[r.Name] = config.NewWorkspaceProfile(r.Packages, time.Now().UTC())

				workspace.Open(cfg.Editor, ws)
				return nil
			}
			HandleConfig(fn)
		},
	}
	cdWsCmd = &cobra.Command{
		Use: "cd",
		Run: func(cmd *cobra.Command, args []string) {
			fn := func(cfg *config.Config) error {

				r, err := forms.FindWorkspace(cfg.Workspaces)
				if err != nil {
					return err
				}

				ws := path.Join(cfg.Root, r.Name)

				if _, err := os.Stat(ws); os.IsNotExist(err) {
					return fmt.Errorf("workspace \"%s\" does not exist. Please run qail ws create", ws)
				}

				cfg.Workspaces[r.Name] = config.NewWorkspaceProfile(r.Packages, time.Now().UTC())
				workspace.Cd(ws)
				return nil
			}

			HandleConfig(fn)
		},
	}
	tmuxWsCmd = &cobra.Command{
		Use:     "mux",
		Aliases: []string{"m"},
		Run: func(cmd *cobra.Command, args []string) {
			fn := func(cfg *config.Config) error {
				r, err := forms.FindWorkspace(cfg.Workspaces)
				if err != nil {
					return err
				}

				ws := path.Join(cfg.Root, r.Name)

				if _, err := os.Stat(ws); os.IsNotExist(err) {
					return fmt.Errorf("workspace \"%s\" does not exist. Please run qail ws create", ws)
				}

				cfg.Workspaces[r.Name] = config.NewWorkspaceProfile(r.Packages, time.Now().UTC())
				err = config.WriteToFile(*cfg)
				if err != nil {
					return err
				}

				err = workspace.Tmux(ws)
				if err != nil {
					return err
				}
				return nil
			}

			HandleConfig(fn)
		},
	}
	removeWsCmd = &cobra.Command{
		Use:     "remove",
		Aliases: []string{"rm"},
		Run: func(cmd *cobra.Command, args []string) {
			fn := func(cfg *config.Config) error {
				err := forms.RemoveWorkspace(&cfg.Workspaces)
				if err != nil {
					return err
				}

				return nil
			}
			HandleConfig(fn)
		},
	}
	listWsCmd = &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fn := func(cfg *config.Config) error {
				forms.DisplayWorkspaces(cfg.Workspaces, cfg.PostInstallScripts.Workspace)
				return nil
			}

			HandleConfig(fn)

		},
	}
	createWsCmd = &cobra.Command{
		Use:     "create",
		Aliases: []string{"c"},
		Run: func(cmd *cobra.Command, args []string) {
			fn := func(cfg *config.Config) error {
				r, err := forms.FindWorkspace(cfg.Workspaces)
				if err != nil {
					return err
				}

				ws := workspace.New(cfg.Root, r.Name, r.Packages, cfg.Repos)
				ws.WithRepoPostInstallScripts(cfg.PostInstallScripts.Repo)
				ws.WithWSPostInstallScripts(cfg.PostInstallScripts.Workspace)
				return ws.Create()
			}
			HandleConfig(fn)
		},
	}
	cloneWsCmd = &cobra.Command{
		Use: "clone",
		Run: func(cmd *cobra.Command, args []string) {
			fn := func(cfg *config.Config) error {

				f, err := forms.FindWorkspace(cfg.Workspaces)
				if err != nil {
					return err
				}

				c, err := forms.CloneWorkspace(f.Name, f.Packages)
				if err != nil {
					return err
				}

				cfg.Workspaces[c.Name] = config.NewWorkspaceProfile(c.Packages, c.LastUsed)

				ws := workspace.New(cfg.Root, c.Name, c.Packages, cfg.Repos)
				ws.WithRepoPostInstallScripts(cfg.PostInstallScripts.Repo)
				ws.WithWSPostInstallScripts(cfg.PostInstallScripts.Workspace)
				return ws.Create()
			}

			err := config.WithConfig(fn)
			if err != nil {
				log.Fatalln(err)
			}
		},
	}
	addWsCmd = &cobra.Command{
		Use:     "add",
		Aliases: []string{"a"},
		Run: func(cmd *cobra.Command, args []string) {
			fn := func(cfg *config.Config) error {
				if cfg.Workspaces == nil {
					cfg.Workspaces = make(config.Workspace)
				}

				r, err := forms.NewWorkspace(cfg.Repos)
				if err != nil {
					return err
				}

				cfg.Workspaces[r.Name] = config.NewWorkspaceProfile(r.Packages, r.LastUsed)

				ws := workspace.New(cfg.Root, r.Name, r.Packages, cfg.Repos)
				ws.WithRepoPostInstallScripts(cfg.PostInstallScripts.Repo)
				ws.WithWSPostInstallScripts(cfg.PostInstallScripts.Workspace)
				return ws.Create()
			}

			err := config.WithConfig(fn)
			if err != nil {
				log.Fatalln(err)
			}
		},
	}
	editWsCmd = &cobra.Command{
		Use:     "edit",
		Aliases: []string{"e"},
		Run: func(cmd *cobra.Command, args []string) {

			fn := func(cfg *config.Config) error {
				if cfg.Workspaces == nil {
					cfg.Workspaces = make(config.Workspace)
				}

				r, err := forms.FindWorkspace(cfg.Workspaces)
				if err != nil {
					return err
				}

				e, err := forms.EditWorkspace(r.Name, r.Packages, cfg.Repos)
				if err != nil {
					return err
				}

				cfg.Workspaces[e.Name] = config.NewWorkspaceProfile(e.Packages, e.LastUsed)

				ws := workspace.New(cfg.Root, e.Name, e.Packages, cfg.Repos)
				ws.WithRepoPostInstallScripts(cfg.PostInstallScripts.Repo)
				ws.WithWSPostInstallScripts(cfg.PostInstallScripts.Workspace)
				return ws.Create()
			}

			err := config.WithConfig(fn)
			if err != nil {
				log.Fatalln(err)
			}
		},
	}
	cleanWSCmd = &cobra.Command{
		Use: "clean",
		Run: func(cmd *cobra.Command, args []string) {

			fn := func(cfg *config.Config) error {

				ok, err := forms.CleanWorkspace()
				if err != nil {
					return err
				}

				if !ok {
					return nil
				}

				return workspace.Clean(cfg.Root, cfg.Workspaces)
			}

			HandleConfig(fn)
		},
	}
	postInstallScriptWsCmd = &cobra.Command{
		Use:     "post-install-script",
		Aliases: []string{"p"},
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fn := func(cfg *config.Config) error {
				if cfg.PostInstallScripts.Workspace == nil {
					cfg.PostInstallScripts.Workspace = make(map[string][]string)
				}

				ws, err := forms.FindWorkspace(cfg.Workspaces)
				if err != nil {
					return err
				}

				var selected []string
				if _, ok := cfg.PostInstallScripts.Workspace[ws.Name]; !ok {
					cfg.PostInstallScripts.Workspace[ws.Name] = []string{}
				}

				selected = cfg.PostInstallScripts.Workspace[ws.Name]

				scripts, err := scripts.ListScripts()
				if err != nil {
					return nil
				}

				updatedScripts, err := forms.SelectScripts(scripts, selected)

				if err != nil {
					return err
				}

				cfg.PostInstallScripts.Workspace[ws.Name] = updatedScripts

				return nil
			}
			HandleConfig(fn)
		},
	}
)

func runWsCmd() cobraReturnType {
	return func(cmd *cobra.Command, args []string) {
		for _, arg := range args {
			switch arg {
			case "tmux":
				tmuxWsCmd.Execute()
			case "create":
				createWsCmd.Execute()
			case "add":
				addWsCmd.Execute()
			case "clone":
				cloneWsCmd.Execute()
			case "edit":
				editWsCmd.Execute()
			case "remove":
				removeWsCmd.Execute()
			case "cd":
				cdWsCmd.Execute()
			case "open":
				openWsCmd.Execute()
			case "clean":
				cleanWSCmd.Execute()
			case "explore":
				exploreCmd.Execute()
			case "post-install-scripts":
				postInstallScriptWsCmd.Execute()
			}

		}
	}
}

func init() {
	wsCmd.AddCommand(addWsCmd, listWsCmd, createWsCmd, cloneWsCmd, editWsCmd, removeWsCmd, cdWsCmd, openWsCmd, cleanWSCmd, tmuxWsCmd, postInstallScriptWsCmd, exploreCmd)
}
