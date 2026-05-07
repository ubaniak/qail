package cmd

import (
	"fmt"
	"log"
	"slices"

	"github.com/spf13/cobra"

	"github.com/ubaniak/qail/internal/forms"
	"github.com/ubaniak/qail/internal/tmux"
)

// sessionExists reports whether the named tmux session is in the live
// session list, so non-interactive `mux rm <name>` fails fast on typos.
func sessionExists(t *tmux.Tmux, name string) (bool, error) {
	all, err := t.ListSessions()
	if err != nil {
		return false, err
	}
	return slices.Contains(all, name), nil
}

var (
	tmuxCmd = &cobra.Command{
		Use:     "mux",
		Short:   "manage tmux",
		Aliases: []string{"m"},
	}
	lsTmuxCmd = &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Run: func(cmd *cobra.Command, args []string) {
			sessions, err := tmux.Default().ListSessions()
			if err != nil {
				log.Fatalln(err)
			}
			forms.DisplayTmuxSessions(sessions)
		},
	}
	rmTmuxCmd = &cobra.Command{
		Use:     "remove [name]",
		Aliases: []string{"rm"},
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			t := tmux.Default()

			if name := firstArg(args); name != "" {
				ok, err := sessionExists(t, name)
				if err != nil {
					log.Fatalln(err)
				}
				if !ok {
					log.Fatalf("tmux session %q not found\n", name)
				}
				if !flagYes {
					if err := requireTTY("--yes for non-interactive remove"); err != nil {
						log.Fatalln(err)
					}
					confirm, err := forms.Confirm(fmt.Sprintf("Remove tmux session %q?", name))
					if err != nil {
						log.Fatalln(err)
					}
					if !confirm {
						return
					}
				}
				if err := t.RemoveSession(name); err != nil {
					log.Fatalln(err)
				}
				return
			}

			if err := requireTTY("tmux session name"); err != nil {
				log.Fatalln(err)
			}
			sessions, err := t.ListSessions()
			if err != nil {
				log.Fatalln(err)
			}
			s, ok, err := forms.RemoveTmuxSession(sessions)
			if err != nil {
				log.Fatalln(err)
			}
			if !ok {
				return
			}
			if err := t.RemoveSession(s); err != nil {
				log.Fatalln(err)
			}
		},
	}
)

func init() {
	tmuxCmd.AddCommand(lsTmuxCmd, rmTmuxCmd)
}
