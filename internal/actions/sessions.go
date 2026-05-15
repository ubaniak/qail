package actions

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/ubaniak/qail/internal/config"
	"github.com/ubaniak/qail/internal/runner"
	"github.com/ubaniak/qail/internal/workspace"
)

// Runner is the narrow subprocess interface session-side actions need.
// runner.OS satisfies it in production; runner.Recorder in tests. The
// interface lives in the consumer (actions) per the workspace-injection
// pattern in ADR-0007.
type Runner interface {
	Run(ctx context.Context, cmd runner.Command) (runner.Result, error)
}

// EditorCommand bundles the editor name and the workspace path arg. The
// CLI launches it directly with inherited stdio; the HTTP API serialises
// it into a JSON shape the web client can either display or relay to a
// helper that has terminal access.
type EditorCommand struct {
	Editor string
	Path   string
}

// String renders the editor command for display: `<editor> <path>`. Pure
// cosmetic; no shell-quoting because workspace paths qail manages don't
// contain shell metacharacters by construction.
func (e EditorCommand) String() string {
	return fmt.Sprintf("%s %s", e.Editor, e.Path)
}

// OpenWorkspaceCommand resolves the workspace, picks the editor via the
// default-resolution chain (workspace override > global default), bumps
// LastUsed, and returns the editor invocation as data. Pure — no
// subprocess. CLI handlers that want to actually launch an editor call
// OpenWorkspace; HTTP handlers return the EditorCommand to the client.
func OpenWorkspaceCommand(s config.Store, name string) (EditorCommand, error) {
	return OpenWorkspaceCommandWith(s, name, "")
}

// OpenWorkspaceCommandWith is the explicit-override variant. editorName
// takes precedence over the workspace override and the global default.
// Empty editorName falls back to the workspace override, then the global
// default. Returns "no editor selected" if all three tiers are empty, or
// "editor X not found" if the requested name does not exist.
func OpenWorkspaceCommandWith(s config.Store, name, editorName string) (EditorCommand, error) {
	cfg, profile, wsPath, err := resolveWorkspace(s, name)
	if err != nil {
		return EditorCommand{}, err
	}
	editor, err := resolveEditor(cfg, profile, editorName)
	if err != nil {
		return EditorCommand{}, err
	}
	if err := touchAndWrite(s, cfg, name, profile); err != nil {
		return EditorCommand{}, err
	}
	return EditorCommand{Editor: editor.Command, Path: wsPath}, nil
}

// resolveEditor picks an editor from cfg using the precedence:
// explicit > workspace.Editor > cfg.DefaultEditor.
func resolveEditor(cfg config.Config, profile config.WorkspaceProfile, explicit string) (config.Editor, error) {
	candidates := []string{explicit, profile.Editor, cfg.DefaultEditor}
	for _, name := range candidates {
		if name == "" {
			continue
		}
		ed, ok := findEditor(cfg, name)
		if !ok {
			return config.Editor{}, fmt.Errorf("editor %q not found", name)
		}
		return ed, nil
	}
	return config.Editor{}, errors.New("no editor selected")
}

