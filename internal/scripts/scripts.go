package scripts

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"qail/internal/color"
)

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

# Function: run
# Takes a folder path as input and performs an action
run() {
    local folder_path="$1"

    if [ -z "$folder_path" ]; then
        echo "Usage: run <folder_path>"
        return 1
    fi

    if [ ! -d "$folder_path" ]; then
        echo "Error: Folder '$folder_path' not found."
        return 1
    fi

    echo "Processing folder: $folder_path"
    # Add your custom logic here (e.g., listing files)
    ls -l "$folder_path"
}

# Execute the run function with the first argument
run "$@"
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

func RunBashScript(scriptName, path string) error {
	scriptsDir, err := GetScriptDir()
	if err != nil {
		return err
	}
	scriptPath := filepath.Join(scriptsDir, scriptName)
	cmd := exec.Command("/bin/bash", scriptPath, path)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	fmt.Println(color.Green("Stdout"))
	fmt.Println(stdout.String())
	if stderr.String() != "" {
		fmt.Println(color.Red("Stderr"))
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

	return scriptNames, nil
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
