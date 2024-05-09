package forms

import (
	"github.com/charmbracelet/huh/spinner"
)

func Spinner(action func(), message string) error {
	return spinner.New().Title(message).Action(action).Run()
}
