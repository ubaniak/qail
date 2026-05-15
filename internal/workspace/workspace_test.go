package workspace

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/ubaniak/qail/internal/installer"
	"github.com/ubaniak/qail/internal/scripts"
)

// fakeInstaller records every Install / RunPostInstall call. Install can be
// configured to fail at a specific package to exercise rollback.
type fakeInstaller struct {
	specs       []installer.PackageSpec
	postCalls   []postCall
	failOnIndex int
	failOnDir   string
}

type postCall struct {
	scope   scripts.Scope
	dir     string
	scripts []string
}

func (f *fakeInstaller) Install(_ context.Context, spec installer.PackageSpec, _ io.Writer) error {
	if f.failOnIndex >= 0 && len(f.specs) == f.failOnIndex {
		f.specs = append(f.specs, spec)
		return errors.New("boom")
	}
	f.specs = append(f.specs, spec)
	return nil
}

func (f *fakeInstaller) RunPostInstall(_ context.Context, scope scripts.Scope, dir string, names []string, _ io.Writer) error {
	if dir == f.failOnDir {
		return errors.New("boom-post")
	}
	f.postCalls = append(f.postCalls, postCall{scope: scope, dir: dir, scripts: names})
	return nil
}

// memFS satisfies FS using an in-memory map. Only the operations Workspace
// exercises are implemented.
type memFS struct {
	dirs    map[string]bool
	removed []string
	entries map[string][]os.DirEntry
}

func newMemFS() *memFS {
	return &memFS{dirs: map[string]bool{}, entries: map[string][]os.DirEntry{}}
}

func (m *memFS) Stat(name string) (os.FileInfo, error) {
	if m.dirs[name] {
		return memInfo{name: name, dir: true}, nil
	}
	return nil, &os.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
}

func (m *memFS) Mkdir(path string, _ os.FileMode) error {
	m.dirs[path] = true
	return nil
}

func (m *memFS) RemoveAll(path string) error {
	m.removed = append(m.removed, path)
	delete(m.dirs, path)
	return nil
}

func (m *memFS) ReadDir(name string) ([]os.DirEntry, error) {
	if e, ok := m.entries[name]; ok {
		return e, nil
	}
	return nil, &os.PathError{Op: "readdir", Path: name, Err: fs.ErrNotExist}
}

type memInfo struct {
	name string
	dir  bool
}

func (i memInfo) Name() string       { return i.name }
func (i memInfo) Size() int64        { return 0 }
func (i memInfo) Mode() os.FileMode  { return 0 }
func (i memInfo) ModTime() time.Time { return time.Time{} }
func (i memInfo) IsDir() bool        { return i.dir }
func (i memInfo) Sys() any           { return nil }

type memEntry struct {
	name string
	dir  bool
}

func (e memEntry) Name() string               { return e.name }
func (e memEntry) IsDir() bool                { return e.dir }
func (e memEntry) Type() os.FileMode          { return 0 }
func (e memEntry) Info() (os.FileInfo, error) { return memInfo{name: e.name, dir: e.dir}, nil }

func TestCreateInvokesInstallerForEachPackage(t *testing.T) {
	inst := &fakeInstaller{failOnIndex: -1}
	fsys := newMemFS()
	w := New(
		"/qroot",
		"alpha",
		[]string{"a", "b"},
		map[string]string{"a": "git@a", "b": "git@b"},
		inst,
		fsys,
	)
	if err := w.Create(context.Background(), io.Discard); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got := len(inst.specs); got != 2 {
		t.Fatalf("specs len = %d, want 2", got)
	}
	if inst.specs[0].Name != "a" || inst.specs[0].RepoURL != "git@a" {
		t.Errorf("first spec = %+v", inst.specs[0])
	}
	if inst.specs[1].Dest != "/qroot/alpha/b" {
		t.Errorf("second spec dest = %s", inst.specs[1].Dest)
	}
}

func TestCreateRollsBackWorkspaceDirOnFailure(t *testing.T) {
	inst := &fakeInstaller{failOnIndex: 1} // fail on second package
	fsys := newMemFS()
	w := New(
		"/qroot",
		"alpha",
		[]string{"a", "b"},
		map[string]string{"a": "git@a", "b": "git@b"},
		inst,
		fsys,
	)
	err := w.Create(context.Background(), io.Discard)
	if err == nil {
		t.Fatal("Create: want error, got nil")
	}
	// /qroot/alpha was created by us, so it should be torn down.
	wantRemoved := "/qroot/alpha"
	found := false
	for _, p := range fsys.removed {
		if p == wantRemoved {
			found = true
		}
	}
	if !found {
		t.Errorf("expected RemoveAll(%s); removed = %v", wantRemoved, fsys.removed)
	}
}

func TestCleanIgnoresTrackedWorkspaces(t *testing.T) {
	fsys := newMemFS()
	fsys.entries["/qroot"] = []os.DirEntry{
		memEntry{name: "tracked", dir: true},
		memEntry{name: "orphan", dir: true},
		memEntry{name: "file.txt", dir: false},
	}
	tracked := map[string]struct {
		Repos    []string
		LastUsed time.Time
	}{}
	tracked["tracked"] = struct {
		Repos    []string
		LastUsed time.Time
	}{}

	// Build a real config.Workspace via the package's type. Inline literal
	// because this test lives in the workspace package, not config.
	ws := buildTrackedWorkspace("tracked")

	confirm := func(string) (bool, error) { return true, nil }
	if err := cleanWithDeps(fsys, confirm, "/qroot", ws, io.Discard); err != nil {
		t.Fatalf("clean: %v", err)
	}
	if len(fsys.removed) != 1 || fsys.removed[0] != "/qroot/orphan" {
		t.Errorf("removed = %v, want [/qroot/orphan]", fsys.removed)
	}
}

func TestCleanRespectsConfirmFalse(t *testing.T) {
	fsys := newMemFS()
	fsys.entries["/qroot"] = []os.DirEntry{
		memEntry{name: "orphan", dir: true},
	}
	confirm := func(string) (bool, error) { return false, nil }
	if err := cleanWithDeps(fsys, confirm, "/qroot", buildTrackedWorkspace(), io.Discard); err != nil {
		t.Fatalf("clean: %v", err)
	}
	if len(fsys.removed) != 0 {
		t.Errorf("removed = %v, want none", fsys.removed)
	}
}
