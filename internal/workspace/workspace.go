package workspace

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/ubaniak/qail/internal/color"
	"github.com/ubaniak/qail/internal/config"
	"github.com/ubaniak/qail/internal/forms"
	"github.com/ubaniak/qail/internal/installer"
	"github.com/ubaniak/qail/internal/tmux"
)

// Installer is the narrow consumer interface workspace.Create needs to
// populate a workspace directory. *installer.Installer satisfies it. The
// interface lives here (the consumer) so workspace tests can substitute a
// fake without importing the production installer.
type Installer interface {
	Install(ctx context.Context, spec installer.PackageSpec, w io.Writer) error
	RunPostInstall(ctx context.Context, dir string, scripts []string, w io.Writer) error
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

// NewStreaming is NewDefault for non-TTY callers — uses installer.NewStreaming
// so progress lines flow through whichever io.Writer the caller hands to
// Create instead of being trapped under a huh spinner.
func NewStreaming(root, name string, packages []string, repos map[string]string) Workspace {
	return New(root, name, packages, repos, installer.NewStreaming(), OSFS{})
}

func (w *Workspace) WithRepoPostInstallScripts(p map[string][]string) *Workspace {
	w.RepoPostInstall = p
	return w
}

func (w *Workspace) WithWSPostInstallScripts(p map[string][]string) *Workspace {
	w.WSPostInstall = p
	return w
}

// Create materialises the workspace on disk: mkdir root + name (if needed),
// install each package, run workspace-level post-install scripts. Progress
// is written to out (defaulting to os.Stdout when nil). On failure the
// freshly-created workspace dir is removed so callers can safely retry.
func (w Workspace) Create(ctx context.Context, out io.Writer) error {
	if out == nil {
		out = os.Stdout
	}
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

	if err := w.populate(ctx, wsPath, out); err != nil {
		if wsCreated {
			w.fs.RemoveAll(wsPath)
		}
		return err
	}
	return nil
}

func (w Workspace) populate(ctx context.Context, wsPath string, out io.Writer) error {
	fmt.Fprintf(out, "Creating workspace %s ...\n", color.Cyan(wsPath))

	for _, p := range w.Packages {
		fmt.Fprintf(out, "* Adding package %s ...\n", color.Cyan(p))
		spec := installer.PackageSpec{
			Name:        p,
			RepoURL:     w.Repos[p],
			Dest:        path.Join(wsPath, p),
			PostInstall: w.RepoPostInstall[p],
		}
		if err := w.inst.Install(ctx, spec, out); err != nil {
			return err
		}
		fmt.Fprintln(out)
	}

	if wsScripts, ok := w.WSPostInstall[w.Name]; ok {
		if len(wsScripts) > 0 {
			fmt.Fprintf(out, "Running post install scripts for %s: %s \n", color.Green("workspace"), color.Cyan(w.Name))
		}
		if err := w.inst.RunPostInstall(ctx, wsPath, wsScripts, out); err != nil {
			return err
		}
	}

	fmt.Fprintln(out, color.Green("Done :)"))

	return nil
}

func (w Workspace) Remove() error {
	wsPath := path.Join(w.Root, w.Name)
	return w.fs.RemoveAll(wsPath)
}

func (w Workspace) RemoveRepo(repo string) error {
	wsPath := path.Join(w.Root, w.Name, repo)

	return w.fs.RemoveAll(wsPath)
}

// Tmux ensures a tmux session exists for the workspace at ws under ctx,
// then returns the shell command the user must paste to attach. The qail
// process cannot attach itself; the caller decides whether to copy the
// command to the clipboard or display it.
func Tmux(ctx context.Context, ws string) (string, error) {
	t := tmux.Default()
	err, _ := t.IsInstalled(ctx)
	if err != nil {
		return "", err
	}
	sessionName := t.SessionName(ws)
	exists, err := t.SessionExists(ctx, sessionName)
	if err != nil {
		return "", err
	}
	if !exists {
		if err := t.Launch(ctx, ws); err != nil {
			return "", err
		}
	}
	return t.AttachCommand(sessionName), nil
}

// Clean lists root, identifies subdirectories not tracked in ws, prompts
// the user, then deletes the orphans. Production callers go through Clean;
// tests substitute fs and confirm via cleanWithDeps.
func Clean(root string, ws config.Workspace, out io.Writer) error {
	return cleanWithDeps(OSFS{}, forms.Confirm, root, ws, out)
}

// cleanWithDeps is the testable shape: pass a FS adapter, confirm function,
// and writer. Production wires OSFS + forms.Confirm + os.Stdout; tests
// wire fakes.
func cleanWithDeps(fs FS, confirm func(string) (bool, error), root string, ws config.Workspace, out io.Writer) error {
	if out == nil {
		out = os.Stdout
	}
	files, err := fs.ReadDir(root)
	if err != nil {
		return err
	}

	fmt.Fprintln(out, "Reading ...", color.Cyan(root))

	var toDelete []string
	for _, file := range files {
		if !file.IsDir() {
			continue
		}
		fmt.Fprintln(out, "Folder name", color.Cyan(file.Name()))
		if _, ok := ws[file.Name()]; !ok {
			toDelete = append(toDelete, file.Name())
		}
	}

	if len(toDelete) == 0 {
		return nil
	}

	fmt.Fprintf(out, "%s The following directories are not tracked and will be deleted:\n", color.Yellow(">>>"))
	for _, name := range toDelete {
		fmt.Fprintf(out, "   * %s\n", color.Cyan(name))
	}

	confirmed, err := confirm("Delete these directories?")
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Fprintln(out, "Aborted.")
		return nil
	}

	for _, name := range toDelete {
		fmt.Fprintf(out, "%s Deleting: %s\n", color.Yellow(">>>"), color.Cyan(name))
		if err := fs.RemoveAll(path.Join(root, name)); err != nil {
			return err
		}
	}

	return nil
}
