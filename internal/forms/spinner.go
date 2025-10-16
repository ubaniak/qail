package forms

import (
	"time"

	"github.com/charmbracelet/huh/spinner"
)

type AnimatedSpinner struct {
	Frames []string
	FPS    time.Duration
}

var Train = AnimatedSpinner{
	Frames: []string{"      🚂  ",
		"   🚂 🚃  ",
		"🚂 🚃 🚃  ",
		"🚃 🚃     ",
		"🚃        "},
	FPS: time.Second / 2,
}

func Spinner(action func(), message string) error {
	return spinner.New().Title(message).Type(spinner.Type(Train)).Action(action).Run()
}
