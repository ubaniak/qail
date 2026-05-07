package qailhome

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewExposesAllPaths(t *testing.T) {
	h := New("/opt/qail")
	if h.Root() != "/opt/qail" {
		t.Fatalf("Root = %q", h.Root())
	}
	if h.DBPath() != "/opt/qail/qail.db" {
		t.Fatalf("DBPath = %q", h.DBPath())
	}
	if h.ScriptsDir() != "/opt/qail/scripts" {
		t.Fatalf("ScriptsDir = %q", h.ScriptsDir())
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
	if h.Root() != override {
		t.Fatalf("Root = %q, want %q", h.Root(), override)
	}
	// Root + ScriptsDir created on disk.
	if _, err := os.Stat(h.Root()); err != nil {
		t.Fatalf("Root not created: %v", err)
	}
	if _, err := os.Stat(h.ScriptsDir()); err != nil {
		t.Fatalf("ScriptsDir not created: %v", err)
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
	want := filepath.Join(dir, ".qail")
	if h.Root() != want {
		t.Fatalf("Root = %q, want %q", h.Root(), want)
	}
}
