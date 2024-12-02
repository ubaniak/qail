package scripts

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"qail/internal/clip"
	"qail/internal/color"
)

func SortScripts(scripts []string) []string {
	sort.Strings(scripts)
	return scripts
}

func GetScriptDir() (string, error) {
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
func CreateBashScript(scriptName string) error {
	scriptContent := `#!/bin/bash

# Add your custom logic here 
ls -l 
`

	scriptsDir, err := GetScriptDir()
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

	err = os.Chmod(scriptName, 0755)
	if err != nil {
		return fmt.Errorf("failed to make script executable: %v", err)
	}

	return nil
}

func RemoveScript(scriptName string) error {
	scriptsDir, err := GetScriptDir()
	if err != nil {
		return err
	}
	scriptPath := filepath.Join(scriptsDir, scriptName)
	return os.Remove(scriptPath)
}

func RunBashScript(scriptName, dir string) error {
	scriptsDir, err := GetScriptDir()
	if err != nil {
		return err
	}
	scriptPath := filepath.Join(scriptsDir, scriptName)
	cmd := exec.Command("/bin/bash", scriptPath)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if stdout.String() != "" {
		fmt.Printf("%s %s %s\n\n", color.Yellow(">>>"), color.Green("Stdout"), color.Yellow("<<<"))
		fmt.Println(stdout.String())
	}
	if stderr.String() != "" {
		fmt.Printf("%s %s %s\n\n", color.Yellow(">>>"), color.Red("Stderr"), color.Yellow("<<<"))
		fmt.Println(stderr.String())
	}

	return err
}

func ListScripts() ([]string, error) {
	scriptsDir, err := GetScriptDir()
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

func Open(editor, scriptName string) error {
	if editor == "" {
		return errors.New("no editor selected ... ")
	}

	scriptDir, err := GetScriptDir()
	if err != nil {
		return err
	}

	scriptPath := filepath.Join(scriptDir, scriptName)

	cmd := exec.Command(editor, scriptPath)

	_, err = cmd.Output()
	return err
}

func Cd() error {
	scriptDir, err := GetScriptDir()
	if err != nil {
		return err
	}
	clip.Cd(scriptDir)
	return nil
}
