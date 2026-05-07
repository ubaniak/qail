package tmux

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ubaniak/qail/internal/clip"
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

// Attach copies the `tmux a -t <session>` command to the clipboard. The
// caller pastes it into a real shell since `tmux a` requires a TTY.
func (t *Tmux) Attach(sessionName string) {
	cmd := fmt.Sprintf("tmux a -t %s", shellQuote(sessionName))
	clip.Cmd(cmd)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// SessionName returns the tmux session name derived from a folder path.
func (t *Tmux) SessionName(path string) string {
	return filepath.Base(path)
}

// Launch creates a tmux session for the workspace at folderPath. Layout
// policy lives in Plan; Launch only reads the filesystem, filters hidden /
// non-directory entries, and dispatches the planned commands through Runner.
func (t *Tmux) Launch(folderPath string) error {
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
		if _, err := t.r.Run(context.Background(), cmd); err != nil {
			return fmt.Errorf("tmux %v: %w", cmd.Args, err)
		}
	}
	return nil
}

// SessionExists reports whether a tmux session of the given name is live.
func (t *Tmux) SessionExists(sessionName string) bool {
	_, err := t.r.Run(context.Background(), runner.Command{
		Name: "tmux",
		Args: []string{"has-session", "-t", sessionName},
	})
	if err == nil {
		return true
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return false
	}
	// Other errors (binary missing etc.) — log and treat as absent.
	fmt.Printf("Error checking tmux session: %v\n", err)
	return false
}

// IsInstalled returns (nil, true) if `tmux -V` succeeds.
func (t *Tmux) IsInstalled() (error, bool) {
	res, err := t.r.Run(context.Background(), runner.Command{
		Name: "tmux",
		Args: []string{"-V"},
	})
	if err != nil {
		return err, false
	}
	fmt.Printf("tmux version: %s\n", res.Stdout)
	return nil, true
}

// ListSessions returns live tmux session names.
func (t *Tmux) ListSessions() ([]string, error) {
	res, err := t.r.Run(context.Background(), runner.Command{
		Name: "tmux",
		Args: []string{"list-sessions", "-F", "#S"},
	})
	if err != nil {
		return nil, fmt.Errorf("error running tmux command: %s, stderr: %s", err, string(res.Stderr))
	}
	sessions := strings.Split(strings.TrimSpace(string(res.Stdout)), "\n")
	return sessions, nil
}

// RemoveSession kills the named tmux session.
func (t *Tmux) RemoveSession(session string) error {
	res, err := t.r.Run(context.Background(), runner.Command{
		Name: "tmux",
		Args: []string{"kill-session", "-t", session},
	})
	if err != nil {
		return fmt.Errorf("failed to remove tmux session '%s': %s", session, string(res.Stderr))
	}
	return nil
}
