package cmd

import (
	"errors"
	"log"

	"github.com/spf13/cobra"

	"qail/internal/config"
	forms "qail/internal/forms"
	"qail/internal/scripts"
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
					return err
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
					return errors.New("no packages found. Please add a package")
				}
				forms.DisplayRepos(cfg.Repos, cfg.PostInstallScripts.Repo)
				return nil
			}
			HandleConfig(fn)
		},
	}
	postInstallScriptRepoCmd = &cobra.Command{
		Use:     "post-install-script",
		Aliases: []string{"p"},
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fn := func(cfg *config.Config) error {
				if cfg.PostInstallScripts.Repo == nil {
					cfg.PostInstallScripts.Repo = make(map[string][]string)
				}

				r, err := forms.SelectRepo(&cfg.Repos)
				if err != nil {
					return err
				}

				var selected []string
				if _, ok := cfg.PostInstallScripts.Repo[r]; !ok {
					cfg.PostInstallScripts.Repo[r] = []string{}
				}

				selected = cfg.PostInstallScripts.Repo[r]

				scripts, err := scripts.ListScripts()
				if err != nil {
					return nil
				}

				updatedScripts, err := forms.SelectScripts(scripts, selected)

				if err != nil {
					return err
				}

				cfg.PostInstallScripts.Repo[r] = updatedScripts

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
			case "post-install-script":
				postInstallScriptRepoCmd.Execute()
			}
		}

	}
}

func init() {
	repoCmd.AddCommand(addRepoCmd, listRepoCmd, rmRepoCmd, postInstallScriptRepoCmd)
}
