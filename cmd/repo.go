package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"qail/internal/config"
	forms "qail/internal/forms"
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
			fn := func(cfg *config.Config) error {
				err := forms.RemoveRepo(&cfg.Repos)
				if err != nil {
					log.Fatalln(err)
				}

				if cfg.Repos == nil {
					cfg.Repos = make(map[string]string)
				}
				return nil
			}

			HandleConfig(fn)
		},
	}
	addRepoCmd = &cobra.Command{
		Use:     "add",
		Aliases: []string{"a"},
		Run: func(cmd *cobra.Command, args []string) {
			fn := func(cfg *config.Config) error {
				r, err := forms.AddRepo()
				if err != nil {
					log.Fatalln(err)
				}

				if cfg.Repos == nil {
					cfg.Repos = make(map[string]string)
				}
				cfg.Repos[r.Name] = r.Repo
				return nil
			}

			HandleConfig(fn)
		},
	}
	listRepoCmd = &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fn := func(cfg *config.Config) error {
				if cfg.Repos == nil {
					fmt.Println("No packages found. Please add a package")
				}
				forms.DisplayRepos(cfg.Repos)
				return nil
			}
			HandleConfig(fn)
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
