// Package qailhome owns the qail home-directory layout. It is the single
// source of truth for every well-known path under ~/.qail (or whatever
// $QAIL_HOME points at). Both internal/config and internal/scripts depend
// on it instead of independently calling os.UserHomeDir + filepath.Join.
//
// Construction:
//   - Default() reads $QAIL_HOME, falls back to ~/.qail, creates Root and
//     ScriptsDir, and is the entry point for production code.
//   - New(root) builds a Home pointing at an explicit root with no mkdirs;
//     tests pass a t.TempDir().
package qailhome

import (
	"os"
	"path/filepath"
)

// Home is the qail home directory and the well-known paths derived from it.
type Home struct {
	root string
}

// New returns a Home rooted at root. No directories are created; the caller
// is responsible for ensuring root and any subdirectories exist.
func New(root string) Home {
	return Home{root: root}
}

// Default returns the production Home. It reads $QAIL_HOME if set,
// otherwise falls back to $HOME/.qail. Both Root() and ScriptsDir() are
// created on disk if missing so that downstream callers (config.Store,
// scripts.Scripts) can assume the layout exists.
func Default() (Home, error) {
	root, err := resolveRoot()
	if err != nil {
		return Home{}, err
	}
	h := New(root)
	if err := os.MkdirAll(h.Root(), 0755); err != nil {
		return Home{}, err
	}
	if err := os.MkdirAll(h.ScriptsDir(), 0755); err != nil {
		return Home{}, err
	}
	return h, nil
}

func resolveRoot() (string, error) {
	if r := os.Getenv("QAIL_HOME"); r != "" {
		return r, nil
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, ".qail"), nil
}

// Root returns the qail home directory.
func (h Home) Root() string { return h.root }

// DBPath returns the on-disk path of the SQLite config database.
func (h Home) DBPath() string { return filepath.Join(h.root, "qail.db") }

// ScriptsDir returns the directory holding user-managed bash scripts.
func (h Home) ScriptsDir() string { return filepath.Join(h.root, "scripts") }
