package forms

import "github.com/charmbracelet/huh"

func Confirm(msg string) (bool, error) {
	var confirm bool
	c := huh.NewConfirm().
		Title(msg).
		Affirmative("Yes").
		Negative("No").
		Value(&confirm)

	f := huh.NewForm(
		huh.NewGroup(c),
	)
	err := f.Run()

	return confirm, err
}
