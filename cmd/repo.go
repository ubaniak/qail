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
	// flag-backed inputs for non-interactive repo commands. Each var is
	// reset by cobra between runs of the same process; tests should not
	// rely on default values surviving across invocations.
	addRepoName    string
	addRepoURL     string
	piRepoScripts  []string
	piRepoClear    bool

	repoCmd = &cobra.Command{
		Use:     "repo",
		Short:   "manage your workspace repos",
		Aliases: []string{"r"},
	}
	rmRepoCmd = &cobra.Command{
		Use:     "remove [name...]",
		Aliases: []string{"rm"},
		Short:   "remove repos by name; with no args, opens a multi-select TUI",
		Run: func(cmd *cobra.Command, args []string) {
			s := mustStore()

			// Non-interactive path: names supplied as positional args.
			// --yes skips the confirm prompt; without it we still ask
			// once so a typo doesn't silently nuke a repo.
			if len(args) > 0 {
				ok, err := confirmOrSkip("Remove the listed repos?", "--yes for non-interactive remove")
				if err != nil {
					log.Fatalln(err)
				}
				if !ok {
					return
				}
				if err := actions.RemoveRepos(s, args); err != nil {
					log.Fatalln(err)
				}
				return
			}

			if err := requireTTY("repo names as positional args"); err != nil {
				log.Fatalln(err)
			}
			repos, _, err := actions.ListRepos(s)
			if err != nil {
				log.Fatalln(err)
			}

			toRemove, err := forms.SelectRepos(repos)
			if err != nil {
				log.Fatalln(err)
			}
			confirmed, err := forms.Confirm("Remove the selected repos?")
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
		Use:     "add [name] [url]",
		Aliases: []string{"a"},
		Short:   "add a repo; pass name+url for non-interactive use",
		Args:    cobra.MaximumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			name, url := addRepoName, addRepoURL
			if len(args) >= 1 && name == "" {
				name = args[0]
			}
			if len(args) >= 2 && url == "" {
				url = args[1]
			}

			if name == "" || url == "" {
				if err := requireTTY("--name and --url (or positional name+url)"); err != nil {
					log.Fatalln(err)
				}
				r, err := forms.AddRepo()
				if err != nil {
					log.Fatalln(err)
				}
				name, url = r.Name, r.Repo
			}

			if err := actions.AddRepo(mustStore(), name, url); err != nil {
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
		Use:     "post-install-script [repo]",
		Aliases: []string{"p"},
		Short:   "set the post-install scripts for a repo",
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			s := mustStore()
			repos, postInstall, err := actions.ListRepos(s)
			if err != nil {
				log.Fatalln(err)
			}

			// Non-interactive path: repo name positional + --script (repeatable)
			// or --clear. --clear wins over --script when both supplied so
			// `--clear --script x` is a no-op clear (intentional: caller is
			// explicit about emptying).
			if len(args) == 1 && (len(piRepoScripts) > 0 || piRepoClear) {
				repo := args[0]
				if _, ok := repos[repo]; !ok {
					log.Fatalf("unknown repo %q\n", repo)
				}
				out := piRepoScripts
				if piRepoClear {
					out = nil
				}
				if err := actions.SetRepoPostInstall(s, repo, out); err != nil {
					log.Fatalln(err)
				}
				return
			}

			if err := requireTTY("repo name + --script/--clear flags"); err != nil {
				log.Fatalln(err)
			}

			var r string
			if len(args) == 1 {
				r = args[0]
				if _, ok := repos[r]; !ok {
					log.Fatalf("unknown repo %q\n", r)
				}
			} else {
				r, err = forms.SelectRepo(&repos)
				if err != nil {
					log.Fatalln(err)
				}
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
	addRepoCmd.Flags().StringVarP(&addRepoName, "name", "n", "", "repo name (skips prompt)")
	addRepoCmd.Flags().StringVarP(&addRepoURL, "url", "u", "", "git url (skips prompt)")

	postInstallScriptRepoCmd.Flags().StringSliceVarP(&piRepoScripts, "script", "s", nil, "post-install script name (repeatable, or comma-separated)")
	postInstallScriptRepoCmd.Flags().BoolVar(&piRepoClear, "clear", false, "clear all post-install scripts for the repo")

	repoCmd.AddCommand(addRepoCmd, listRepoCmd, rmRepoCmd, postInstallScriptRepoCmd)
}
