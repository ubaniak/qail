package scripts

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/ubaniak/qail/internal/color"
	"github.com/ubaniak/qail/internal/qailhome"
	"github.com/ubaniak/qail/internal/runner"
)

// Runner is the narrow subprocess interface the scripts module needs.
type Runner interface {
	Run(ctx context.Context, cmd runner.Command) (runner.Result, error)
}

// Scripts manages bash scripts stored in scriptsDir.
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
		// qailhome creation failure is fatal for production callers; keep
		// the previous panic-on-init behaviour by returning a Scripts
		// pointed at a path that will fail on first use.
		return &Scripts{r: runner.NewOS()}
	}
	return New(runner.NewOS(), h.ScriptsDir())
}

func SortScripts(scripts []string) []string {
	sort.Strings(scripts)
	return scripts
}

// GetScriptDir returns the configured scripts directory. It exists for
// callers that want to display or change directory to the path; mutation
// happens through the Scripts methods, never via this string.
func (s *Scripts) GetScriptDir() (string, error) {
	if s.scriptsDir == "" {
		return "", fmt.Errorf("scripts directory not configured")
	}
	return s.scriptsDir, nil
}

// CreateBashScript generates a bash script with a specified name
func (s *Scripts) CreateBashScript(scriptName string) error {
	scriptContent := `#!/bin/bash

# Add your custom logic here
ls -l
`

	scriptsDir, err := s.GetScriptDir()
	if err != nil {
		return err
	}

	if len(scriptName) < 3 || scriptName[len(scriptName)-3:] != ".sh" {
		scriptName += ".sh"
	}

	scriptPath := filepath.Join(scriptsDir, scriptName)

	if _, err := os.Stat(scriptPath); err == nil {
		return fmt.Errorf("script '%s' already exists in directory '%s'", scriptName, scriptsDir)
	}

	file, err := os.Create(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to create script file: %v", err)
	}
	defer file.Close()

	_, err = file.WriteString(scriptContent)
	if err != nil {
		return fmt.Errorf("failed to write to script file: %v", err)
	}

	err = os.Chmod(scriptPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to make script executable: %v", err)
	}

	return nil
}

func validateScriptPath(scriptsDir, scriptPath string) error {
	abs, err := filepath.Abs(scriptPath)
	if err != nil {
		return fmt.Errorf("invalid script path: %v", err)
	}
	if !strings.HasPrefix(abs, scriptsDir+string(filepath.Separator)) {
		return fmt.Errorf("script path escapes scripts directory: %s", abs)
	}
	return nil
}

func (s *Scripts) RemoveScript(scriptName string) error {
	scriptsDir, err := s.GetScriptDir()
	if err != nil {
		return err
	}
	scriptPath := filepath.Join(scriptsDir, scriptName)
	if err := validateScriptPath(scriptsDir, scriptPath); err != nil {
		return err
	}
	return os.Remove(scriptPath)
}

// RunBashScript runs scriptName under ctx in dir. Stdout/stderr captured
// from the subprocess are echoed to w (defaulting to os.Stdout when nil)
// preceded by a header so HTTP/SSE callers can stream output without
// touching the actual terminal.
func (s *Scripts) RunBashScript(ctx context.Context, scriptName, dir string, w io.Writer) error {
	if w == nil {
		w = os.Stdout
	}
	scriptsDir, err := s.GetScriptDir()
	if err != nil {
		return err
	}
	scriptPath := filepath.Join(scriptsDir, scriptName)
	if err := validateScriptPath(scriptsDir, scriptPath); err != nil {
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

// Has reports whether a script with the given name exists in the scripts
// directory. Used by callers that want to fail fast on a typo before
// running confirm prompts; the domain owns this check so cmd handlers do
// not reach into ListScripts to roll their own.
func (s *Scripts) Has(name string) (bool, error) {
	all, err := s.ListScripts()
	if err != nil {
		return false, err
	}
	return slices.Contains(all, name), nil
}

func (s *Scripts) ListScripts() ([]string, error) {
	scriptsDir, err := s.GetScriptDir()
	if err != nil {
		return nil, err
	}

	files, err := os.ReadDir(scriptsDir)
	if err != nil {
		return nil, err
	}
	scriptNames := make([]string, len(files))
	for i, file := range files {
		scriptNames[i] = file.Name()
	}

	return SortScripts(scriptNames), nil
}

func (s *Scripts) Open(ctx context.Context, editor, scriptName string) error {
	if editor == "" {
		return errors.New("no editor selected ... ")
	}

	scriptDir, err := s.GetScriptDir()
	if err != nil {
		return err
	}

	scriptPath := filepath.Join(scriptDir, scriptName)

	_, err = s.r.Run(ctx, runner.Command{
		Name: editor,
		Args: []string{scriptPath},
	})
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
