package tmux

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ubaniak/qail/internal/runner"
)

// Runner is the narrow subprocess interface the tmux module needs.
type Runner interface {
	Run(ctx context.Context, cmd runner.Command) (runner.Result, error)
}

// Tmux wraps the tmux CLI behind a Runner seam.
type Tmux struct {
	r Runner
}

// New returns a Tmux wired to r.
func New(r Runner) *Tmux { return &Tmux{r: r} }

// Default returns a Tmux wired to the OS Runner.
func Default() *Tmux { return New(runner.NewOS()) }

// AttachCommand returns the shell command the user must run to attach to
// sessionName from a real terminal. tmux requires a TTY, so the qail
// process cannot attach itself; the caller decides what to do with the
// returned string (typically copy it to the clipboard).
func (t *Tmux) AttachCommand(sessionName string) string {
	return fmt.Sprintf("tmux a -t %s", shellQuote(sessionName))
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// SessionName returns the tmux session name derived from a folder path.
func (t *Tmux) SessionName(path string) string {
	return filepath.Base(path)
}

// Launch creates a tmux session for the workspace at folderPath under ctx.
// Layout policy lives in Plan; Launch only reads the filesystem, filters
// hidden / non-directory entries, and dispatches the planned commands
// through Runner.
func (t *Tmux) Launch(ctx context.Context, folderPath string) error {
	sessionName := t.SessionName(folderPath)

	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return err
	}

	var subfolders []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		subfolders = append(subfolders, e.Name())
	}

	for _, cmd := range Plan(sessionName, folderPath, subfolders) {
		if _, err := t.r.Run(ctx, cmd); err != nil {
			return fmt.Errorf("tmux %v: %w", cmd.Args, err)
		}
	}
	return nil
}

// SessionExists reports whether a tmux session of the given name is live.
// Returns (false, nil) when tmux is reachable and the session is absent;
// (false, err) when the tmux binary itself is unreachable. Splitting the
// two lets callers distinguish "no such session" from "tmux is broken".
func (t *Tmux) SessionExists(ctx context.Context, sessionName string) (bool, error) {
	_, err := t.r.Run(ctx, runner.Command{
		Name: "tmux",
		Args: []string{"has-session", "-t", sessionName},
	})
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, err
}

// IsInstalled returns (nil, true) if `tmux -V` succeeds. The tmux version
// string is discarded; CLI callers that want it should call `tmux -V`
// themselves.
func (t *Tmux) IsInstalled(ctx context.Context) (error, bool) {
	_, err := t.r.Run(ctx, runner.Command{
		Name: "tmux",
		Args: []string{"-V"},
	})
	if err != nil {
		return err, false
	}
	return nil, true
}

// ListSessions returns live tmux session names.
func (t *Tmux) ListSessions(ctx context.Context) ([]string, error) {
	res, err := t.r.Run(ctx, runner.Command{
		Name: "tmux",
		Args: []string{"list-sessions", "-F", "#S"},
	})
	if err != nil {
		return nil, fmt.Errorf("error running tmux command: %s, stderr: %s", err, string(res.Stderr))
	}
	sessions := strings.Split(strings.TrimSpace(string(res.Stdout)), "\n")
	return sessions, nil
}

// HasSession reports whether the named session appears in `tmux
// list-sessions`. Used by cmd handlers to fail fast on a typo before
// prompting for confirmation; concentrates the list+contains check so
// callers don't roll their own.
func (t *Tmux) HasSession(ctx context.Context, name string) (bool, error) {
	all, err := t.ListSessions(ctx)
	if err != nil {
		return false, err
	}
	return slices.Contains(all, name), nil
}

// RemoveSession kills the named tmux session.
func (t *Tmux) RemoveSession(ctx context.Context, session string) error {
	res, err := t.r.Run(ctx, runner.Command{
		Name: "tmux",
		Args: []string{"kill-session", "-t", session},
	})
	if err != nil {
		return fmt.Errorf("failed to remove tmux session '%s': %s", session, string(res.Stderr))
	}
	return nil
}
