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

func isEven(i int) bool {
	return i%2 == 0
}

// Launch creates a tmux session rooted at folderPath and adds windows/panes
// for each non-hidden subfolder. Even-indexed subfolders split the current
// window horizontally; odd-indexed ones open a new window.
func (t *Tmux) Launch(folderPath string) error {
	if err := os.Chdir(folderPath); err != nil {
		return fmt.Errorf("failed to change directory: %v", err)
	}

	sessionName := t.SessionName(folderPath)
	if _, err := t.r.Run(context.Background(), runner.Command{
		Name: "tmux",
		Args: []string{"new-session", "-d", "-s", sessionName, "-c", folderPath, "-n", "root"},
	}); err != nil {
		return fmt.Errorf("failed to create tmux session: %v", err)
	}

	subFolders, err := os.ReadDir(folderPath)
	if err != nil {
		return err
	}
	if len(subFolders) == 0 {
		return nil
	}

	if _, err := t.r.Run(context.Background(), runner.Command{
		Name: "tmux",
		Args: []string{"new-window", "-t", sessionName, "-n", "SubFolders"},
	}); err != nil {
		return fmt.Errorf("failed to create new window: %v", err)
	}

	folderNumber := 0
	windowIndex := 0

	for _, subFolder := range subFolders {
		if subFolder.IsDir() && strings.HasPrefix(subFolder.Name(), ".") {
			continue
		}
		if !subFolder.IsDir() {
			continue
		}
		subfolderPath := filepath.Join(folderPath, subFolder.Name())

		if isEven(folderNumber) {
			if _, err := t.r.Run(context.Background(), runner.Command{
				Name: "tmux",
				Args: []string{"split-window", "-t", fmt.Sprintf("%s:%d", sessionName, windowIndex), "-c", subfolderPath, "-h"},
			}); err != nil {
				return fmt.Errorf("failed to split window: %v", err)
			}
		} else {
			windowIndex++
			if _, err := t.r.Run(context.Background(), runner.Command{
				Name: "tmux",
				Args: []string{"new-window", "-t", sessionName, "-c", subfolderPath, "-n", "SubFolders" + fmt.Sprintf("%d", windowIndex)},
			}); err != nil {
				return fmt.Errorf("failed to create new window: %v", err)
			}
		}
		folderNumber++
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
