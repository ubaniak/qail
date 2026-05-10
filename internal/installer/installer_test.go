package installer

import (
	"context"
	"errors"
	"io"
	"testing"
)

type fakeGit struct {
	calls  []string
	err    error
	called bool
}

func (f *fakeGit) Clone(_ context.Context, repo, path, _ string, _ io.Writer) error {
	f.called = true
	f.calls = append(f.calls, "clone "+repo+" "+path)
	return f.err
}

type fakeScripts struct {
	calls []string
	err   error
}

func (f *fakeScripts) RunBashScript(_ context.Context, script, dir string, _ io.Writer) error {
	f.calls = append(f.calls, "script "+script+" "+dir)
	return f.err
}

func TestInstallClonesThenRunsScripts(t *testing.T) {
	g := &fakeGit{}
	s := &fakeScripts{}
	i := New(g, s)

	err := i.Install(context.Background(), PackageSpec{
		Name:        "svc-a",
		RepoURL:     "git@example.com:foo/svc-a.git",
		Dest:        "/work/svc-a",
		PostInstall: []string{"bootstrap.sh"},
	}, io.Discard)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	// Verify ordering: clone first, then script.
	if len(g.calls) != 1 {
		t.Fatalf("git calls = %d, want 1", len(g.calls))
	}
	if g.calls[0] != "clone git@example.com:foo/svc-a.git /work/svc-a" {
		t.Fatalf("git call = %q", g.calls[0])
	}
	if len(s.calls) != 1 {
		t.Fatalf("script calls = %d, want 1", len(s.calls))
	}
	if s.calls[0] != "script bootstrap.sh /work/svc-a" {
		t.Fatalf("script call = %q", s.calls[0])
	}
}

func TestInstallAbortsOnCloneFailure(t *testing.T) {
	g := &fakeGit{err: errors.New("network down")}
	s := &fakeScripts{}
	i := New(g, s)

	err := i.Install(context.Background(), PackageSpec{
		Name:        "svc-a",
		RepoURL:     "git@example.com:foo/svc-a.git",
		Dest:        "/work/svc-a",
		PostInstall: []string{"bootstrap.sh"},
	}, io.Discard)
	if err == nil {
		t.Fatalf("expected error from failed clone")
	}
	if len(s.calls) != 0 {
		t.Fatalf("scripts ran after clone failure; calls = %v", s.calls)
	}
}

func TestInstallSkipsCloneWhenRepoURLEmpty(t *testing.T) {
	g := &fakeGit{}
	s := &fakeScripts{}
	i := New(g, s)

	err := i.Install(context.Background(), PackageSpec{
		Name:        "svc-a",
		Dest:        "/work/svc-a",
		PostInstall: []string{"bootstrap.sh"},
	}, io.Discard)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if g.called {
		t.Fatalf("git was invoked despite empty RepoURL")
	}
	if len(s.calls) != 1 {
		t.Fatalf("script calls = %d, want 1", len(s.calls))
	}
}
