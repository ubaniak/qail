package qailhome

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewExposesAllPaths(t *testing.T) {
	h := New("/opt/qail")
	if h.DBPath() != "/opt/qail/qail.db" {
		t.Fatalf("DBPath = %q", h.DBPath())
	}
	if h.ScriptsDir() != "/opt/qail/scripts" {
		t.Fatalf("ScriptsDir = %q", h.ScriptsDir())
	}
	if h.LegacyJSONPath() != "/opt/qail/config.json" {
		t.Fatalf("LegacyJSONPath = %q", h.LegacyJSONPath())
	}
}

func TestDefaultUsesQAIL_HOME(t *testing.T) {
	dir := t.TempDir()
	override := filepath.Join(dir, "alt-home")
	t.Setenv("QAIL_HOME", override)

	h, err := Default()
	if err != nil {
		t.Fatalf("Default: %v", err)
	}
	// Home root + ScriptsDir created on disk; verified via the accessors.
	if _, err := os.Stat(filepath.Dir(h.DBPath())); err != nil {
		t.Fatalf("home root not created: %v", err)
	}
	if _, err := os.Stat(h.ScriptsDir()); err != nil {
		t.Fatalf("ScriptsDir not created: %v", err)
	}
	if h.DBPath() != filepath.Join(override, "qail.db") {
		t.Fatalf("DBPath = %q", h.DBPath())
	}
}

func TestDefaultFallsBackToUserHome(t *testing.T) {
	t.Setenv("QAIL_HOME", "")
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	h, err := Default()
	if err != nil {
		t.Fatalf("Default: %v", err)
	}
	want := filepath.Join(dir, ".qail", "qail.db")
	if h.DBPath() != want {
		t.Fatalf("DBPath = %q, want %q", h.DBPath(), want)
	}
}
