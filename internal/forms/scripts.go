package forms

import (
	"sort"

	"github.com/charmbracelet/huh"
)

func NewScript() (string, error) {
	var name string

	g := huh.NewGroup(
		huh.NewInput().Title("Script name").Value(&name),
	)

	f := huh.NewForm(g)
	err := f.Run()
	return name, err
}

func DisplayScripts(scripts []string) {

	var rows [][]string
	sort.Strings(scripts)
	for _, s := range scripts {
		row := []string{s}
		rows = append(rows, row)
	}

	headers := []string{"Name"}

	displayTable(headers, rows)
}

func SelectScript(scripts []string) (string, error) {

	var script string
	s := huh.NewSelect[string]().Title("Choose a script").Value(&script)

	var opts []huh.Option[string]
	for _, s := range scripts {
		opts = append(opts, huh.NewOption(s, s))
	}
	s.Options(opts...)

	f := huh.NewForm(
		huh.NewGroup(s),
	)
	err := f.Run()
	return script, err
}

func SelectScripts(scripts, selected []string) ([]string, error) {

	s := huh.NewMultiSelect[string]().Title("Select scripts").Value(&selected)

	var opts []huh.Option[string]
	for _, s := range scripts {
		opts = append(opts, huh.NewOption(s, s))
	}
	s.Options(opts...)

	f := huh.NewForm(
		huh.NewGroup(s),
	)
	err := f.Run()
	return selected, err
}
