package cmd

import (
	"fmt"
	"log"
	"qail/internal/config"

	forms "qail/internal/forms"

	"github.com/spf13/cobra"
)

var (
	repoCmd = &cobra.Command{
		Use:     "repo",
		Short:   "manage your workspace repos",
		Aliases: []string{"r"},
		Run:     runRepoCmd(),
	}
	rmRepoCmd = &cobra.Command{
		Use:     "remove",
		Aliases: []string{"rm"},
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.ReadFromFile()
			if err != nil {
				log.Fatalln(err)
			}

			err = forms.RemoveRepo(&cfg.Repos)
			if err != nil {
				log.Fatalln(err)
			}

			if cfg.Repos == nil {
				cfg.Repos = make(map[string]string)
			}

			err = config.WriteToFile(cfg)
			if err != nil {
				log.Fatalln(err)
			}
		},
	}
	addRepoCmd = &cobra.Command{
		Use:     "add",
		Aliases: []string{"a"},
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.ReadFromFile()
			if err != nil {
				log.Fatalln(err)
			}

			r, err := forms.AddRepo()
			if err != nil {
				log.Fatalln(err)
			}

			if cfg.Repos == nil {
				cfg.Repos = make(map[string]string)
			}
			cfg.Repos[r.Name] = r.Repo

			err = config.WriteToFile(cfg)
			if err != nil {
				log.Fatalln(err)
			}
		},
	}
	listRepoCmd = &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.ReadFromFile()
			if err != nil {
				log.Fatalln(err)
			}
			if cfg.Repos == nil {
				fmt.Println("No packages found. Please add a package")
			}
			forms.DisplayRepos(cfg.Repos)
		},
	}
)

func runRepoCmd() cobraReturnType {
	return func(cmd *cobra.Command, args []string) {

		for _, arg := range args {
			switch arg {
			case "Add":
				addRepoCmd.Execute()
			case "list":
				listRepoCmd.Execute()
			case "remove":
				rmRepoCmd.Execute()
			}
		}

	}
}

func init() {
	repoCmd.AddCommand(addRepoCmd, listRepoCmd, rmRepoCmd)
}
