package cmd

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/ubaniak/qail/internal/forms"
	"github.com/ubaniak/qail/internal/tmux"
)

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
		Use:     "remove",
		Aliases: []string{"rm"},
		Run: func(cmd *cobra.Command, args []string) {
			t := tmux.Default()
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
