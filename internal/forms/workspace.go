package forms

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
)

type workspaceModel struct {
	Name     string
	Packages []string
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
	}, nil
}

func FindWorkspace(ws map[string][]string) (workspaceModel, error) {
	var name string
	s := huh.NewSelect[string]().Title("Choose a workspace").Value(&name)

	var opts []huh.Option[string]
	for k, v := range ws {
		fmtStr := fmt.Sprintf("%s [%s]", k, strings.Join(v[:], ","))
		opts = append(opts, huh.NewOption(fmtStr, k))
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
		Packages: ws[name],
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
	}, nil
}

func DisplayWorkspaces(ws map[string][]string) {

	var rows [][]string
	for k, v := range ws {
		var fmtPkg []string
		for _, p := range v {
			fmtPkg = append(fmtPkg, fmt.Sprintf("* %s", p))
		}
		row := []string{k, strings.Join(fmtPkg, "\n")}
		rows = append(rows, row)
	}

	headers := []string{"Name", "Package"}

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
	}, nil
}

func RemoveWorkspace(ws *map[string][]string) error {
	var name string
	s := huh.NewSelect[string]().Title("Choose a workspace").Value(&name)

	var opts []huh.Option[string]
	for k, v := range *ws {
		fmtStr := fmt.Sprintf("%s [%s]", k, strings.Join(v[:], ","))
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
