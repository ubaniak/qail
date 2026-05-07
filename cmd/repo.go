package cmd

import (
	"errors"
	"log"

	"github.com/spf13/cobra"

	"github.com/ubaniak/qail/internal/actions"
	"github.com/ubaniak/qail/internal/config"
	"github.com/ubaniak/qail/internal/forms"
	"github.com/ubaniak/qail/internal/scripts"
)

var (
	repoCmd = &cobra.Command{
		Use:     "repo",
		Short:   "manage your workspace repos",
		Aliases: []string{"r"},
	}
	rmRepoCmd = &cobra.Command{
		Use:     "remove",
		Aliases: []string{"rm"},
		Run: func(cmd *cobra.Command, args []string) {
			s := mustStore()
			repos, _, err := actions.ListRepos(s)
			if err != nil {
				log.Fatalln(err)
			}

			toRemove, confirmed, err := forms.SelectReposToRemove(repos)
			if err != nil {
				log.Fatalln(err)
			}
			if !confirmed {
				return
			}

			if err := actions.RemoveRepos(s, toRemove); err != nil {
				log.Fatalln(err)
			}
		},
	}
	addRepoCmd = &cobra.Command{
		Use:     "add",
		Aliases: []string{"a"},
		Run: func(cmd *cobra.Command, args []string) {
			r, err := forms.AddRepo()
			if err != nil {
				log.Fatalln(err)
			}
			if err := actions.AddRepo(mustStore(), r.Name, r.Repo); err != nil {
				log.Fatalln(err)
			}
		},
	}
	listRepoCmd = &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			repos, postInstall, err := actions.ListRepos(mustStore())
			if err != nil {
				log.Fatalln(err)
			}
			if repos == nil {
				log.Fatalln(errors.New("no packages found. Please add a package"))
			}
			forms.DisplayRepos(repos, postInstall)
		},
	}
	postInstallScriptRepoCmd = &cobra.Command{
		Use:     "post-install-script",
		Aliases: []string{"p"},
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			s := mustStore()
			repos, postInstall, err := actions.ListRepos(s)
			if err != nil {
				log.Fatalln(err)
			}

			r, err := forms.SelectRepo(&repos)
			if err != nil {
				log.Fatalln(err)
			}

			selected := postInstall[r]

			scriptList, err := scripts.Default().ListScripts()
			if err != nil {
				log.Fatalln(err)
			}

			updatedScripts, err := forms.SelectScripts(scriptList, selected)
			if err != nil {
				log.Fatalln(err)
			}

			if err := actions.SetRepoPostInstall(s, r, updatedScripts); err != nil {
				log.Fatalln(err)
			}
		},
	}
)

// mustStore returns the production config.Store or fatally exits.
func mustStore() config.Store {
	s, err := config.DefaultStore()
	if err != nil {
		log.Fatalln(err)
	}
	return s
}

func init() {
	repoCmd.AddCommand(addRepoCmd, listRepoCmd, rmRepoCmd, postInstallScriptRepoCmd)
}
