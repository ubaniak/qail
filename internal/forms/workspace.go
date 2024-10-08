package forms

import (
	"fmt"
	"qail/internal/config"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
)

type workspaceModel struct {
	Name     string
	Packages []string
	LastUsed time.Time
}

func SortWorkspaces(ws config.Workspace) []string {

	keys := make([]string, 0, len(ws))
	for key := range ws {
		keys = append(keys, key)
	}

	sort.Slice(keys, func(i, j int) bool {
		return ws[keys[i]].LastUsed.After(ws[keys[j]].LastUsed)
	})

	return keys
}

func formatWorkspaces(ws config.Workspace) ([]string, []string) {
	keys := SortWorkspaces(ws)
	formatted := make([]string, 0, len(ws))
	for _, key := range keys {
		repos := ws[key].Repos
		fmtStr := fmt.Sprintf("%s [%s] %s", key, strings.Join(repos[:], ","), ws[key].LastUsed)
		formatted = append(formatted, fmtStr)
	}
	return keys, formatted
}

func NewWorkspace(allRepos map[string]string) (workspaceModel, error) {
	var name string
	var repos []string

	s := huh.NewMultiSelect[string]().Value(&repos)

	var opts []huh.Option[string]
	for k, v := range allRepos {
		fmtStr := fmt.Sprintf("%s: %s", k, v)
		opts = append(opts, huh.NewOption(fmtStr, k))
	}

	s.Options(opts...)
	g := huh.NewGroup(
		huh.NewInput().Title("Workspace name").Value(&name),
		s,
	)

	f := huh.NewForm(g)
	err := f.Run()
	if err != nil {
		return workspaceModel{}, err
	}
	fmt.Println(repos)

	return workspaceModel{
		Name:     name,
		Packages: repos,
		LastUsed: time.Now().UTC(),
	}, nil
}

func FindWorkspace(ws config.Workspace) (workspaceModel, error) {
	var name string
	s := huh.NewSelect[string]().Title("Choose a workspace").Value(&name)

	var opts []huh.Option[string]
	keys, fmt := formatWorkspaces(ws)
	for i := range keys {
		opts = append(opts, huh.NewOption(fmt[i], keys[i]))
	}
	s.Options(opts...)

	g := huh.NewGroup(
		s,
	)

	f := huh.NewForm(g)
	err := f.Run()
	if err != nil {
		return workspaceModel{}, err
	}

	return workspaceModel{
		Name:     name,
		Packages: ws[name].Repos,
		LastUsed: ws[name].LastUsed,
	}, nil
}

func CloneWorkspace(name string, packages []string) (workspaceModel, error) {

	name = fmt.Sprintf("Copy of %s", name)

	g := huh.NewGroup(
		huh.NewInput().Title("Workspace name").Value(&name),
	)

	f := huh.NewForm(g)
	err := f.Run()
	if err != nil {
		return workspaceModel{}, err
	}

	return workspaceModel{
		Packages: packages,
		Name:     name,
		LastUsed: time.Now().UTC(),
	}, nil
}

func DisplayWorkspaces(ws config.Workspace) {

	var rows [][]string
	for k, v := range ws {
		var fmtPkg []string
		for _, p := range v.Repos {
			fmtPkg = append(fmtPkg, fmt.Sprintf("* %s", p))
		}
		row := []string{k, strings.Join(fmtPkg, "\n"), v.LastUsed.String()}
		rows = append(rows, row)
	}

	headers := []string{"Name", "Package", "Last Used"}

	displayTable(headers, rows)
}

func EditWorkspace(n string, packages []string, allPackages map[string]string) (workspaceModel, error) {
	var pkgs []string

	s := huh.NewMultiSelect[string]().Value(&pkgs)

	pkgMap := make(map[string]bool)
	for _, p := range packages {
		pkgMap[p] = true
	}

	var opts []huh.Option[string]
	for k, v := range allPackages {
		fmtStr := fmt.Sprintf("%s: %s", k, v)
		_, ok := pkgMap[k]

		opts = append(opts, huh.NewOption(fmtStr, k).Selected(ok))
	}

	s.Options(opts...)
	g := huh.NewGroup(
		s,
	)

	f := huh.NewForm(g)
	err := f.Run()
	if err != nil {
		return workspaceModel{}, err
	}

	return workspaceModel{
		Name:     n,
		Packages: pkgs,
		LastUsed: time.Now().UTC(),
	}, nil
}

func RemoveWorkspace(ws *config.Workspace) error {
	var name string
	s := huh.NewSelect[string]().Title("Choose a workspace").Value(&name)

	var opts []huh.Option[string]
	for k, v := range *ws {
		repos := v.Repos
		fmtStr := fmt.Sprintf("%s [%s]", k, strings.Join(repos[:], ","))
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

	delete(*ws, name)
	return nil
}
