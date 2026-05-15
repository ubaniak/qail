package scripts

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"

	"github.com/ubaniak/qail/internal/color"
	"github.com/ubaniak/qail/internal/qailhome"
	"github.com/ubaniak/qail/internal/runner"
)

// Scope tags a script as workspace-level or repo-level post-install. The
// value also names the on-disk subdirectory under scriptsDir.
type Scope string

const (
	ScopeWorkspace Scope = "workspace"
	ScopeRepo      Scope = "repo"
)

// Valid reports whether the scope is one of the recognised values.
func (s Scope) Valid() bool {
	return s == ScopeWorkspace || s == ScopeRepo
}

// Runner is the narrow subprocess interface the scripts module needs.
type Runner interface {
	Run(ctx context.Context, cmd runner.Command) (runner.Result, error)
}

// Scripts manages bash scripts stored under scriptsDir. Each script
// lives in scriptsDir/<scope>/<name>.sh; the scope subdirectories are
// created by qailhome.Default() so this package can assume they exist.
type Scripts struct {
	r          Runner
	scriptsDir string
}

// New returns a Scripts wired to r and rooted at scriptsDir. The directory
// must already exist; qailhome.Default() creates it during bootstrap.
func New(r Runner, scriptsDir string) *Scripts {
	return &Scripts{r: r, scriptsDir: scriptsDir}
}

// Default returns a Scripts wired to the OS Runner and the production
// scripts directory from qailhome.Default().
func Default() *Scripts {
	h, err := qailhome.Default()
	if err != nil {
		return &Scripts{r: runner.NewOS()}
	}
	return New(runner.NewOS(), h.ScriptsDir())
}

func SortScripts(scripts []string) []string {
	sort.Strings(scripts)
	return scripts
}

// GetScriptDir returns the configured scripts root directory. It exists
// for callers that want to display the path; mutation happens through
// the scoped methods below.
func (s *Scripts) GetScriptDir() (string, error) {
	if s.scriptsDir == "" {
		return "", fmt.Errorf("scripts directory not configured")
	}
	return s.scriptsDir, nil
}

// ScopeDir returns the on-disk directory holding scripts of the given
// scope. Errors when the scripts root is unset or the scope is invalid.
func (s *Scripts) ScopeDir(scope Scope) (string, error) {
	if s.scriptsDir == "" {
		return "", fmt.Errorf("scripts directory not configured")
	}
	if !scope.Valid() {
		return "", fmt.Errorf("invalid scope: %q", scope)
	}
	return filepath.Join(s.scriptsDir, string(scope)), nil
}

// CreateBashScript creates a new bash script in the given scope's
// directory. The `.sh` suffix is added if missing.
func (s *Scripts) CreateBashScript(scriptName string, scope Scope) error {
	scriptContent := `#!/bin/bash

# Add your custom logic here
ls -l
`

	scopeDir, err := s.ScopeDir(scope)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(scopeDir, 0755); err != nil {
		return fmt.Errorf("failed to ensure scope dir: %v", err)
	}

	if len(scriptName) < 3 || scriptName[len(scriptName)-3:] != ".sh" {
		scriptName += ".sh"
	}

	scriptPath := filepath.Join(scopeDir, scriptName)

	if _, err := os.Stat(scriptPath); err == nil {
		return fmt.Errorf("script '%s' already exists in scope '%s'", scriptName, scope)
	}

	file, err := os.Create(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to create script file: %v", err)
	}
	defer file.Close()

	if _, err = file.WriteString(scriptContent); err != nil {
		return fmt.Errorf("failed to write to script file: %v", err)
	}

	if err = os.Chmod(scriptPath, 0755); err != nil {
		return fmt.Errorf("failed to make script executable: %v", err)
	}

	return nil
}

// validateScopedPath ensures abs(scriptPath) lives under scopeDir. Guards
// against path-traversal in callers that accept user-supplied script names.
func validateScopedPath(scopeDir, scriptPath string) error {
	abs, err := filepath.Abs(scriptPath)
	if err != nil {
		return fmt.Errorf("invalid script path: %v", err)
	}
	if !strings.HasPrefix(abs, scopeDir+string(filepath.Separator)) {
		return fmt.Errorf("script path escapes scope directory: %s", abs)
	}
	return nil
}

// RemoveScript deletes scriptName from the given scope's directory.
func (s *Scripts) RemoveScript(scriptName string, scope Scope) error {
	scopeDir, err := s.ScopeDir(scope)
	if err != nil {
		return err
	}
	scriptPath := filepath.Join(scopeDir, scriptName)
	if err := validateScopedPath(scopeDir, scriptPath); err != nil {
		return err
	}
	return os.Remove(scriptPath)
}

