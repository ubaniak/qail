// Package installer owns per-package install: clone the repo (if any) and run
// post-install scripts, in order, fail-fast. It is the single place where the
// "clone-then-script" invariant lives, so any future caller (qail update,
// qail ws add-package) reuses the same orchestration.
package installer

import (
	"fmt"

	"github.com/ubaniak/qail/internal/color"
	"github.com/ubaniak/qail/internal/git"
	"github.com/ubaniak/qail/internal/scripts"
)

// PackageSpec describes one package install. RepoURL == "" skips the clone
// (e.g. packages whose source is provided out-of-band but still want
// post-install scripts).
type PackageSpec struct {
	Name        string
	RepoURL     string
	Dest        string
	PostInstall []string
}

// GitClient is the narrow consumer interface for cloning. *git.Git satisfies it.
type GitClient interface {
	CloneWithProgress(repo, path, message string) error
}

// ScriptsClient is the narrow consumer interface for executing a named
// post-install script. *scripts.Scripts satisfies it.
type ScriptsClient interface {
	RunBashScript(scriptName, dir string) error
}

// Installer wires a GitClient + ScriptsClient.
type Installer struct {
	git     GitClient
	scripts ScriptsClient
}

// New returns an Installer wired to the given clients.
func New(g GitClient, s ScriptsClient) *Installer {
	return &Installer{git: g, scripts: s}
}

// Default returns an Installer wired to the OS-backed git + scripts clients.
func Default() *Installer {
	return New(git.Default(), scripts.Default())
}

// Install clones the repo (if RepoURL set) into Dest, then runs the package's
// post-install scripts in Dest. Returns on the first error.
func (i *Installer) Install(spec PackageSpec) error {
	if spec.RepoURL != "" {
		msg := fmt.Sprintf("Cloning %s", color.Cyan(spec.Name))
		if err := i.git.CloneWithProgress(spec.RepoURL, spec.Dest, msg); err != nil {
			return fmt.Errorf("clone %s: %w", spec.Name, err)
		}
	}
	if len(spec.PostInstall) > 0 {
		fmt.Printf("Running post install scripts for %s: %s\n", color.Green("repos"), color.Cyan(spec.Name))
	}
	return i.RunPostInstall(spec.Dest, spec.PostInstall)
}

// RunPostInstall runs the given scripts against dir, fail-fast.
func (i *Installer) RunPostInstall(dir string, scripts []string) error {
	for _, s := range scripts {
		fmt.Printf("   * Running post install script: %s\n", color.Cyan(s))
		if err := i.scripts.RunBashScript(s, dir); err != nil {
			return fmt.Errorf("post-install script %s: %w", s, err)
		}
	}
	return nil
}
