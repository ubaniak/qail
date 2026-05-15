package actions

import (
	"os"
	"path/filepath"
	"testing"
)

// seedScopedScripts points $QAIL_HOME at t.TempDir() and writes the
// named scripts under scripts/<scope>/. Used by action tests that need
// the post-install validators (Has / Set*PostInstall) to find the
// referenced scripts on disk.
func seedScopedScripts(t *testing.T, scope string, names ...string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("QAIL_HOME", home)
	scopeDir := filepath.Join(home, "scripts", scope)
	if err := os.MkdirAll(scopeDir, 0755); err != nil {
		t.Fatalf("mkdir scope: %v", err)
	}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(scopeDir, n), []byte("#!/bin/bash\n"), 0755); err != nil {
			t.Fatalf("write %s: %v", n, err)
		}
	}
	return home
}
