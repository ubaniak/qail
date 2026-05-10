package forms

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/huh"

	"github.com/ubaniak/qail/internal/config"
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
		fmtStr := fmt.Sprintf("%s [%s] %s", key, strings.Join(repos[:], ","), ws[key].LastUsed.Format(time.RFC822))
		formatted = append(formatted, fmtStr)
	}
	return keys, formatted
}

func NewWorkspace(allRepos map[string]string) (workspaceModel, error) {
	var name string
	var selectedRepos []string

	s := huh.NewMultiSelect[string]().Value(&selectedRepos)
	repoNames := SortRepos(allRepos)

	var opts []huh.Option[string]
	for _, k := range repoNames {
		v := allRepos[k]
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

	return workspaceModel{
		Name:     name,
		Packages: selectedRepos,
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

func DisplayWorkspaces(ws config.Workspace, postInstallScripts map[string][]string) {

	headers := []string{"Name", "Package", "Last Used", "Post install scripts"}
	keys := SortWorkspaces(ws)
	var rows [][]string
	for _, k := range keys {
		v := ws[k]
		var fmtPkg []string
		for _, p := range v.Repos {
			fmtPkg = append(fmtPkg, fmt.Sprintf("* %s", p))
		}
		psScripts := []string{}
		if scripts, ok := postInstallScripts[k]; ok {
			for _, s := range scripts {
				psScripts = append(psScripts, fmt.Sprintf("* %s", s))
			}
		}

		row := []string{k, strings.Join(fmtPkg, "\n"), v.LastUsed.Format(time.RFC822), strings.Join(psScripts, "\n")}
		rows = append(rows, row)
	}

	displayTable(headers, rows)
}

func EditWorkspace(n string, packages []string, allRepos map[string]string) (workspaceModel, error) {
	var pkgs []string

	s := huh.NewMultiSelect[string]().Value(&pkgs)

	pkgMap := make(map[string]bool)
	for _, p := range packages {
		pkgMap[p] = true
	}

	var opts []huh.Option[string]

	repoNames := SortRepos(allRepos)

	for _, k := range repoNames {
		v := allRepos[k]
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

// SelectWorkspaceName prompts the user to pick a workspace and returns the
// chosen name. Pure selector — confirmation is the caller's job (chain
// with forms.Confirm or cmd's confirmOrSkip helper). Splitting the two
// concerns lets each be tested and reused independently.
func SelectWorkspaceName(ws config.Workspace) (string, error) {
	var name string
	s := huh.NewSelect[string]().Title("Choose a workspace").Value(&name)

	var opts []huh.Option[string]
	keys, fmt := formatWorkspaces(ws)
	for i := range keys {
		opts = append(opts, huh.NewOption(fmt[i], keys[i]))
	}
	s.Options(opts...)

	f := huh.NewForm(huh.NewGroup(s))
	if err := f.Run(); err != nil {
		return "", err
	}
	return name, nil
}
