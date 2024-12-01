package forms

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

func DisplayTmuxSessions(sessions []string) {

	headers := []string{"Name"}
	var rows [][]string
	for _, s := range sessions {
		row := []string{s}
		rows = append(rows, row)
	}

	displayTable(headers, rows)
}

func RemoveTmuxSession(sessions []string) (string, bool, error) {
	var name string
	s := huh.NewSelect[string]().Title("Choose a session").Value(&name)

	var opts []huh.Option[string]
	for _, s := range sessions {
		fmt.Println(s)
		opts = append(opts, huh.NewOption(s, s))
	}
	s.Options(opts...)

	var confirm bool
	c := huh.NewConfirm().
		Title("This will remove the selected session. Are you sure?").
		Affirmative("Yes").
		Negative("No").
		Value(&confirm)

	f := huh.NewForm(
		huh.NewGroup(s),
		huh.NewGroup(c),
	)
	err := f.Run()
	return name, confirm, err
}
