package cmd

import (
	"github.com/spf13/cobra"

	"github.com/ubaniak/qail/internal/config"
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
			fn := func(cfg *config.Config) error {
				sessions, err := tmux.ListSessions()
				if err != nil {
					return err
				}
				forms.DisplayTmuxSessions(sessions)
				return nil
			}

			HandleConfig(fn)
		},
	}
	rmTmuxCmd = &cobra.Command{
		Use:     "remove",
		Aliases: []string{"rm"},
		Run: func(cmd *cobra.Command, args []string) {
			fn := func(cfg *config.Config) error {
				sessions, err := tmux.ListSessions()
				if err != nil {
					return err
				}
				s, ok, err := forms.RemoveTmuxSession(sessions)
				if !ok {
					return nil
				}
				if err != nil {
					return err
				}

				return tmux.RemoveSession(s)
			}

			HandleConfig(fn)
		},
	}
)

func init() {
	tmuxCmd.AddCommand(lsTmuxCmd, rmTmuxCmd)
}
