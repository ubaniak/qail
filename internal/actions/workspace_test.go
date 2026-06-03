package actions

import (
	"errors"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/ubaniak/qail/internal/config"
)

// fakeFileInfo is the minimum os.FileInfo restore tests need: a name
// plus an IsDir bool. Other fields stay zero — restore only checks
// IsDir().
type fakeFileInfo struct {
	name string
	dir  bool
}

func (f fakeFileInfo) Name() string       { return f.name }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() os.FileMode  { return 0 }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return f.dir }
func (f fakeFileInfo) Sys() any           { return nil }

// withFakeStat swaps osStat for the duration of the test so restore
// can be exercised without touching the real filesystem.
func withFakeStat(t *testing.T, exists, isDir bool) {
	t.Helper()
	orig := osStat
	t.Cleanup(func() { osStat = orig })
	osStat = func(name string) (os.FileInfo, error) {
		if !exists {
			return nil, &os.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
		}
		return fakeFileInfo{name: name, dir: isDir}, nil
	}
}

// Note: AddWorkspace, EditWorkspace, CloneWorkspace, CreateWorkspaceOnDisk
// all call workspace.Create which touches disk and runs git/scripts. They
// are exercised end-to-end via the cmd handlers, not unit-tested here.
// The tests below cover the pure registry mutations.

func TestRemoveWorkspacePreservesPostInstallForRestore(t *testing.T) {
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
	got := cfg.PostInstallScripts.Workspace["alpha"]
	if len(got) != 1 || got[0] != "setup.sh" {
		t.Errorf("PostInstallScripts.Workspace[alpha] = %v, want preserved for restore", got)
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

func TestRestoreWorkspaceRegistersWithRepos(t *testing.T) {
	withFakeStat(t, true, true)
	s := config.NewMemoryStoreFrom(config.Config{
		Root: "/q",
		Repos: map[string]string{
			"alpha": "git@x:alpha",
			"beta":  "git@x:beta",
		},
		PostInstallScripts: config.PostInstallScripts{
			Workspace: map[string][]string{"recovered": {"setup.sh"}},
		},
	})

	if err := RestoreWorkspace(s, "recovered", []string{"alpha", "beta"}); err != nil {
		t.Fatalf("RestoreWorkspace: %v", err)
	}

	cfg, _ := s.Read()
	profile, ok := cfg.Workspaces["recovered"]
	if !ok {
		t.Fatal("recovered not registered")
	}
	if len(profile.Repos) != 2 || profile.Repos[0] != "alpha" || profile.Repos[1] != "beta" {
		t.Errorf("Repos = %v", profile.Repos)
	}
	if profile.LastUsed.IsZero() {
		t.Error("LastUsed not stamped")
	}
	// preserved scripts auto-rebound
	if got := cfg.PostInstallScripts.Workspace["recovered"]; len(got) != 1 || got[0] != "setup.sh" {
		t.Errorf("preserved scripts = %v", got)
	}
}

func TestRestoreWorkspaceRejectsUnknownRepos(t *testing.T) {
	withFakeStat(t, true, true)
	s := config.NewMemoryStoreFrom(config.Config{
		Root:  "/q",
		Repos: map[string]string{"alpha": "git@x:alpha"},
	})
	if err := RestoreWorkspace(s, "ws", []string{"alpha", "ghost"}); err == nil {
		t.Fatal("expected error for unknown repo")
	}
	cfg, _ := s.Read()
	if _, ok := cfg.Workspaces["ws"]; ok {
		t.Error("ws registered despite invalid repo")
	}
}

func TestRestoreWorkspaceRejectsDuplicate(t *testing.T) {
	withFakeStat(t, true, true)
	s := config.NewMemoryStoreFrom(config.Config{
		Root:       "/q",
		Workspaces: config.Workspace{"existing": {Repos: []string{"alpha"}}},
		Repos:      map[string]string{"alpha": "git@x"},
	})
	if err := RestoreWorkspace(s, "existing", []string{"alpha"}); err == nil {
		t.Fatal("expected error for duplicate workspace")
	}
}

func TestRestoreWorkspaceRejectsMissingDir(t *testing.T) {
	withFakeStat(t, false, false)
	s := config.NewMemoryStoreFrom(config.Config{
		Root:  "/q",
		Repos: map[string]string{"alpha": "git@x"},
	})
	err := RestoreWorkspace(s, "ws", []string{"alpha"})
	if err == nil || !errors.Is(err, err) || err.Error() == "" {
		t.Fatalf("expected non-empty error, got %v", err)
	}
}

func TestRemoveOrphanWorkspacesDropsPreservedScripts(t *testing.T) {
	// Use a tempdir as cfg.Root so the real RemoveOrphans call has
	// something to delete; no go-git involvement here.
	root := t.TempDir()
	if err := os.MkdirAll(root+"/ghost", 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	s := config.NewMemoryStoreFrom(config.Config{
		Root: root,
		PostInstallScripts: config.PostInstallScripts{
			Workspace: map[string][]string{"ghost": {"setup.sh"}},
		},
	})
	if err := RemoveOrphanWorkspaces(s, []string{"ghost"}); err != nil {
		t.Fatalf("RemoveOrphanWorkspaces: %v", err)
	}
	cfg, _ := s.Read()
	if _, ok := cfg.PostInstallScripts.Workspace["ghost"]; ok {
		t.Error("preserved scripts not gc'd after orphan purge")
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
