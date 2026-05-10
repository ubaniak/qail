package scripts

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ubaniak/qail/internal/runner"
)

func TestCreateBashScriptWritesFile(t *testing.T) {
	dir := t.TempDir()
	s := New(runner.NewRecorder(), dir)

	if err := s.CreateBashScript("deploy"); err != nil {
		t.Fatalf("CreateBashScript: %v", err)
	}

	path := filepath.Join(dir, "deploy.sh")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("script not created: %v", err)
	}
	if info.Mode().Perm()&0100 == 0 {
		t.Fatalf("script not executable: %v", info.Mode())
	}
}

func TestCreateBashScriptAppendsSh(t *testing.T) {
	dir := t.TempDir()
	s := New(runner.NewRecorder(), dir)

	if err := s.CreateBashScript("setup"); err != nil {
		t.Fatalf("CreateBashScript: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "setup.sh")); err != nil {
		t.Fatalf("expected setup.sh: %v", err)
	}
}

func TestCreateBashScriptKeepsExistingSuffix(t *testing.T) {
	dir := t.TempDir()
	s := New(runner.NewRecorder(), dir)

	if err := s.CreateBashScript("run.sh"); err != nil {
		t.Fatalf("CreateBashScript: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "run.sh")); err != nil {
		t.Fatalf("expected run.sh: %v", err)
	}
}

func TestCreateBashScriptRejectsDuplicate(t *testing.T) {
	dir := seedScripts(t, "exists.sh", "#!/bin/bash")
	s := New(runner.NewRecorder(), dir)

	if err := s.CreateBashScript("exists.sh"); err == nil {
		t.Fatal("expected error for duplicate script")
	}
}

func TestListScriptsReturnsSorted(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"z.sh", "a.sh", "m.sh"} {
		os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/bash"), 0755)
	}
	s := New(runner.NewRecorder(), dir)

	got, err := s.ListScripts()
	if err != nil {
		t.Fatalf("ListScripts: %v", err)
	}
	want := []string{"a.sh", "m.sh", "z.sh"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSortScripts(t *testing.T) {
	got := SortScripts([]string{"b", "a", "c"})
	want := []string{"a", "b", "c"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SortScripts[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestRemoveScriptDeletesFile(t *testing.T) {
	dir := seedScripts(t, "bye.sh", "#!/bin/bash")
	s := New(runner.NewRecorder(), dir)

	if err := s.RemoveScript("bye.sh"); err != nil {
		t.Fatalf("RemoveScript: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "bye.sh")); !os.IsNotExist(err) {
		t.Fatalf("script still exists after remove")
	}
}

func TestRemoveScriptRejectsPathTraversal(t *testing.T) {
	dir := seedScripts(t, "", "")
	s := New(runner.NewRecorder(), dir)

	if err := s.RemoveScript("../etc/passwd"); err == nil {
		t.Fatal("expected error for path traversal")
	}
}

func TestGetScriptDirEmptyReturnsError(t *testing.T) {
	s := New(runner.NewRecorder(), "")
	if _, err := s.GetScriptDir(); err == nil {
		t.Fatal("expected error for empty scriptsDir")
	}
}
