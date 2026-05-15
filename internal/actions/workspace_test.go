package actions

import (
	"testing"
	"time"

	"github.com/ubaniak/qail/internal/config"
)

// Note: AddWorkspace, EditWorkspace, CloneWorkspace, CreateWorkspaceOnDisk
// all call workspace.Create which touches disk and runs git/scripts. They
// are exercised end-to-end via the cmd handlers, not unit-tested here.
// The tests below cover the pure registry mutations.

func TestRemoveWorkspaceStripsRegistryAndPostInstall(t *testing.T) {
	s := config.NewMemoryStoreFrom(config.Config{
		Workspaces: config.Workspace{
			"alpha": config.WorkspaceProfile{Repos: []string{"a"}, LastUsed: time.Now()},
			"beta":  config.WorkspaceProfile{Repos: []string{"b"}, LastUsed: time.Now()},
		},
		PostInstallScripts: config.PostInstallScripts{
			Workspace: map[string][]string{
				"alpha": {"setup.sh"},
				"beta":  {"setup.sh"},
			},
		},
	})

	if err := RemoveWorkspace(s, "alpha"); err != nil {
		t.Fatalf("RemoveWorkspace: %v", err)
	}

	cfg, _ := s.Read()
	if _, ok := cfg.Workspaces["alpha"]; ok {
		t.Errorf("Workspaces[alpha] not deleted")
	}
	if _, ok := cfg.PostInstallScripts.Workspace["alpha"]; ok {
		t.Errorf("PostInstallScripts.Workspace[alpha] not deleted")
	}
	if _, ok := cfg.Workspaces["beta"]; !ok {
		t.Errorf("Workspaces[beta] removed by mistake")
	}
}

func TestRemoveWorkspaceEmptyNameNoOp(t *testing.T) {
	s := config.NewMemoryStoreFrom(config.Config{
		Workspaces: config.Workspace{"alpha": {}},
	})
	if err := RemoveWorkspace(s, ""); err != nil {
		t.Fatalf("RemoveWorkspace empty: %v", err)
	}
	cfg, _ := s.Read()
	if _, ok := cfg.Workspaces["alpha"]; !ok {
		t.Errorf("alpha removed despite empty name")
	}
}

func TestTouchWorkspaceUpdatesLastUsed(t *testing.T) {
	old := time.Now().Add(-24 * time.Hour)
	s := config.NewMemoryStoreFrom(config.Config{
		Workspaces: config.Workspace{
			"alpha": config.WorkspaceProfile{Repos: []string{"a", "b"}, LastUsed: old},
		},
	})

	profile, err := TouchWorkspace(s, "alpha")
	if err != nil {
		t.Fatalf("TouchWorkspace: %v", err)
	}
	if !profile.LastUsed.Equal(old) {
		t.Errorf("returned profile.LastUsed = %v, want pre-touch %v", profile.LastUsed, old)
	}

	cfg, _ := s.Read()
	stored := cfg.Workspaces["alpha"]
	if !stored.LastUsed.After(old) {
		t.Errorf("stored LastUsed = %v, want > %v", stored.LastUsed, old)
	}
	if len(stored.Repos) != 2 || stored.Repos[0] != "a" {
		t.Errorf("Repos drift: %v", stored.Repos)
	}
}

func TestTouchWorkspaceMissing(t *testing.T) {
	s := config.NewMemoryStore()
	if _, err := TouchWorkspace(s, "ghost"); err == nil {
		t.Fatal("expected error for missing workspace")
	}
}

func TestSetWorkspacePostInstallReplacesAndClears(t *testing.T) {
	seedScopedScripts(t, "workspace", "new.sh", "extra.sh")
	s := config.NewMemoryStoreFrom(config.Config{
		PostInstallScripts: config.PostInstallScripts{
			Workspace: map[string][]string{"alpha": {"old.sh"}},
		},
	})

	if err := SetWorkspacePostInstall(s, "alpha", []string{"new.sh", "extra.sh"}); err != nil {
		t.Fatalf("SetWorkspacePostInstall: %v", err)
	}
	cfg, _ := s.Read()
	got := cfg.PostInstallScripts.Workspace["alpha"]
	if len(got) != 2 || got[0] != "new.sh" || got[1] != "extra.sh" {
		t.Errorf("scripts = %v", got)
	}

	// Empty slice clears the entry.
	if err := SetWorkspacePostInstall(s, "alpha", nil); err != nil {
		t.Fatalf("SetWorkspacePostInstall clear: %v", err)
	}
	cfg, _ = s.Read()
	if _, ok := cfg.PostInstallScripts.Workspace["alpha"]; ok {
		t.Errorf("entry not cleared")
	}
}

func TestSetWorkspacePostInstallEmptyName(t *testing.T) {
	s := config.NewMemoryStore()
	if err := SetWorkspacePostInstall(s, "", []string{"a.sh"}); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestListWorkspacesReadOnly(t *testing.T) {
	s := config.NewMemoryStoreFrom(config.Config{
		Workspaces: config.Workspace{
			"alpha": config.WorkspaceProfile{Repos: []string{"a"}},
		},
		PostInstallScripts: config.PostInstallScripts{
			Workspace: map[string][]string{"alpha": {"x.sh"}},
		},
	})

	ws, post, err := ListWorkspaces(s)
	if err != nil {
		t.Fatalf("ListWorkspaces: %v", err)
	}
	if _, ok := ws["alpha"]; !ok {
		t.Errorf("alpha missing")
	}
	if post["alpha"][0] != "x.sh" {
		t.Errorf("post-install missing")
	}
}

func TestReadWorkspaceContextReturnsCfgFields(t *testing.T) {
	s := config.NewMemoryStoreFrom(config.Config{
		Root: "/q",
		Workspaces: config.Workspace{
			"alpha": config.WorkspaceProfile{Repos: []string{"a"}},
		},
	})
	ctx, err := ReadWorkspaceContext(s)
	if err != nil {
		t.Fatalf("ReadWorkspaceContext: %v", err)
	}
	if ctx.Root != "/q" {
		t.Errorf("ctx = %+v", ctx)
	}
	if _, ok := ctx.Workspaces["alpha"]; !ok {
		t.Errorf("workspaces missing")
	}
}
