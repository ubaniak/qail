package workspace

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"

	"qail/internal/clip"
	"qail/internal/color"
	"qail/internal/config"
	"qail/internal/git"
	"qail/internal/scripts"
	"qail/internal/tmux"
)

type Workspace struct {
	Root            string
	Name            string
	Packages        []string
	Repos           map[string]string
	RepoPostInstall map[string][]string
}

func New(root, name string, packages []string, repos map[string]string) Workspace {
	return Workspace{
		Root:            root,
		Name:            name,
		Packages:        packages,
		Repos:           repos,
		RepoPostInstall: make(map[string][]string),
	}
}

func (w *Workspace) WithRepoPostInstallScripts(p map[string][]string) *Workspace {
	w.RepoPostInstall = p
	return w
}

func (w Workspace) Create() error {
	if _, err := os.Stat(w.Root); os.IsNotExist(err) {
		os.Mkdir(w.Root, 0755)
	}

	wsPath := path.Join(w.Root, w.Name)

	if _, err := os.Stat(wsPath); os.IsNotExist(err) {
		os.Mkdir(wsPath, 0755)
	}

	fmt.Printf("Creating workspace %s ...\n", color.Cyan(wsPath))
	for _, p := range w.Packages {
		fmt.Printf("* Adding package %s ...\n", color.Cyan(p))
		rPath := path.Join(wsPath, p)
		if r, ok := w.Repos[p]; ok {
			m := fmt.Sprintf("Cloning %s", color.Cyan(p))
			git.ConeWithProgress(r, rPath, m)
		}
		if postInstallScipts, ok := w.RepoPostInstall[p]; ok {
			for _, s := range postInstallScipts {
				fmt.Printf("   * Running post install script: %s\n", color.Cyan(s))
				scripts.RunBashScript(s, rPath)
			}

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

	for _, file := range files {
		if !file.IsDir() {
			continue
		}
		fmt.Println("Folder name", color.Cyan(file.Name()))
		_, ok := ws[file.Name()]
		if !ok {
			fmt.Printf("%s Deleting: %s\n", color.Yellow(">>>"), color.Cyan(file.Name()))
			err := os.RemoveAll(path.Join(root, file.Name()))
			if err != nil {
				return err
			}
		}
	}

	return nil
}
