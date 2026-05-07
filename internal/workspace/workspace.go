package workspace

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/atotto/clipboard"

	"github.com/ubaniak/qail/internal/color"
	"github.com/ubaniak/qail/internal/config"
	"github.com/ubaniak/qail/internal/forms"
	"github.com/ubaniak/qail/internal/installer"
	"github.com/ubaniak/qail/internal/runner"
	"github.com/ubaniak/qail/internal/tmux"
)

// Installer is the narrow consumer interface workspace.Create needs to
// populate a workspace directory. *installer.Installer satisfies it. The
// interface lives here (the consumer) so workspace tests can substitute a
// fake without importing the production installer.
type Installer interface {
	Install(spec installer.PackageSpec) error
	RunPostInstall(dir string, scripts []string) error
}

type Workspace struct {
	Root            string
	Name            string
	Packages        []string
	Repos           map[string]string
	RepoPostInstall map[string][]string
	WSPostInstall   map[string][]string
	inst            Installer
	fs              FS
}

// New constructs a Workspace wired to inst and fs. Production callers
// should use NewDefault; tests inject fakes here.
func New(root, name string, packages []string, repos map[string]string, inst Installer, fs FS) Workspace {
	return Workspace{
		Root:            root,
		Name:            name,
		Packages:        packages,
		Repos:           repos,
		RepoPostInstall: make(map[string][]string),
		WSPostInstall:   make(map[string][]string),
		inst:            inst,
		fs:              fs,
	}
}

// NewDefault constructs a Workspace wired to installer.Default() and
// OSFS{} — the production OS-backed git + scripts + filesystem trio. Cmd
// handlers and actions use this; tests use New with fake adapters.
func NewDefault(root, name string, packages []string, repos map[string]string) Workspace {
	return New(root, name, packages, repos, installer.Default(), OSFS{})
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
	if _, err := w.fs.Stat(w.Root); os.IsNotExist(err) {
		if err := w.fs.Mkdir(w.Root, 0755); err != nil {
			return fmt.Errorf("failed to create root directory: %v", err)
		}
	}

	wsPath := path.Join(w.Root, w.Name)

	wsCreated := false
	if _, err := w.fs.Stat(wsPath); os.IsNotExist(err) {
		if err := w.fs.Mkdir(wsPath, 0755); err != nil {
			return fmt.Errorf("failed to create workspace directory: %v", err)
		}
		wsCreated = true
	}

	if err := w.populate(wsPath); err != nil {
		if wsCreated {
			w.fs.RemoveAll(wsPath)
		}
		return err
	}
	return nil
}

func (w Workspace) populate(wsPath string) error {
	fmt.Printf("Creating workspace %s ...\n", color.Cyan(wsPath))

	for _, p := range w.Packages {
		fmt.Printf("* Adding package %s ...\n", color.Cyan(p))
		spec := installer.PackageSpec{
			Name:        p,
			RepoURL:     w.Repos[p],
			Dest:        path.Join(wsPath, p),
			PostInstall: w.RepoPostInstall[p],
		}
		if err := w.inst.Install(spec); err != nil {
			return err
		}
		fmt.Println()
	}

	if wsScripts, ok := w.WSPostInstall[w.Name]; ok {
		if len(wsScripts) > 0 {
			fmt.Printf("Running post install scripts for %s: %s \n", color.Green("workspace"), color.Cyan(w.Name))
		}
		if err := w.inst.RunPostInstall(wsPath, wsScripts); err != nil {
			return err
		}
	}

	fmt.Println(color.Green("Done :)"))

	return nil
}

func (w Workspace) Remove() error {
	wsPath := path.Join(w.Root, w.Name)

	fmt.Printf("removing %s", wsPath)
	return w.fs.RemoveAll(wsPath)
}

func (w Workspace) RemoveRepo(repo string) error {
	wsPath := path.Join(w.Root, w.Name, repo)

	return w.fs.RemoveAll(wsPath)
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

// Cd copies a `cd <ws>` command to the clipboard so the user can paste it
// into a real shell. The qail process cannot change the parent shell's
// directory directly.
func Cd(ws string) {
	cmd := fmt.Sprintf("cd %s", ws)
	fmt.Printf("%s copied %s to clipboard\n\n", color.Yellow(">>>"), color.Green(cmd))
	clipboard.WriteAll(cmd)
}

// Tmux ensures a tmux session exists for the workspace at ws, then returns
// the shell command the user must paste to attach. The qail process cannot
// attach itself; the caller decides whether to copy the command to the
// clipboard or display it.
func Tmux(ws string) (string, error) {
	t := tmux.Default()
	err, _ := t.IsInstalled()
	if err != nil {
		return "", err
	}
	sessionName := t.SessionName(ws)
	if !t.SessionExists(sessionName) {
		if err := t.Launch(ws); err != nil {
			return "", err
		}
	}
	return t.AttachCommand(sessionName), nil
}

// Clean lists root, identifies subdirectories not tracked in ws, prompts
// the user, then deletes the orphans. Production callers go through Clean;
// tests substitute fs and confirm via cleanWithDeps.
func Clean(root string, ws config.Workspace) error {
	return cleanWithDeps(OSFS{}, forms.Confirm, root, ws)
}

// cleanWithDeps is the testable shape: pass a FS adapter and a confirm
// function. Production wires OSFS + forms.Confirm; tests wire fakes.
func cleanWithDeps(fs FS, confirm func(string) (bool, error), root string, ws config.Workspace) error {
	files, err := fs.ReadDir(root)
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

	confirmed, err := confirm("Delete these directories?")
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("Aborted.")
		return nil
	}

	for _, name := range toDelete {
		fmt.Printf("%s Deleting: %s\n", color.Yellow(">>>"), color.Cyan(name))
		if err := fs.RemoveAll(path.Join(root, name)); err != nil {
			return err
		}
	}

	return nil
}
