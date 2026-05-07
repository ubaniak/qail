// Package config owns the qail Config types and the Store seam over their
// persistence. Two adapters live alongside: SQLiteStore (production) and
// MemoryStore (tests). Package-level helpers (ReadFromFile, WriteToFile,
// WithConfig, ValidateConfig) are thin shims that delegate to a lazily
// initialised default SQLiteStore at ~/.qail/qail.db.
package config

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Public types

type Workspace map[string]WorkspaceProfile

type WorkspaceProfile struct {
	Repos    []string
	LastUsed time.Time
}

func NewWorkspaceProfile(repos []string, lastUsed time.Time) WorkspaceProfile {
	return WorkspaceProfile{
		Repos:    repos,
		LastUsed: lastUsed,
	}
}

type PostInstallScripts struct {
	Repo      map[string][]string
	Workspace map[string][]string
}

type Config struct {
	Root               string
	Editor             string
	Workspaces         Workspace
	Repos              map[string]string
	PostInstallScripts PostInstallScripts
}

// Default store — lazy SQLite at ~/.qail/qail.db.

var (
	defaultStoreOnce sync.Once
	defaultStoreVal  *SQLiteStore
	defaultStoreErr  error
	defaultRootDir   string
)

// defaultStore returns (and memoises) the production SQLite store living at
// ~/.qail/qail.db. The parent directory is created on first use.
func defaultStore() (*SQLiteStore, error) {
	defaultStoreOnce.Do(func() {
		h, err := os.UserHomeDir()
		if err != nil {
			defaultStoreErr = err
			return
		}
		defaultRootDir = filepath.Join(h, ".qail")
		if err := os.MkdirAll(defaultRootDir, 0755); err != nil {
			defaultStoreErr = err
			return
		}
		s, err := NewSQLiteStore(filepath.Join(defaultRootDir, "qail.db"))
		if err != nil {
			defaultStoreErr = err
			return
		}
		defaultStoreVal = s
	})
	return defaultStoreVal, defaultStoreErr
}

// Package-level shims preserve the existing call sites in cmd/. They will be
// dropped once cmd/ migrates to taking a Store directly (see #5 Actions).

func ValidateConfig() error {
	cfg, err := ReadFromFile()
	if err != nil {
		return err
	}
	if cfg.Root == "" {
		return errors.New("root folder is not set")
	}
	return nil
}

func WithConfig(fn func(cfg *Config) error) error {
	cfg, err := ReadFromFile()
	if err != nil {
		return err
	}
	if err := fn(&cfg); err != nil {
		return err
	}
	return WriteToFile(cfg)
}

func ReadFromFile() (Config, error) {
	s, err := defaultStore()
	if err != nil {
		return Config{}, err
	}
	return s.Read()
}

func WriteToFile(cfg Config) error {
	s, err := defaultStore()
	if err != nil {
		return err
	}
	return s.Write(cfg)
}
