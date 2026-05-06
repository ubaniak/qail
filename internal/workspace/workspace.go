package workspace

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"

	"github.com/ubaniak/qail/internal/clip"
	"github.com/ubaniak/qail/internal/color"
	"github.com/ubaniak/qail/internal/config"
	"github.com/ubaniak/qail/internal/forms"
	"github.com/ubaniak/qail/internal/git"
	"github.com/ubaniak/qail/internal/scripts"
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
	for _, p := range w.Packages {
		fmt.Printf("* Adding package %s ...\n", color.Cyan(p))
		rPath := path.Join(wsPath, p)
		if r, ok := w.Repos[p]; ok {
			m := fmt.Sprintf("Cloning %s", color.Cyan(p))
			git.CloneWithProgress(r, rPath, m)
		}
		if postInstallScripts, ok := w.RepoPostInstall[p]; ok {
			if len(postInstallScripts) > 0 {
				fmt.Printf("Running post install scripts for %s: %s \n", color.Green("repos"), color.Cyan(p))
			}
			for _, s := range postInstallScripts {
				fmt.Printf("   * Running post install script: %s\n", color.Cyan(s))
				scripts.RunBashScript(s, rPath)
			}
		}
		fmt.Println()
	}

	if postInstallScripts, ok := w.WSPostInstall[w.Name]; ok {
		if len(postInstallScripts) > 0 {
			fmt.Printf("Running post install scripts for %s: %s \n", color.Green("workspace"), color.Cyan(w.Name))
		}
		for _, s := range postInstallScripts {
			fmt.Printf("   * Running post install script: %s\n", color.Cyan(s))
			scripts.RunBashScript(s, wsPath)
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

func Open(editor, workspace string) {
	if editor == "" {
		log.Fatalln("No editor selected ... ")
	}

	cmd := exec.Command(editor, workspace)

	cmd.Output()
}

func Explore(ws string) (string, error) {
	cmd := exec.Command("open", ws)

	out, err := cmd.Output()

	return string(out), err
}

func Cd(ws string) {
	clip.Cd(ws)
}

func Tmux(ws string) error {
	err, _ := tmux.IsInstalled()
	if err != nil {
		return err
	}
	sessionName := tmux.SessionName(ws)
	if !tmux.SessionExists(sessionName) {
		err := tmux.Launch(ws)
		if err != nil {
			return err
		}
	}
	tmux.Attach(sessionName)
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
