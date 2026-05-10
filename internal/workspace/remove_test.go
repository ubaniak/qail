package workspace

import "testing"

func TestRemoveCallsRemoveAllOnWorkspacePath(t *testing.T) {
	fsys := newMemFS()
	fsys.dirs["/qroot"] = true
	fsys.dirs["/qroot/alpha"] = true

	w := New("/qroot", "alpha", nil, nil, &fakeInstaller{failOnIndex: -1}, fsys)
	if err := w.Remove(); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if len(fsys.removed) != 1 || fsys.removed[0] != "/qroot/alpha" {
		t.Fatalf("removed = %v, want [/qroot/alpha]", fsys.removed)
	}
}

func TestRemoveRepoRemovesSpecificRepo(t *testing.T) {
	fsys := newMemFS()
	fsys.dirs["/qroot/alpha/svc-a"] = true

	w := New("/qroot", "alpha", nil, nil, &fakeInstaller{failOnIndex: -1}, fsys)
	if err := w.RemoveRepo("svc-a"); err != nil {
		t.Fatalf("RemoveRepo: %v", err)
	}
	if len(fsys.removed) != 1 || fsys.removed[0] != "/qroot/alpha/svc-a" {
		t.Fatalf("removed = %v, want [/qroot/alpha/svc-a]", fsys.removed)
	}
}
