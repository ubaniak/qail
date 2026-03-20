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
	dest := filepath.Join(rootDir, "qail.db.bak")
	if err := copyFile(dbPath, dest); err != nil {
		panic(fmt.Sprintf("failed to back up database: %v", err))
	}
}

func RestoreConfig() {
	src := filepath.Join(rootDir, "qail.db.bak")
	if err := copyFile(src, dbPath); err != nil {
		panic(fmt.Sprintf("failed to restore database: %v", err))
	}
}

// jsonConfig mirrors Config with JSON tags for parsing config.json.
type jsonConfig struct {
	Root               string                    `json:"root"`
	Editor             string                    `json:"editor"`
	Workspaces         map[string]jsonWorkspace  `json:"workspaces"`
	Repos              map[string]string         `json:"repos"`
	PostInstallScripts jsonPostInstallScripts    `json:"post_install_scripts"`
}

type jsonWorkspace struct {
	Repos    []string  `json:"repos"`
	LastUsed time.Time `json:"last_used"`
}

type jsonPostInstallScripts struct {
	Repo      map[string][]string `json:"repo"`
	Workspace map[string][]string `json:"workspace"`
}

// ConvertOldToNew reads config.json (either old or new format) and imports it into the SQLite database.
func ConvertOldToNew() {
	jsonPath := filepath.Join(rootDir, "config.json")
	file, err := os.ReadFile(jsonPath)
	if err != nil {
		panic(fmt.Sprintf("failed to read config.json: %v", err))
	}

	// Try new JSON format first (workspaces with repos+last_used objects).
	var jcfg jsonConfig
	if err := json.Unmarshal(file, &jcfg); err != nil {
		panic(fmt.Sprintf("failed to parse config.json: %v", err))
	}

	cfg := Config{
		Root:   jcfg.Root,
		Editor: jcfg.Editor,
		Repos:  jcfg.Repos,
		Workspaces: make(Workspace, len(jcfg.Workspaces)),
		PostInstallScripts: PostInstallScripts{
			Repo:      jcfg.PostInstallScripts.Repo,
			Workspace: jcfg.PostInstallScripts.Workspace,
		},
	}

	for name, ws := range jcfg.Workspaces {
		lastUsed := ws.LastUsed
		if lastUsed.IsZero() {
			lastUsed = time.Now().UTC()
		}
		cfg.Workspaces[name] = WorkspaceProfile{
			Repos:    ws.Repos,
			LastUsed: lastUsed,
		}
	}

	if err := WriteToFile(cfg); err != nil {
		panic(fmt.Sprintf("failed to write to database: %v", err))
	}
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	if _, err = io.Copy(destinationFile, sourceFile); err != nil {
		return err
	}

	return destinationFile.Sync()
}
