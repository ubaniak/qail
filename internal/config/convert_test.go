package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConvertJSONRoundTrips(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "config.json")

	jsonBody := `{
		"root": "/work",
		"editor": "code",
		"repos": {"svc-a": "git@a", "svc-b": "git@b"},
		"workspaces": {
			"team-x": {
				"repos": ["svc-a", "svc-b"],
				"last_used": "2026-01-15T10:00:00Z"
			}
		},
		"post_install_scripts": {
			"repo": {"svc-a": ["bootstrap.sh"]},
			"workspace": {"team-x": ["setup.sh"]}
		}
	}`
	if err := os.WriteFile(jsonPath, []byte(jsonBody), 0644); err != nil {
		t.Fatalf("write json: %v", err)
	}

	dbPath := filepath.Join(dir, "qail.db")
	s, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}

	if err := s.ConvertJSON(jsonPath); err != nil {
		t.Fatalf("ConvertJSON: %v", err)
	}

	cfg, err := s.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if cfg.Root != "/work" {
		t.Fatalf("Root = %q, want /work", cfg.Root)
	}
	if cfg.Editor != "code" {
		t.Fatalf("Editor = %q, want code", cfg.Editor)
	}
	if cfg.Repos["svc-a"] != "git@a" {
		t.Fatalf("Repos[svc-a] = %q", cfg.Repos["svc-a"])
	}

	ws, ok := cfg.Workspaces["team-x"]
	if !ok {
		t.Fatalf("workspace team-x missing")
	}
	wantTime := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	if !ws.LastUsed.Equal(wantTime) {
		t.Fatalf("LastUsed = %v, want %v", ws.LastUsed, wantTime)
	}

	if cfg.PostInstallScripts.Repo["svc-a"][0] != "bootstrap.sh" {
		t.Fatalf("repo post-install = %v", cfg.PostInstallScripts.Repo["svc-a"])
	}
	if cfg.PostInstallScripts.Workspace["team-x"][0] != "setup.sh" {
		t.Fatalf("ws post-install = %v", cfg.PostInstallScripts.Workspace["team-x"])
	}
}

func TestConvertJSONDefaultsZeroLastUsed(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "config.json")

	jsonBody := `{
		"workspaces": {
			"alpha": {"repos": ["a"]}
		}
	}`
	if err := os.WriteFile(jsonPath, []byte(jsonBody), 0644); err != nil {
		t.Fatalf("write json: %v", err)
	}

	dbPath := filepath.Join(dir, "qail.db")
	s, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}

	before := time.Now().UTC()
	if err := s.ConvertJSON(jsonPath); err != nil {
		t.Fatalf("ConvertJSON: %v", err)
	}

	cfg, _ := s.Read()
	ws := cfg.Workspaces["alpha"]
	if ws.LastUsed.Before(before) {
		t.Fatalf("LastUsed = %v, want >= %v", ws.LastUsed, before)
	}
}

func TestConvertJSONInvalidPath(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "qail.db")
	s, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}

	if err := s.ConvertJSON("/nonexistent/config.json"); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestBackUpAndRestore(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "qail.db")
	s, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}

	// Write initial config
	original := Config{Root: "/original", Editor: "vim"}
	if err := s.Write(original); err != nil {
		t.Fatalf("Write original: %v", err)
	}

	// Back up
	if err := s.BackUp(); err != nil {
		t.Fatalf("BackUp: %v", err)
	}
	if _, err := os.Stat(dbPath + ".bak"); err != nil {
		t.Fatalf("backup file missing: %v", err)
	}

	// Overwrite with different config
	modified := Config{Root: "/modified", Editor: "code"}
	if err := s.Write(modified); err != nil {
		t.Fatalf("Write modified: %v", err)
	}

	// Restore from backup
	if err := s.Restore(); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	// Re-open store to read restored data
	s2, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore after restore: %v", err)
	}
	cfg, err := s2.Read()
	if err != nil {
		t.Fatalf("Read after restore: %v", err)
	}
	if cfg.Root != "/original" {
		t.Fatalf("Root = %q after restore, want /original", cfg.Root)
	}
}
