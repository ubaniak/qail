// Package config owns the qail Config types and the Store seam over their
// persistence. Two adapters live alongside: SQLiteStore (production) and
// MemoryStore (tests). DefaultStore() returns the lazily-initialised
// production Store at ~/.qail/qail.db.
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
	defaultStoreOnce  sync.Once
	defaultStoreVal   *SQLiteStore
	defaultStoreErr   error
	defaultLegacyJSON string
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
		defaultLegacyJSON = h.LegacyJSONPath()
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
// ~/.qail/qail.db. Cmd handlers go through this; tests construct
// MemoryStore directly.
func DefaultStore() (Store, error) {
	return defaultStore()
}

// ValidateConfig reports whether the persisted config has the minimum
// fields required for qail to operate (currently: Root must be set).
func ValidateConfig() error {
	s, err := defaultStore()
	if err != nil {
		return err
	}
	cfg, err := s.Read()
	if err != nil {
		return err
	}
	if cfg.Root == "" {
		return errors.New("root folder is not set")
	}
	return nil
}
