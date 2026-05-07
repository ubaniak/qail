// Package config owns the qail Config types and the Store seam over their
// persistence. Two adapters live alongside: SQLiteStore (production) and
// MemoryStore (tests). Package-level helpers (ReadFromFile, WriteToFile,
// WithConfig, ValidateConfig) are thin shims that delegate to a lazily
// initialised default SQLiteStore at ~/.qail/qail.db.
package config

import (
	"errors"
	"sync"
	"time"

	"github.com/ubaniak/qail/internal/qailhome"
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

// defaultStore returns (and memoises) the production SQLite store. Path
// resolution and directory creation happen in qailhome.Default.
func defaultStore() (*SQLiteStore, error) {
	defaultStoreOnce.Do(func() {
		h, err := qailhome.Default()
		if err != nil {
			defaultStoreErr = err
			return
		}
		defaultRootDir = h.Root()
		s, err := NewSQLiteStore(h.DBPath())
		if err != nil {
			defaultStoreErr = err
			return
		}
		defaultStoreVal = s
	})
	return defaultStoreVal, defaultStoreErr
}

// DefaultStore returns the lazily-initialised production Store at
// ~/.qail/qail.db. New cmd handlers should prefer this over the package-level
// shims below.
func DefaultStore() (Store, error) {
	return defaultStore()
}

// Package-level shims preserve the existing call sites in cmd/. They will be
// dropped once cmd/ migrates to taking a Store directly (see #5 Actions).
//
// Deprecated: call config.DefaultStore() and use Store.Read / Store.Write,
// or the actions package, which atomically wraps Read+mutate+Write.

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

// WithConfig runs fn against a snapshot, then writes the mutated snapshot
// back atomically through the default Store.
//
// Deprecated: prefer the actions package which exposes one function per
// user-facing operation, each backed by an explicit Store.
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
