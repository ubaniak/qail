package forms

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
)

type repoModel struct {
	Repo string
	Name string
}

func getRepo() (string, error) {
	var repo string
	g := huh.NewGroup(
		huh.NewInput().Title("Github repo path").Value(&repo).Validate(
			func(str string) error {
				return nil
			},
		),
	)

	f := huh.NewForm(g)
	err := f.Run()
	if err != nil {
		return "", err
	}
	return repo, nil
}

func getName(n string) (string, error) {
	name := n
	g := huh.NewGroup(
		huh.NewInput().Title("Give the repo a name").Value(&name),
	)

	f := huh.NewForm(g)
	err := f.Run()

	if err != nil {
		return "", err
	}

	return name, nil
}

func AddRepo() (repoModel, error) {

	repo, err := getRepo()
	if err != nil {
		return repoModel{}, err
	}

	defaultName := ""
	p := strings.Split(repo, "/")

	if len(p) > 0 {
		defaultName = p[len(p)-1]
	}

	name, err := getName(defaultName)
	if err != nil {
		return repoModel{}, err
	}

	return repoModel{
		Repo: repo,
		Name: name,
	}, nil
}

func RemoveRepo(repos *map[string]string) error {

	var toRemove []string
	s := huh.NewMultiSelect[string]().Value(&toRemove)

	var opts []huh.Option[string]
	for k, v := range *repos {
		fmtStr := fmt.Sprintf("%s: %s", k, v)
		opts = append(opts, huh.NewOption(fmtStr, k))
	}

	s.Options(opts...)

	var confirm bool
	c := huh.NewConfirm().
		Title("This will remove the selected repos. Are you sure?").
		Affirmative("Yes").
		Negative("No").
		Value(&confirm)

	f := huh.NewForm(
		huh.NewGroup(s),
		huh.NewGroup(c),
	)
	err := f.Run()
	if err != nil {
		return err
	}

	if !confirm {
		return nil
	}

	for _, r := range toRemove {
		delete(*repos, r)
	}

	return nil
}

func DisplayRepos(p map[string]string) {
	headers := []string{"Name", "Repo"}
	var rows [][]string
	for k, v := range p {
		row := []string{k, v}
		rows = append(rows, row)
	}

	displayTable(headers, rows)
}
