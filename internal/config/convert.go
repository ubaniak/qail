package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type OldConfig struct {
	Root       string              `json:"root"`
	Editor     string              `json:"editor"`
	Workspaces map[string][]string `json:"workspaces"`
	Repos      map[string]string   `json:"repos"`
}

func BackUpConfig() {
	dest := filepath.Join(rootDir, "config.json.bak")

	err := copyFile(configPath, dest)
	if err != nil {
		panic(fmt.Sprintf("failed to copy config file: %v", err))
	}
}

func RestoreConfig() {
	dest := filepath.Join(rootDir, "config.json.bak")

	err := copyFile(dest, configPath)
	if err != nil {
		panic(fmt.Sprintf("failed to copy config file: %v", err))
	}

}

func ConvertOldToNew() {
	var old OldConfig
	new := Config{}

	file, err := os.ReadFile(configPath)
	if err != nil {
		panic(fmt.Sprintf("failed to read input file: %v", err))
	}

	err = json.Unmarshal(file, &old)
	if err != nil {
		panic(fmt.Sprintf("failed to unmarshal input JSON: %v", err))
	}

	new.Editor = old.Editor
	new.Repos = old.Repos
	new.Root = old.Root
	new.Workspaces = make(map[string]WorkspaceProfile)

	for key, value := range old.Workspaces {
		new.Workspaces[key] = WorkspaceProfile{
			Repos:    value,
			LastUsed: time.Now().UTC(),
		}
	}

	data, err := json.Marshal(new)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal input JSON: %v", err))
	}

	os.WriteFile(configPath, data, 0600)
}

func copyFile(src, dst string) error {
	// Open the source file
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create the destination file
	destinationFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	// Copy the content from source to destination
	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return err
	}

	// Ensure all data is flushed to the destination file
	err = destinationFile.Sync()
	if err != nil {
		return err
	}

	return nil
}
