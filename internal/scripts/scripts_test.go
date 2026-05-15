package scripts

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ubaniak/qail/internal/runner"
)

// seedScripts creates an isolated scripts root under t.TempDir() with a
// workspace/ subdir and optionally writes scriptName under it. Returns
// the root directory.
func seedScripts(t *testing.T, scriptName, scriptBody string) string {
	t.Helper()
	dir := t.TempDir()
	scopeDir := filepath.Join(dir, string(ScopeWorkspace))
	if err := os.MkdirAll(scopeDir, 0755); err != nil {
		t.Fatalf("mkdir scope: %v", err)
	}
	if scriptName != "" {
		if err := os.WriteFile(filepath.Join(scopeDir, scriptName), []byte(scriptBody), 0755); err != nil {
			t.Fatalf("write script: %v", err)
		}
	}
	return dir
}

func TestRunBashScriptInvokesBashWithScriptPath(t *testing.T) {
	dir := seedScripts(t, "hello.sh", "echo hi")

	rec := runner.NewRecorder().RespondOK([]byte("hi\n"))
	s := New(rec, dir)

	if err := s.RunBashScript(context.Background(), ScopeWorkspace, "hello.sh", "/tmp/work", nil); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if len(rec.Calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(rec.Calls))
	}
	got := rec.LastCall()
	if got.Name != "/bin/bash" {
		t.Fatalf("name = %q, want /bin/bash", got.Name)
	}
	wantPath := filepath.Join(dir, string(ScopeWorkspace), "hello.sh")
	if len(got.Args) != 1 || got.Args[0] != wantPath {
		t.Fatalf("args = %v, want [%s]", got.Args, wantPath)
	}
	if got.Dir != "/tmp/work" {
		t.Fatalf("dir = %q, want /tmp/work", got.Dir)
	}
}

func TestRunBashScriptRejectsPathTraversal(t *testing.T) {
	dir := seedScripts(t, "", "")

	rec := runner.NewRecorder()
	s := New(rec, dir)

	err := s.RunBashScript(context.Background(), ScopeWorkspace, "../etc/passwd", "/tmp", nil)
	if err == nil {
		t.Fatalf("expected error for path traversal")
	}
	if len(rec.Calls) != 0 {
		t.Fatalf("runner was called for invalid path; calls = %d", len(rec.Calls))
	}
}