// OpenWorkspace launches the configured editor against the workspace
// directory after touching LastUsed. Inherits stdio so vim/nvim can take
// over the terminal. Only safe to call from a TTY caller (CLI); HTTP
// handlers should use OpenWorkspaceCommand instead.
func OpenWorkspace(ctx context.Context, s config.Store, r Runner, name string) error {
	cmd, err := OpenWorkspaceCommand(s, name)
	if err != nil {
		return err
	}
	_, err = r.Run(ctx, runner.Command{
		Name:   cmd.Editor,
		Args:   []string{cmd.Path},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
	return err
}

// LaunchEditor spawns the configured editor against the workspace
// directory and returns immediately. No stdio is inherited and no
// context is bound — the editor process must survive the desktop app's
// goroutine lifetime. Intended for the Wails GUI where there is no TTY
// to hand over; works with GUI editors (code, cursor, subl) that
// self-detach. TTY editors (vim) won't render usefully and should still
// use the CLI `qail ws open` path that calls OpenWorkspace.
func LaunchEditor(s config.Store, name string) error {
	return LaunchEditorWith(s, name, "")
}

// LaunchEditorWith is the explicit-override variant of LaunchEditor.
func LaunchEditorWith(s config.Store, name, editorName string) error {
	cmd, err := OpenWorkspaceCommandWith(s, name, editorName)
	if err != nil {
		return err
	}
	c := exec.Command(cmd.Editor, cmd.Path)
	return c.Start()
}

// OpenWorkspace launches the configured editor against the workspace
// with optional override. See OpenWorkspace for stdio semantics.
func OpenWorkspaceWith(ctx context.Context, s config.Store, r Runner, name, editorName string) error {
	cmd, err := OpenWorkspaceCommandWith(s, name, editorName)
	if err != nil {
		return err
	}
	_, err = r.Run(ctx, runner.Command{
		Name:   cmd.Editor,
		Args:   []string{cmd.Path},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
	return err
}

// DefaultEditorCommand returns the command string for the global default
// editor, or "" if none is set. Used by callers (e.g. script open) that
// don't have a workspace context.
func DefaultEditorCommand(s config.Store) (string, error) {
	cfg, err := s.Read()
	if err != nil {
		return "", err
	}
	if cfg.DefaultEditor == "" {
		return "", nil
	}
	ed, ok := findEditor(cfg, cfg.DefaultEditor)
	if !ok {
		return "", nil
	}
	return ed.Command, nil
}

// ExploreWorkspacePath touches LastUsed and returns the workspace
// directory path. Pure — the OS-specific `open` invocation lives in
// ExploreWorkspace, not here, so HTTP handlers can return the path
// without trying to spawn Finder on the server host.
func ExploreWorkspacePath(s config.Store, name string) (string, error) {
	cfg, profile, wsPath, err := resolveWorkspace(s, name)
	if err != nil {
		return "", err
	}
	if err := touchAndWrite(s, cfg, name, profile); err != nil {
		return "", err
	}
	return wsPath, nil
}

// ExploreWorkspace opens the workspace directory in the OS file browser
// (macOS Finder via `open`). Touches LastUsed before invoking. CLI-only;
// HTTP handlers use ExploreWorkspacePath.
func ExploreWorkspace(ctx context.Context, s config.Store, r Runner, name string) error {
	wsPath, err := ExploreWorkspacePath(s, name)
	if err != nil {
		return err
	}
	_, err = r.Run(ctx, runner.Command{
		Name: "open",
		Args: []string{wsPath},
	})
	return err
}

// CdWorkspace touches the workspace and returns its on-disk path so the
// caller can copy a `cd <path>` command to the clipboard. Per ADR-0010 the
// clipboard write happens at the cmd layer, not here.
func CdWorkspace(s config.Store, name string) (string, error) {
	cfg, profile, wsPath, err := resolveWorkspace(s, name)
	if err != nil {
		return "", err
	}
	if err := touchAndWrite(s, cfg, name, profile); err != nil {
		return "", err
	}
	return wsPath, nil
}

// MuxWorkspace ensures a tmux session exists for the workspace, touches
// LastUsed, and returns the shell command the user must paste to attach.
// Per ADR-0011 the clipboard write happens at the cmd layer.
func MuxWorkspace(ctx context.Context, s config.Store, name string) (string, error) {
	cfg, profile, wsPath, err := resolveWorkspace(s, name)
	if err != nil {
		return "", err
	}
	if err := touchAndWrite(s, cfg, name, profile); err != nil {
		return "", err
	}
	return workspace.Tmux(ctx, wsPath)
}

// resolveWorkspace loads cfg, validates the workspace is registered, and
// asserts the on-disk directory exists. Returned profile is pre-touch so
// callers can hand it to touchAndWrite once they're ready to commit.
func resolveWorkspace(s config.Store, name string) (config.Config, config.WorkspaceProfile, string, error) {
	cfg, err := s.Read()
	if err != nil {
		return config.Config{}, config.WorkspaceProfile{}, "", err
	}
	profile, ok := cfg.Workspaces[name]
	if !ok {
		return config.Config{}, config.WorkspaceProfile{}, "", fmt.Errorf("workspace %q not found", name)
	}
	wsPath := path.Join(cfg.Root, name)
	if _, err := os.Stat(wsPath); os.IsNotExist(err) {
		return config.Config{}, config.WorkspaceProfile{}, "", fmt.Errorf("workspace %q does not exist on disk. Please run qail ws create", wsPath)
	}
	return cfg, profile, wsPath, nil
}

// touchAndWrite bumps LastUsed for name and persists the snapshot. Splits
// out so resolveWorkspace can run side-effect-free validations first.
// Acquires storeMu so a concurrent action's Read+Write doesn't lose the
// LastUsed bump.
func touchAndWrite(s config.Store, cfg config.Config, name string, profile config.WorkspaceProfile) error {
	storeMu.Lock()
	defer storeMu.Unlock()
	cfg.Workspaces[name] = config.NewWorkspaceProfile(profile.Repos, time.Now().UTC())
	return s.Write(cfg)
}
