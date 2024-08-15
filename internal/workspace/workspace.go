package workspace

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"qail/internal/compression"
	"qail/internal/git"
)

type Workspace struct {
	Root     string
	Name     string
	Packages []string
	Repos    map[string]string
}

func New(root, name string, packages []string, repos map[string]string) Workspace {
	return Workspace{
		Root:     root,
		Name:     name,
		Packages: packages,
		Repos:    repos,
	}
}

func (w Workspace) Create() error {
	if _, err := os.Stat(w.Root); os.IsNotExist(err) {
		os.Mkdir(w.Root, 0755)
	}

	wsPath := path.Join(w.Root, w.Name)

	if _, err := os.Stat(wsPath); os.IsNotExist(err) {
		os.Mkdir(wsPath, 0755)
	}

	fmt.Printf("Creating workspace %s ...\n", wsPath)
	for _, p := range w.Packages {
		fmt.Printf("Adding package %s ...\n", p)
		r, ok := w.Repos[p]
		if ok {
			rPath := path.Join(wsPath, p)
			m := fmt.Sprintf("Cloning %s", p)
			git.ConeWithProgress(r, rPath, m)
		}
	}

	fmt.Println("Done :)")

	return nil
}

func (w Workspace) Remove() error {
	wsPath := path.Join(w.Root, w.Name)

	fmt.Printf("removing %s", wsPath)
	return nil
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
	fmt.Printf("cd %s\n", ws)
}

func Clean(ws map[string][]string, rootDir, archiveDir string) error {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			folderName := entry.Name()
			if _, exists := ws[folderName]; !exists {
				src := filepath.Join(rootDir, folderName)
				dest := filepath.Join(archiveDir, folderName+".tar.gz")
				fmt.Printf("Archiving folder: %s\n", src)
				if err := compression.TarGzipFolder(src, dest); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
