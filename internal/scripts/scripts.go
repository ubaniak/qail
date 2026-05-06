package scripts

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ubaniak/qail/internal/clip"
	"github.com/ubaniak/qail/internal/color"
	"github.com/ubaniak/qail/internal/runner"
)

// Runner is the narrow subprocess interface the scripts module needs.
type Runner interface {
	Run(ctx context.Context, cmd runner.Command) (runner.Result, error)
}

// Scripts manages bash scripts stored under ~/.qail/scripts.
type Scripts struct {
	r Runner
}

// New returns a Scripts wired to r.
func New(r Runner) *Scripts { return &Scripts{r: r} }

// Default returns a Scripts wired to the OS Runner.
func Default() *Scripts { return New(runner.NewOS()) }

func SortScripts(scripts []string) []string {
	sort.Strings(scripts)
	return scripts
}

func (s *Scripts) GetScriptDir() (string, error) {
	var rootDir string
	var scriptsDir string
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	rootDir = filepath.Join(h, ".qail")
	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		os.Mkdir(rootDir, 0755)
	}

	scriptsDir = filepath.Join(rootDir, "scripts")

	if _, err := os.Stat(scriptsDir); os.IsNotExist(err) {
		os.Mkdir(scriptsDir, 0755)
	}

	return scriptsDir, nil
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

func (s *Scripts) RunBashScript(scriptName, dir string) error {
	scriptsDir, err := s.GetScriptDir()
	if err != nil {
		return err
	}
	scriptPath := filepath.Join(scriptsDir, scriptName)
	if err := validateScriptPath(scriptsDir, scriptPath); err != nil {
		return err
	}

	res, err := s.r.Run(context.Background(), runner.Command{
		Name: "/bin/bash",
		Args: []string{scriptPath},
		Dir:  dir,
	})
	if len(res.Stdout) > 0 {
		fmt.Printf("%s %s %s\n\n", color.Yellow(">>>"), color.Green("Stdout"), color.Yellow("<<<"))
		fmt.Println(string(res.Stdout))
	}
	if len(res.Stderr) > 0 {
		fmt.Printf("%s %s %s\n\n", color.Yellow(">>>"), color.Red("Stderr"), color.Yellow("<<<"))
		fmt.Println(string(res.Stderr))
	}

	return err
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

func (s *Scripts) Open(editor, scriptName string) error {
	if editor == "" {
		return errors.New("no editor selected ... ")
	}

	scriptDir, err := s.GetScriptDir()
	if err != nil {
		return err
	}

	scriptPath := filepath.Join(scriptDir, scriptName)

	_, err = s.r.Run(context.Background(), runner.Command{
		Name: editor,
		Args: []string{scriptPath},
	})
	return err
}

func (s *Scripts) Cd() error {
	scriptDir, err := s.GetScriptDir()
	if err != nil {
		return err
	}
	clip.Cd(scriptDir)
	return nil
}
