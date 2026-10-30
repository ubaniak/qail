package actions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ubaniak/qail/internal/config"
)

type staticStore struct{ cfg config.Config }

func (s staticStore) Read() (config.Config, error) { return s.cfg, nil }
func (s staticStore) Write(config.Config) error    { return nil }

func TestWorkspaceBranchesReadsHeadOfClonedRepos(t *testing.T) {
	root := t.TempDir()
	gitDir := filepath.Join(root, "ws", "api", ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := staticStore{cfg: config.Config{
		Root: root,
		Workspaces: config.Workspace{
			"ws":    {Repos: []string{"api", "web"}},
			"empty": {Repos: []string{"api"}},
		},
	}}
	got, err := WorkspaceBranches(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || len(got["ws"]) != 1 || got["ws"]["api"] != "main" {
		t.Fatalf("WorkspaceBranches = %v, want map[ws:map[api:main]]", got)
	}
}
