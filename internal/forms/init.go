package forms

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
)

type initModel struct {
	Root string
}

func Init() (initModel, error) {
	var root string
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return initModel{}, err
	}

	root = filepath.Join(homeDir, "workspaces")
	g := huh.NewGroup(
		huh.NewInput().Title("Set the workspace root").Value(&root),
	)

	form := huh.NewForm(g)

	err = form.Run()

	if err != nil {
		return initModel{}, err
	}

	return initModel{
		Root: root,
	}, nil
}
