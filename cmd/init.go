package cmd

import (
	"fmt"
	"log"
	"qail/internal/config"
	forms "qail/internal/forms"

	"github.com/spf13/cobra"
)

var (
	initCmd = &cobra.Command{
		Use:   "init",
		Short: "sets the root folder to the default path",
		Run:   runInitCmd(),
	}
)

func runInitCmd() cobraReturnType {
	return func(cmd *cobra.Command, args []string) {

		cfg, err := config.ReadFromFile()
		if err != nil {
			log.Fatalln(err)
		}

		r, err := forms.Init()
		if err != nil {
			log.Fatalln(err)
		}

		cfg.Root = r.Root
		fmt.Printf("Setting root folder to %s\n", cfg.Root)

		err = config.WriteToFile(cfg)
		if err != nil {
			log.Fatalln(err)
		}
	}
}

func init() {
	// initCmd.AddCommand(configRootCmd, configEditorCmd)
}
