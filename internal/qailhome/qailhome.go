// Package qailhome owns the qail home-directory layout. It is the single
// source of truth for every well-known path under ~/.qail (or whatever
// $QAIL_HOME points at). Both internal/config and internal/scripts depend
// on it instead of independently calling os.UserHomeDir + filepath.Join.
//
// Construction:
//   - Default() reads $QAIL_HOME, falls back to ~/.qail, creates the home
//     and ScriptsDir, and is the entry point for production code.
//   - New(root) builds a Home pointing at an explicit root with no mkdirs;
//     tests pass a t.TempDir().
//
// Path accessors are deliberately purpose-specific (DBPath, ScriptsDir,
// LegacyJSONPath). Callers should not compute paths under the home root
// themselves; add a new accessor here when a new well-known path appears.
package qailhome

import (
	"os"
	"path/filepath"
	"strings"
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
// otherwise falls back to $HOME/.qail. The home root, ScriptsDir, and
// per-scope script subdirectories are created on disk if missing so that
// downstream callers (config.Store, scripts.Scripts) can assume the
// layout exists. Legacy flat scripts under ScriptsDir are migrated into
// the workspace subdir on first Default() call (see migrateLegacyScripts).
func Default() (Home, error) {
	root, err := resolveRoot()
	if err != nil {
		return Home{}, err
	}
	h := New(root)
	if err := os.MkdirAll(h.root, 0755); err != nil {
		return Home{}, err
	}
	if err := os.MkdirAll(h.ScriptsDir(), 0755); err != nil {
		return Home{}, err
	}
	if err := os.MkdirAll(h.WorkspaceScriptsDir(), 0755); err != nil {
		return Home{}, err
	}
	if err := os.MkdirAll(h.RepoScriptsDir(), 0755); err != nil {
		return Home{}, err
	}
	if err := migrateLegacyScripts(h); err != nil {
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

// DBPath returns the on-disk path of the SQLite config database.
func (h Home) DBPath() string { return filepath.Join(h.root, "qail.db") }

// ScriptsDir returns the root directory holding user-managed bash scripts.
// Individual scripts live under the workspace/ or repo/ subdirs accessed
// via WorkspaceScriptsDir / RepoScriptsDir.
func (h Home) ScriptsDir() string { return filepath.Join(h.root, "scripts") }

// WorkspaceScriptsDir is the bucket for workspace-scoped post-install
// scripts.
func (h Home) WorkspaceScriptsDir() string {
	return filepath.Join(h.ScriptsDir(), "workspace")
}

// RepoScriptsDir is the bucket for repo-scoped post-install scripts.
func (h Home) RepoScriptsDir() string {
	return filepath.Join(h.ScriptsDir(), "repo")
}

// LegacyJSONPath returns the on-disk path of the pre-SQLite config.json,
// used only by the one-shot `qail config convert` migration.
func (h Home) LegacyJSONPath() string { return filepath.Join(h.root, "config.json") }

// migrateLegacyScripts moves any .sh file living at the root of
// ScriptsDir into WorkspaceScriptsDir. Predates the scope split, when
// scripts only ever ran as workspace post-install. Idempotent: if a
// file already exists at the destination it is left in place.
func migrateLegacyScripts(h Home) error {
	entries, err := os.ReadDir(h.ScriptsDir())
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".sh") {
			continue
		}
		src := filepath.Join(h.ScriptsDir(), e.Name())
		dst := filepath.Join(h.WorkspaceScriptsDir(), e.Name())
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		if err := os.Rename(src, dst); err != nil {
			return err
		}
	}
	return nil
}
