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

// jsonConfig mirrors Config with JSON tags for parsing config.json.
type jsonConfig struct {
	Root               string                   `json:"root"`
	Editor             string                   `json:"editor"`
	Workspaces         map[string]jsonWorkspace `json:"workspaces"`
	Repos              map[string]string        `json:"repos"`
	PostInstallScripts jsonPostInstallScripts   `json:"post_install_scripts"`
}

type jsonWorkspace struct {
	Repos    []string  `json:"repos"`
	LastUsed time.Time `json:"last_used"`
}

type jsonPostInstallScripts struct {
	Repo      map[string][]string `json:"repo"`
	Workspace map[string][]string `json:"workspace"`
}

// BackUp copies the on-disk SQLite DB next to itself as <dbPath>.bak.
func (s *SQLiteStore) BackUp() error {
	return copyFile(s.dbPath, s.dbPath+".bak")
}

// Restore copies <dbPath>.bak back over the live DB.
func (s *SQLiteStore) Restore() error {
	return copyFile(s.dbPath+".bak", s.dbPath)
}

// ConvertJSON reads jsonPath (legacy config.json) and writes it into the
// store via Write. Migration is SQLite-specific (the in-memory store is
// fresh each run), so it lives on SQLiteStore rather than the Store
// interface.
func (s *SQLiteStore) ConvertJSON(jsonPath string) error {
	file, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %v", jsonPath, err)
	}

	var jcfg jsonConfig
	if err := json.Unmarshal(file, &jcfg); err != nil {
		return fmt.Errorf("failed to parse %s: %v", jsonPath, err)
	}

	cfg := Config{
		Root:       jcfg.Root,
		Editor:     jcfg.Editor,
		Repos:      jcfg.Repos,
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

	return s.Write(cfg)
}

// Package-level shims that hit the default SQLite store. cmd/config.go
// drives these from the `qail config convert` command. They panic to
// preserve the previous user-visible behaviour; the underlying methods
// return errors and should be preferred by new callers.

func BackUpConfig() {
	s, err := defaultStore()
	if err != nil {
		panic(fmt.Sprintf("failed to back up database: %v", err))
	}
	if err := s.BackUp(); err != nil {
		panic(fmt.Sprintf("failed to back up database: %v", err))
	}
}

func RestoreConfig() {
	s, err := defaultStore()
	if err != nil {
		panic(fmt.Sprintf("failed to restore database: %v", err))
	}
	if err := s.Restore(); err != nil {
		panic(fmt.Sprintf("failed to restore database: %v", err))
	}
}

func ConvertOldToNew() {
	s, err := defaultStore()
	if err != nil {
		panic(fmt.Sprintf("failed to open store: %v", err))
	}
	jsonPath := filepath.Join(defaultRootDir, "config.json")
	if err := s.ConvertJSON(jsonPath); err != nil {
		panic(err.Error())
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
