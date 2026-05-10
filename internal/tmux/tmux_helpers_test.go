package tmux

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ubaniak/qail/internal/runner"
)

func TestAttachCommandFormatsCorrectly(t *testing.T) {
	tx := New(nil)
	got := tx.AttachCommand("my-session")
	want := "tmux a -t 'my-session'"
	if got != want {
		t.Fatalf("AttachCommand = %q, want %q", got, want)
	}
}

func TestAttachCommandQuotesSingleQuotes(t *testing.T) {
	tx := New(nil)
	got := tx.AttachCommand("it's")
	want := "tmux a -t 'it'\\''s'"
	if got != want {
		t.Fatalf("AttachCommand = %q, want %q", got, want)
	}
}

func TestSessionNameReturnsBasename(t *testing.T) {
	tx := New(nil)
	tests := []struct {
		path string
		want string
	}{
		{"/work/projects/alpha", "alpha"},
		{"/single", "single"},
		{"relative/path", "path"},
	}
	for _, tt := range tests {
		got := tx.SessionName(tt.path)
		if got != tt.want {
			t.Errorf("SessionName(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestLaunchCreatesSessionWithSubfolders(t *testing.T) {
	dir := t.TempDir()
	// Create two visible subdirectories and one hidden
	for _, name := range []string{"svc-a", "svc-b", ".hidden"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
	}
	// Create a regular file (should be ignored)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi"), 0644)

	rec := runner.NewRecorder()
	// Seed enough responses for all tmux commands Plan generates
	for i := 0; i < 20; i++ {
		rec.RespondOK(nil)
	}
	tx := New(rec)

	if err := tx.Launch(context.Background(), dir); err != nil {
		t.Fatalf("Launch: %v", err)
	}
	if len(rec.Calls) == 0 {
		t.Fatal("expected tmux commands, got none")
	}
	// First call should be "tmux new-session"
	first := rec.Calls[0]
	if first.Name != "tmux" {
		t.Fatalf("first call name = %q, want tmux", first.Name)
	}
	if first.Args[0] != "new-session" {
		t.Fatalf("first call args[0] = %q, want new-session", first.Args[0])
	}
}
