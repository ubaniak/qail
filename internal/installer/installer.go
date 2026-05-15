// Package installer owns per-package install: clone the repo (if any) and run
// post-install scripts, in order, fail-fast. It is the single place where the
// "clone-then-script" invariant lives, so any future caller (qail update,
// qail ws add-package) reuses the same orchestration.
package installer

import (
	"context"
	"fmt"
	"io"
	"os"

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

// GitClient is the narrow consumer interface for cloning. The production
// adapter (spinnerGit) wraps *git.Git in a progress UI; the streaming
// adapter (streamGit) writes plain progress lines for non-TTY callers
// (HTTP handlers); tests substitute a fake.
type GitClient interface {
	Clone(ctx context.Context, repo, path, message string, w io.Writer) error
}

// ScriptsClient is the narrow consumer interface for executing a named
// post-install script. *scripts.Scripts satisfies it. Scope selects the
// scripts/<scope>/ subdirectory the name resolves under.
type ScriptsClient interface {
	RunBashScript(ctx context.Context, scope scripts.Scope, scriptName, dir string, w io.Writer) error
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
// The git client is wrapped in spinnerGit so workspace creation shows a
// progress UI without leaking that concern into the git package. Use this
// from CLI code; HTTP handlers use NewStreaming.
func Default() *Installer {
	return New(spinnerGit{inner: git.Default()}, scripts.Default())
}

// NewStreaming returns an Installer that emits plain progress lines through
// the writer the caller supplies at call time. No TUI dependencies; suitable
// for HTTP/SSE flows.
func NewStreaming() *Installer {
	return New(streamGit{inner: git.Default()}, scripts.Default())
}

// Install clones the repo (if RepoURL set) into Dest, then runs the package's
// post-install scripts in Dest. Returns on the first error. Progress lines
// are written to w; w may be nil in which case os.Stdout is used.
func (i *Installer) Install(ctx context.Context, spec PackageSpec, w io.Writer) error {
	w = orStdout(w)
	if spec.RepoURL != "" {
		msg := fmt.Sprintf("Cloning %s", color.Cyan(spec.Name))
		if err := i.git.Clone(ctx, spec.RepoURL, spec.Dest, msg, w); err != nil {
			return fmt.Errorf("clone %s: %w", spec.Name, err)
		}
	}
	if len(spec.PostInstall) > 0 {
		fmt.Fprintf(w, "Running post install scripts for %s: %s\n", color.Green("repos"), color.Cyan(spec.Name))
	}
	return i.RunPostInstall(ctx, scripts.ScopeRepo, spec.Dest, spec.PostInstall, w)
}

// RunPostInstall runs the given scripts against dir, fail-fast. scope
// selects which script directory the names resolve under so the same
// installer can drive both per-repo and per-workspace flows.
func (i *Installer) RunPostInstall(ctx context.Context, scope scripts.Scope, dir string, names []string, w io.Writer) error {
	w = orStdout(w)
	for _, s := range names {
		fmt.Fprintf(w, "   * Running post install script: %s\n", color.Cyan(s))
		if err := i.scripts.RunBashScript(ctx, scope, s, dir, w); err != nil {
			return fmt.Errorf("post-install script %s: %w", s, err)
		}
	}
	return nil
}

// orStdout returns w, or os.Stdout if w is nil. Lets domain code assume a
// non-nil writer without forcing every caller to pass os.Stdout explicitly.
func orStdout(w io.Writer) io.Writer {
	if w == nil {
		return os.Stdout
	}
	return w
}
