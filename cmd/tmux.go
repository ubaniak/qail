package cmd

import (
	"context"
	"fmt"
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
			sessions, err := tmux.Default().ListSessions(context.Background())
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
			ctx := context.Background()

			if name := firstArg(args); name != "" {
				ok, err := t.HasSession(ctx, name)
				if err != nil {
					log.Fatalln(err)
				}
				if !ok {
					log.Fatalf("tmux session %q not found\n", name)
				}
				confirmed, err := confirmOrSkip(fmt.Sprintf("Remove tmux session %q?", name), "--yes for non-interactive remove")
				if err != nil {
					log.Fatalln(err)
				}
				if !confirmed {
					return
				}
				if err := t.RemoveSession(ctx, name); err != nil {
					log.Fatalln(err)
				}
				return
			}

			if err := requireTTY("tmux session name"); err != nil {
				log.Fatalln(err)
			}
			sessions, err := t.ListSessions(ctx)
			if err != nil {
				log.Fatalln(err)
			}
			s, err := forms.SelectTmuxSession(sessions)
			if err != nil {
				log.Fatalln(err)
			}
			confirmed, err := forms.Confirm(fmt.Sprintf("Remove tmux session %q?", s))
			if err != nil {
				log.Fatalln(err)
			}
			if !confirmed {
				return
			}
			if err := t.RemoveSession(ctx, s); err != nil {
				log.Fatalln(err)
			}
		},
	}
)

func init() {
	tmuxCmd.AddCommand(lsTmuxCmd, rmTmuxCmd)
}
