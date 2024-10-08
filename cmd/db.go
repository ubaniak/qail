package cmd

import (
	"qail/internal/db"

	"github.com/spf13/cobra"
)

var (
	dbCmd = &cobra.Command{
		Use:   "db",
		Short: "sets the root folder to the default path",
		Run:   runDbCmd(),
	}
)

func runDbCmd() cobraReturnType {
	return func(cmd *cobra.Command, args []string) {
		dbPath := "./test.db"
		driver, err := db.CreateSqliteDb(dbPath)
		if err != nil {
			panic(err)
		}

		d := db.New(driver)

		d.SetupDB()

		c := db.Config{
			Editor:    "bl",
			Workspace: "s",
		}

		d.NewConfig(&c)
	}
}

func init() {
	// initCmd.AddCommand(configRootCmd, configEditorCmd)
}
