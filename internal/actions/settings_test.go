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

func TestAddEditorRegistersAndSetsFirstAsDefault(t *testing.T) {
	s := config.NewMemoryStore()
	if err := AddEditor(s, "vscode", "code"); err != nil {
		t.Fatalf("AddEditor: %v", err)
	}
	cfg, _ := s.Read()
	if len(cfg.Editors) != 1 || cfg.Editors[0].Name != "vscode" || cfg.Editors[0].Command != "code" {
		t.Fatalf("Editors = %+v", cfg.Editors)
	}
	if cfg.DefaultEditor != "vscode" {
		t.Fatalf("DefaultEditor = %q, want vscode (first registered)", cfg.DefaultEditor)
	}
}

func TestAddEditorRejectsDuplicate(t *testing.T) {
	s := config.NewMemoryStoreFrom(config.Config{
		Editors: []config.Editor{{Name: "vim", Command: "vim"}},
	})
	if err := AddEditor(s, "vim", "nvim"); err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestAddEditorRejectsEmptyFields(t *testing.T) {
	s := config.NewMemoryStore()
	if err := AddEditor(s, "", "code"); err == nil {
		t.Fatal("expected empty-name error")
	}
	if err := AddEditor(s, "x", ""); err == nil {
		t.Fatal("expected empty-command error")
	}
}

func TestRemoveEditorClearsDefaultAndWorkspaceRefs(t *testing.T) {
	s := config.NewMemoryStoreFrom(config.Config{
		Editors:       []config.Editor{{Name: "vim", Command: "vim"}, {Name: "vscode", Command: "code"}},
		DefaultEditor: "vim",
		Workspaces: config.Workspace{
			"alpha": {Repos: []string{"a"}, Editor: "vim"},
			"beta":  {Repos: []string{"b"}, Editor: "vscode"},
		},
	})
	if err := RemoveEditor(s, "vim"); err != nil {
		t.Fatalf("RemoveEditor: %v", err)
	}
	cfg, _ := s.Read()
	if len(cfg.Editors) != 1 || cfg.Editors[0].Name != "vscode" {
		t.Fatalf("Editors = %+v", cfg.Editors)
	}
	if cfg.DefaultEditor != "" {
		t.Fatalf("DefaultEditor = %q, want empty after removal", cfg.DefaultEditor)
	}
	if cfg.Workspaces["alpha"].Editor != "" {
		t.Fatalf("alpha.Editor = %q, want cleared", cfg.Workspaces["alpha"].Editor)
	}
	if cfg.Workspaces["beta"].Editor != "vscode" {
		t.Fatalf("beta.Editor = %q, want untouched", cfg.Workspaces["beta"].Editor)
	}
}

func TestSetDefaultEditorValidatesName(t *testing.T) {
	s := config.NewMemoryStoreFrom(config.Config{
		Editors: []config.Editor{{Name: "vim", Command: "vim"}},
	})
	if err := SetDefaultEditor(s, "ghost"); err == nil {
		t.Fatal("expected error for unknown editor")
	}
	if err := SetDefaultEditor(s, "vim"); err != nil {
		t.Fatalf("SetDefaultEditor: %v", err)
	}
	cfg, _ := s.Read()
	if cfg.DefaultEditor != "vim" {
		t.Fatalf("DefaultEditor = %q", cfg.DefaultEditor)
	}
}

func TestSetWorkspaceEditorAndUnset(t *testing.T) {
	s := config.NewMemoryStoreFrom(config.Config{
		Editors: []config.Editor{{Name: "vim", Command: "vim"}},
		Workspaces: config.Workspace{
			"alpha": {Repos: []string{"a"}},
		},
	})
	if err := SetWorkspaceEditor(s, "alpha", "vim"); err != nil {
		t.Fatalf("SetWorkspaceEditor: %v", err)
	}
	cfg, _ := s.Read()
	if cfg.Workspaces["alpha"].Editor != "vim" {
		t.Fatalf("alpha.Editor = %q", cfg.Workspaces["alpha"].Editor)
	}
	// Unset by passing empty string.
	if err := SetWorkspaceEditor(s, "alpha", ""); err != nil {
		t.Fatalf("SetWorkspaceEditor unset: %v", err)
	}
	cfg, _ = s.Read()
	if cfg.Workspaces["alpha"].Editor != "" {
		t.Fatalf("alpha.Editor after unset = %q", cfg.Workspaces["alpha"].Editor)
	}
}

func TestSetWorkspaceEditorRejectsUnknownEditor(t *testing.T) {
	s := config.NewMemoryStoreFrom(config.Config{
		Workspaces: config.Workspace{"alpha": {}},
	})
	if err := SetWorkspaceEditor(s, "alpha", "ghost"); err == nil {
		t.Fatal("expected error for unknown editor")
	}
}
