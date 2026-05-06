package git

import (
	"testing"

	"github.com/ubaniak/qail/internal/runner"
)

func TestCloneInvokesGitWithCorrectArgs(t *testing.T) {
	rec := runner.NewRecorder()
	g := New(rec)

	_, err := g.Clone("git@github.com:foo/bar.git", "/tmp/bar")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if len(rec.Calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(rec.Calls))
	}
	got := rec.LastCall()
	if got.Name != "git" {
		t.Fatalf("name = %q, want git", got.Name)
	}
	want := []string{"clone", "git@github.com:foo/bar.git", "/tmp/bar"}
	if len(got.Args) != len(want) {
		t.Fatalf("args = %v, want %v", got.Args, want)
	}
	for i, a := range want {
		if got.Args[i] != a {
			t.Fatalf("args[%d] = %q, want %q", i, got.Args[i], a)
		}
	}
}
