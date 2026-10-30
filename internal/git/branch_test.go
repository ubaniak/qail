package git

import (
	"os"
	"path/filepath"
	"testing"
)

func writeHead(t *testing.T, gitDir, contents string) {
	t.Helper()
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCurrentBranch(t *testing.T) {
	root := t.TempDir()

	onBranch := filepath.Join(root, "on-branch")
	writeHead(t, filepath.Join(onBranch, ".git"), "ref: refs/heads/feature/x\n")

	detached := filepath.Join(root, "detached")
	writeHead(t, filepath.Join(detached, ".git"), "0123456789abcdef0123456789abcdef01234567\n")

	worktree := filepath.Join(root, "worktree")
	writeHead(t, filepath.Join(root, "main.git", "worktrees", "wt"), "ref: refs/heads/wt-branch\n")
	if err := os.MkdirAll(worktree, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worktree, ".git"), []byte("gitdir: ../main.git/worktrees/wt\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := map[string]string{
		onBranch:                       "feature/x",
		detached:                       "0123456",
		worktree:                       "wt-branch",
		filepath.Join(root, "missing"): "",
	}
	for dir, want := range cases {
		if got := CurrentBranch(dir); got != want {
			t.Errorf("CurrentBranch(%s) = %q, want %q", filepath.Base(dir), got, want)
		}
	}
}
