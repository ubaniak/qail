package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/ubaniak/qail/internal/actions"
	"github.com/ubaniak/qail/internal/forms"
)

var (
	initCmd = &cobra.Command{
		Use:   "init",
		Short: "sets the root folder to the default path",
		Run: func(cmd *cobra.Command, args []string) {
			r, err := forms.Init()
			if err != nil {
				log.Fatalln(err)
			}
			fmt.Printf("Setting root folder to %s\n", r.Root)
			if err := actions.SetRoot(mustStore(), r.Root); err != nil {
				log.Fatalln(err)
			}
		},
	}
)
