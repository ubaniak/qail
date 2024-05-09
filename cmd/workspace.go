package cmd

import (
	"fmt"
	"log"
	"os"
	"path"
	"qail/internal/config"
	forms "qail/internal/forms"
	"qail/internal/workspace"

	"github.com/spf13/cobra"
)

var (
	wsCmd = &cobra.Command{
		Use:     "workspace",
		Aliases: []string{"ws"},
		Short:   "Manage your workspaces",
		Run:     runWsCmd(),
	}
	openWsCmd = &cobra.Command{
		Use:     "open",
		Aliases: []string{"o"},
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.ReadFromFile()
			if err != nil {
				log.Fatalln(err)
			}

			r, err := forms.FindWorkspace(cfg.Workspaces)
			if err != nil {
				log.Fatalln(err)
			}

			ws := path.Join(cfg.Root, r.Name)

			if _, err := os.Stat(ws); os.IsNotExist(err) {
				log.Fatalln(fmt.Errorf("workspace \"%s\" does not exist. Please run qail ws create", ws))
			}

			workspace.Open(cfg.Editor, ws)
		},
	}
	cdWsCmd = &cobra.Command{
		Use: "cd",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.ReadFromFile()
			if err != nil {
				log.Fatalln(err)
			}

			r, err := forms.FindWorkspace(cfg.Workspaces)
			if err != nil {
				log.Fatalln(err)
			}

			ws := path.Join(cfg.Root, r.Name)

			if _, err := os.Stat(ws); os.IsNotExist(err) {
				log.Fatalln(fmt.Errorf("workspace \"%s\" does not exist. Please run qail ws create", ws))
			}
			workspace.Cd(ws)
		},
	}
	removeWsCmd = &cobra.Command{
		Use:     "remove",
		Aliases: []string{"rm"},
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.ReadFromFile()
			if err != nil {
				log.Fatalln(err)
			}

			err = forms.RemoveWorkspace(&cfg.Workspaces)
			if err != nil {
				log.Fatalln(err)
			}

			err = config.WriteToFile(cfg)
			if err != nil {
				log.Fatalln(err)
			}
		},
	}
	listWsCmd = &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.ReadFromFile()
			if err != nil {
				log.Fatalln(err)
			}

			forms.DisplayWorkspaces((cfg.Workspaces))
		},
	}
	createWsCmd = &cobra.Command{
		Use:     "create",
		Aliases: []string{"c"},
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.ReadFromFile()
			if err != nil {
				log.Fatalln(err)
			}

			r, err := forms.FindWorkspace(cfg.Workspaces)
			if err != nil {
				log.Fatalln(err)
			}

			ws := workspace.New(cfg.Root, r.Name, r.Packages, cfg.Repos)
			ws.Create()
		},
	}
	cloneWsCmd = &cobra.Command{
		Use: "clone",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.ReadFromFile()
			if err != nil {
				log.Fatalln(err)
			}

			f, err := forms.FindWorkspace(cfg.Workspaces)
			if err != nil {
				log.Fatalln(err)
			}

			c, err := forms.CloneWorkspace(f.Name, f.Packages)
			if err != nil {
				log.Fatalln(err)
			}

			cfg.Workspaces[c.Name] = c.Packages

			err = config.WriteToFile(cfg)
			if err != nil {
				log.Fatalln(err)
			}

			ws := workspace.New(cfg.Root, c.Name, c.Packages, cfg.Repos)
			ws.Create()
		},
	}
	addWsCmd = &cobra.Command{
		Use:     "add",
		Aliases: []string{"a"},
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.ReadFromFile()
			if err != nil {
				log.Fatalln(err)
			}
			if cfg.Workspaces == nil {
				cfg.Workspaces = make(map[string][]string)
			}

			r, err := forms.NewWorkspace(cfg.Repos)
			if err != nil {
				log.Fatalln(err)
			}

			cfg.Workspaces[r.Name] = r.Packages

			err = config.WriteToFile(cfg)
			if err != nil {
				log.Fatalln(err)
			}

			ws := workspace.New(cfg.Root, r.Name, r.Packages, cfg.Repos)
			ws.Create()
		},
	}
	editWsCmd = &cobra.Command{
		Use:     "edit",
		Aliases: []string{"e"},
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.ReadFromFile()
			if err != nil {
				log.Fatalln(err)
			}
			if cfg.Workspaces == nil {
				cfg.Workspaces = make(map[string][]string)
			}

			r, err := forms.FindWorkspace(cfg.Workspaces)
			if err != nil {
				log.Fatalln(err)
			}

			e, err := forms.EditWorkspace(r.Name, r.Packages, cfg.Repos)
			if err != nil {
				log.Fatalln(err)
			}

			cfg.Workspaces[e.Name] = e.Packages

			err = config.WriteToFile(cfg)
			if err != nil {
				log.Fatalln(err)
			}

			ws := workspace.New(cfg.Root, e.Name, e.Packages, cfg.Repos)
			ws.Create()
		},
	}
)

func runWsCmd() cobraReturnType {
	return func(cmd *cobra.Command, args []string) {
		for _, arg := range args {
			switch arg {
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
			}

		}
	}
}

func init() {
	wsCmd.AddCommand(addWsCmd, listWsCmd, createWsCmd, cloneWsCmd, editWsCmd, removeWsCmd, cdWsCmd, openWsCmd)
}