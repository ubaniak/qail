package cmd

import (
	"github.com/spf13/cobra"

	"qail/internal/config"
	"qail/internal/forms"
	"qail/internal/tmux"
)

var (
	tmuxCmd = &cobra.Command{
		Use:     "mux",
		Short:   "manage tmux",
		Aliases: []string{"m"},
		Run:     runTmuxCmd(),
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

func runTmuxCmd() cobraReturnType {
	return func(cmd *cobra.Command, args []string) {

		for _, arg := range args {
			switch arg {
			case "list":
				lsTmuxCmd.Execute()
			case "remove":
				rmTmuxCmd.Execute()
			}
		}

	}
}

func init() {
	tmuxCmd.AddCommand(lsTmuxCmd, rmTmuxCmd)
}
