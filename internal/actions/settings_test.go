package actions

import (
	"testing"

	"github.com/ubaniak/qail/internal/config"
)

func TestSetRoot(t *testing.T) {
	s := config.NewMemoryStore()
	if err := SetRoot(s, "/work"); err != nil {
		t.Fatalf("SetRoot: %v", err)
	}
	cfg, _ := s.Read()
	if cfg.Root != "/work" {
		t.Fatalf("Root = %q, want /work", cfg.Root)
	}
}

func TestSetEditor(t *testing.T) {
	s := config.NewMemoryStoreFrom(config.Config{Editor: "vim"})
	if err := SetEditor(s, "code"); err != nil {
		t.Fatalf("SetEditor: %v", err)
	}
	cfg, _ := s.Read()
	if cfg.Editor != "code" {
		t.Fatalf("Editor = %q, want code", cfg.Editor)
	}
}

func TestSetEditorPreservesOtherFields(t *testing.T) {
	s := config.NewMemoryStoreFrom(config.Config{
		Root:   "/work",
		Editor: "vim",
		Repos:  map[string]string{"svc-a": "url"},
	})
	if err := SetEditor(s, "code"); err != nil {
		t.Fatalf("SetEditor: %v", err)
	}
	cfg, _ := s.Read()
	if cfg.Root != "/work" {
		t.Fatalf("Root lost: %q", cfg.Root)
	}
	if cfg.Repos["svc-a"] != "url" {
		t.Fatalf("Repos lost: %v", cfg.Repos)
	}
}
