package app

import (
	"errors"
	"sort"

	"github.com/ubaniak/qail/internal/actions"
	"github.com/ubaniak/qail/internal/console"
)

// Agent Console (ADR-0018, ADR-0020). The Electron tray builds its "Agent
// Console" submenu from ConsoleStatus, and drives the in-process console
// through OpenConsole and StopConsole.

// ConsoleStatusDTO tells the tray what to show.
type ConsoleStatusDTO struct {
	// ClaudeFound is false when the Claude Code CLI can't be found; the
	// console still opens, but agents can't run.
	ClaudeFound bool `json:"claudeFound"`
	// Running is true while this backend serves the console. A console
	// served elsewhere (e.g. `qail console`) is not counted.
	Running bool `json:"running"`
	// Workspaces to scope the console to, most recently used first. It
	// skips ListWorkspaces' branch lookup so the tray menu opens fast.
	Workspaces []ConsoleWorkspaceDTO `json:"workspaces"`
}

type ConsoleWorkspaceDTO struct {
	Name  string   `json:"name"`
	Repos []string `json:"repos"`
}

var errNoConsole = errors.New("agent console is not available in this backend")

// ConsoleStatus reports whether claude is installed and the console running.
func (b *Bindings) ConsoleStatus() (ConsoleStatusDTO, error) {
	var out ConsoleStatusDTO
	if b.console != nil {
		out.ClaudeFound, out.Running = b.console.ClaudeFound(), b.console.Running()
	}
	workspaces, _, err := actions.ListWorkspaces(b.store)
	if err != nil {
		return out, err
	}
	for name, ws := range workspaces {
		out.Workspaces = append(out.Workspaces, ConsoleWorkspaceDTO{Name: name, Repos: ws.Repos})
	}
	sort.Slice(out.Workspaces, func(i, j int) bool {
		return workspaces[out.Workspaces[i].Name].LastUsed.After(workspaces[out.Workspaces[j].Name].LastUsed)
	})
	return out, nil
}

// OpenConsole starts the console if needed and opens it in the browser,
// scoped to workspace (and repo) when given. Empty workspace opens the
// console's home page.
func (b *Bindings) OpenConsole(workspace, repo string) error {
	if b.console == nil {
		return errNoConsole
	}
	return b.console.Open(b.ctx, console.ScopePath(workspace, repo))
}

// StopConsole stops the console this backend serves.
func (b *Bindings) StopConsole() error {
	if b.console != nil {
		b.console.Stop()
	}
	return nil
}
