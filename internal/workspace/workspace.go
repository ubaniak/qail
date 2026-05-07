package workspace

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/ubaniak/qail/internal/clip"
	"github.com/ubaniak/qail/internal/color"
	"github.com/ubaniak/qail/internal/config"
	"github.com/ubaniak/qail/internal/forms"
	"github.com/ubaniak/qail/internal/installer"
	"github.com/ubaniak/qail/internal/runner"
	"github.com/ubaniak/qail/internal/tmux"
)

type Workspace struct {
	Root            string
	Name            string
	Packages        []string
	Repos           map[string]string
	RepoPostInstall map[string][]string
	WSPostInstall   map[string][]string
}

func New(root, name string, packages []string, repos map[string]string) Workspace {
	return Workspace{
		Root:            root,
		Name:            name,
		Packages:        packages,
		Repos:           repos,
		RepoPostInstall: make(map[string][]string),
		WSPostInstall:   make(map[string][]string),
	}
}

func (w *Workspace) WithRepoPostInstallScripts(p map[string][]string) *Workspace {
	w.RepoPostInstall = p
	return w
}

func (w *Workspace) WithWSPostInstallScripts(p map[string][]string) *Workspace {
	w.WSPostInstall = p
	return w
}

func (w Workspace) Create() error {
	if _, err := os.Stat(w.Root); os.IsNotExist(err) {
		if err := os.Mkdir(w.Root, 0755); err != nil {
			return fmt.Errorf("failed to create root directory: %v", err)
		}
	}

	wsPath := path.Join(w.Root, w.Name)

	wsCreated := false
	if _, err := os.Stat(wsPath); os.IsNotExist(err) {
		if err := os.Mkdir(wsPath, 0755); err != nil {
			return fmt.Errorf("failed to create workspace directory: %v", err)
		}
		wsCreated = true
	}

	if err := w.populate(wsPath); err != nil {
		if wsCreated {
			os.RemoveAll(wsPath)
		}
		return err
	}
	return nil
}

func (w Workspace) populate(wsPath string) error {
	fmt.Printf("Creating workspace %s ...\n", color.Cyan(wsPath))
	inst := installer.Default()

	for _, p := range w.Packages {
		fmt.Printf("* Adding package %s ...\n", color.Cyan(p))
		spec := installer.PackageSpec{
			Name:        p,
			RepoURL:     w.Repos[p],
			Dest:        path.Join(wsPath, p),
			PostInstall: w.RepoPostInstall[p],
		}
		if err := inst.Install(spec); err != nil {
			return err
		}
		fmt.Println()
	}

	if wsScripts, ok := w.WSPostInstall[w.Name]; ok {
		if len(wsScripts) > 0 {
			fmt.Printf("Running post install scripts for %s: %s \n", color.Green("workspace"), color.Cyan(w.Name))
		}
		if err := inst.RunPostInstall(wsPath, wsScripts); err != nil {
			return err
		}
	}

	fmt.Println(color.Green("Done :)"))

	return nil
}

func (w Workspace) Remove() error {
	wsPath := path.Join(w.Root, w.Name)

	fmt.Printf("removing %s", wsPath)
	return os.RemoveAll(wsPath)
}

func (w Workspace) RemoveRepo(repo string) error {
	wsPath := path.Join(w.Root, w.Name, repo)

	return os.RemoveAll(wsPath)
}

// defaultRunner returns the production Runner used by free functions in this
// package. Methods on Workspace will gain explicit injection in a later pass.
func defaultRunner() *runner.OS { return runner.NewOS() }

// Open launches the configured editor against the workspace path, inheriting
// stdio so editors like vim/nvim can take over the terminal.
func Open(editor, workspace string) {
	if editor == "" {
		log.Fatalln("No editor selected ... ")
	}

	defaultRunner().Run(context.Background(), runner.Command{
		Name:   editor,
		Args:   []string{workspace},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
}

// Explore opens the workspace in the macOS Finder via `open`.
func Explore(ws string) (string, error) {
	res, err := defaultRunner().Run(context.Background(), runner.Command{
		Name: "open",
		Args: []string{ws},
	})
	return string(res.Stdout), err
}

func Cd(ws string) {
	clip.Cd(ws)
}

func Tmux(ws string) error {
	t := tmux.Default()
	err, _ := t.IsInstalled()
	if err != nil {
		return err
	}
	sessionName := t.SessionName(ws)
	if !t.SessionExists(sessionName) {
		err := t.Launch(ws)
		if err != nil {
			return err
		}
	}
	t.Attach(sessionName)
	return nil
}

func Clean(root string, ws config.Workspace) error {
	files, err := os.ReadDir(root)
	if err != nil {
		return err
	}

	fmt.Println("Reading ...", color.Cyan(root))

	var toDelete []string
	for _, file := range files {
		if !file.IsDir() {
			continue
		}
		fmt.Println("Folder name", color.Cyan(file.Name()))
		if _, ok := ws[file.Name()]; !ok {
			toDelete = append(toDelete, file.Name())
		}
	}

	if len(toDelete) == 0 {
		return nil
	}

	fmt.Printf("%s The following directories are not tracked and will be deleted:\n", color.Yellow(">>>"))
	for _, name := range toDelete {
		fmt.Printf("   * %s\n", color.Cyan(name))
	}

	confirmed, err := forms.Confirm("Delete these directories?")
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("Aborted.")
		return nil
	}

	for _, name := range toDelete {
		fmt.Printf("%s Deleting: %s\n", color.Yellow(">>>"), color.Cyan(name))
		if err := os.RemoveAll(path.Join(root, name)); err != nil {
			return err
		}
	}

	return nil
}
