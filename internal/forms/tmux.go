package forms

import (
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

// SelectTmuxSession prompts the user to pick a tmux session and returns
// the chosen name. Pure selector — confirmation is the caller's job
// (chain with forms.Confirm).
func SelectTmuxSession(sessions []string) (string, error) {
	var name string
	s := huh.NewSelect[string]().Title("Choose a session").Value(&name)

	var opts []huh.Option[string]
	for _, s := range sessions {
		opts = append(opts, huh.NewOption(s, s))
	}
	s.Options(opts...)

	f := huh.NewForm(huh.NewGroup(s))
	if err := f.Run(); err != nil {
		return "", err
	}
	return name, nil
}