// ReadScript returns the textual contents of scriptName under scope.
// View-only consumers (the desktop UI) call this to preview a script;
// mutation goes through Open + the user's editor.
func (s *Scripts) ReadScript(scriptName string, scope Scope) (string, error) {
	scopeDir, err := s.ScopeDir(scope)
	if err != nil {
		return "", err
	}
	scriptPath := filepath.Join(scopeDir, scriptName)
	if err := validateScopedPath(scopeDir, scriptPath); err != nil {
		return "", err
	}
	b, err := os.ReadFile(scriptPath)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// WriteScript replaces the on-disk contents of scriptName under scope.
// Requires the script to already exist so the UI cannot bypass
// CreateBashScript's name validation and scope-bucketing.
func (s *Scripts) WriteScript(scriptName string, scope Scope, contents string) error {
	scopeDir, err := s.ScopeDir(scope)
	if err != nil {
		return err
	}
	scriptPath := filepath.Join(scopeDir, scriptName)
	if err := validateScopedPath(scopeDir, scriptPath); err != nil {
		return err
	}
	info, err := os.Stat(scriptPath)
	if err != nil {
		return err
	}
	mode := info.Mode().Perm()
	if err := os.WriteFile(scriptPath, []byte(contents), mode); err != nil {
		return fmt.Errorf("failed to write script: %v", err)
	}
	return nil
}

// RunBashScript runs scriptName under ctx in dir. Stdout/stderr captured
// from the subprocess are echoed to w (defaulting to os.Stdout when nil)
// preceded by a header so HTTP/SSE callers can stream output without
// touching the actual terminal.
func (s *Scripts) RunBashScript(ctx context.Context, scope Scope, scriptName, dir string, w io.Writer) error {
	if w == nil {
		w = os.Stdout
	}
	scopeDir, err := s.ScopeDir(scope)
	if err != nil {
		return err
	}
	scriptPath := filepath.Join(scopeDir, scriptName)
	if err := validateScopedPath(scopeDir, scriptPath); err != nil {
		return err
	}

	res, err := s.r.Run(ctx, runner.Command{
		Name: "/bin/bash",
		Args: []string{scriptPath},
		Dir:  dir,
	})
	if len(res.Stdout) > 0 {
		fmt.Fprintf(w, "%s %s %s\n\n", color.Yellow(">>>"), color.Green("Stdout"), color.Yellow("<<<"))
		fmt.Fprintln(w, string(res.Stdout))
	}
	if len(res.Stderr) > 0 {
		fmt.Fprintf(w, "%s %s %s\n\n", color.Yellow(">>>"), color.Red("Stderr"), color.Yellow("<<<"))
		fmt.Fprintln(w, string(res.Stderr))
	}

	return err
}

// Has reports whether a script with the given name exists in the scope.
func (s *Scripts) Has(scope Scope, name string) (bool, error) {
	all, err := s.ListScripts(scope)
	if err != nil {
		return false, err
	}
	return slices.Contains(all, name), nil
}

// ListScripts returns the sorted file names in the given scope's directory.
func (s *Scripts) ListScripts(scope Scope) ([]string, error) {
	scopeDir, err := s.ScopeDir(scope)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(scopeDir, 0755); err != nil {
		return nil, err
	}

	files, err := os.ReadDir(scopeDir)
	if err != nil {
		return nil, err
	}
	scriptNames := make([]string, 0, len(files))
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		scriptNames = append(scriptNames, file.Name())
	}

	return SortScripts(scriptNames), nil
}

// Open invokes editor against scriptName in scope.
func (s *Scripts) Open(ctx context.Context, editor string, scope Scope, scriptName string) error {
	if editor == "" {
		return errors.New("no editor selected ... ")
	}

	scopeDir, err := s.ScopeDir(scope)
	if err != nil {
		return err
	}

	scriptPath := filepath.Join(scopeDir, scriptName)
	if err := validateScopedPath(scopeDir, scriptPath); err != nil {
		return err
	}

	_, err = s.r.Run(ctx, runner.Command{
		Name: editor,
		Args: []string{scriptPath},
	})
	return err
}

// OpenDefault hands scriptName to the OS's registered default handler
// (`open` on macOS, `xdg-open` on Linux, `cmd /c start` on Windows).
// Lets users edit a script without configuring a qail editor.
func (s *Scripts) OpenDefault(ctx context.Context, scope Scope, scriptName string) error {
	scopeDir, err := s.ScopeDir(scope)
	if err != nil {
		return err
	}
	scriptPath := filepath.Join(scopeDir, scriptName)
	if err := validateScopedPath(scopeDir, scriptPath); err != nil {
		return err
	}

	var cmd runner.Command
	switch runtime.GOOS {
	case "darwin":
		cmd = runner.Command{Name: "open", Args: []string{scriptPath}}
	case "linux":
		cmd = runner.Command{Name: "xdg-open", Args: []string{scriptPath}}
	case "windows":
		cmd = runner.Command{Name: "cmd", Args: []string{"/c", "start", "", scriptPath}}
	default:
		return fmt.Errorf("no default opener for GOOS=%s", runtime.GOOS)
	}
	_, err = s.r.Run(ctx, cmd)
	return err
}

// CdCommand returns the `cd <scriptsDir>` shell command the caller can
// copy to the clipboard or echo to the user. The qail process cannot
// change the parent shell's directory directly, and the clipboard write
// is a CLI concern (ADR-0010), so this method is pure.
func (s *Scripts) CdCommand() (string, error) {
	scriptDir, err := s.GetScriptDir()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("cd %s", scriptDir), nil
}
