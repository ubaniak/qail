package scripts

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ubaniak/qail/internal/runner"
)

// withTempHome points os.UserHomeDir() at a tempdir for the duration of the
// test by overriding HOME, and pre-creates ~/.qail/scripts/<file>.
func withTempHome(t *testing.T, scriptName, scriptBody string) (home, scriptsDir string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)

	scriptsDir = filepath.Join(home, ".qail", "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if scriptName != "" {
		if err := os.WriteFile(filepath.Join(scriptsDir, scriptName), []byte(scriptBody), 0755); err != nil {
			t.Fatalf("write script: %v", err)
		}
	}
	return home, scriptsDir
}

func TestRunBashScriptInvokesBashWithScriptPath(t *testing.T) {
	_, scriptsDir := withTempHome(t, "hello.sh", "echo hi")

	rec := runner.NewRecorder().RespondOK([]byte("hi\n"))
	s := New(rec)

	if err := s.RunBashScript("hello.sh", "/tmp/work"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if len(rec.Calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(rec.Calls))
	}
	got := rec.LastCall()
	if got.Name != "/bin/bash" {
		t.Fatalf("name = %q, want /bin/bash", got.Name)
	}
	wantPath := filepath.Join(scriptsDir, "hello.sh")
	if len(got.Args) != 1 || got.Args[0] != wantPath {
		t.Fatalf("args = %v, want [%s]", got.Args, wantPath)
	}
	if got.Dir != "/tmp/work" {
		t.Fatalf("dir = %q, want /tmp/work", got.Dir)
	}
}

func TestRunBashScriptRejectsPathTraversal(t *testing.T) {
	withTempHome(t, "", "")

	rec := runner.NewRecorder()
	s := New(rec)

	err := s.RunBashScript("../etc/passwd", "/tmp")
	if err == nil {
		t.Fatalf("expected error for path traversal")
	}
	if len(rec.Calls) != 0 {
		t.Fatalf("runner was called for invalid path; calls = %d", len(rec.Calls))
	}
}
