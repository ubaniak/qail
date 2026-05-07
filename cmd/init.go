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
		Use:   "init [root]",
		Short: "sets the root folder to the default path",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			var root string
			if len(args) == 1 {
				root = args[0]
			} else {
				if err := requireTTY("workspace root path"); err != nil {
					log.Fatalln(err)
				}
				r, err := forms.Init()
				if err != nil {
					log.Fatalln(err)
				}
				root = r.Root
			}
			fmt.Printf("Setting root folder to %s\n", root)
			if err := actions.SetRoot(mustStore(), root); err != nil {
				log.Fatalln(err)
			}
		},
	}
)
