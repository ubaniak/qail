package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
)

// mkRepo creates a bare-but-initialised git working dir at path with the
// given origin URL. Empty url skips the remote, so callers can build
// dirs with no origin.
func mkRepo(t *testing.T, path, url string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	repo, err := git.PlainInit(path, false)
	if err != nil {
		t.Fatalf("PlainInit %s: %v", path, err)
	}
	if url == "" {
		return
	}
	if _, err := repo.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{url}}); err != nil {
		t.Fatalf("CreateRemote: %v", err)
	}
}

func TestDetectReposMatchesByURL(t *testing.T) {
	root := t.TempDir()
	mkRepo(t, filepath.Join(root, "alpha"), "git@github.com:org/alpha.git")
	mkRepo(t, filepath.Join(root, "renamed-on-disk"), "git@github.com:org/beta.git")

	known := map[string]string{
		"alpha": "git@github.com:org/alpha.git",
		"beta":  "git@github.com:org/beta.git",
	}
	got, err := DetectRepos(root, known)
	if err != nil {
		t.Fatalf("DetectRepos: %v", err)
	}
	byDir := map[string]DetectedRepo{}
	for _, d := range got {
		byDir[d.Dir] = d
	}
	if byDir["alpha"].Match != "alpha" {
		t.Errorf("alpha match = %q, want alpha", byDir["alpha"].Match)
	}
	if byDir["renamed-on-disk"].Match != "beta" {
		t.Errorf("renamed-on-disk match = %q, want beta (URL match)", byDir["renamed-on-disk"].Match)
	}
}

func TestDetectReposFallbackToDirName(t *testing.T) {
	root := t.TempDir()
	mkRepo(t, filepath.Join(root, "alpha"), "") // git repo, no origin

	known := map[string]string{"alpha": "git@x"}
	got, err := DetectRepos(root, known)
	if err != nil {
		t.Fatalf("DetectRepos: %v", err)
	}
	if len(got) != 1 || got[0].Match != "alpha" {
		t.Errorf("got = %+v, want one entry matching by dir name", got)
	}
	if got[0].RemoteURL != "" {
		t.Errorf("RemoteURL = %q, want empty (no origin)", got[0].RemoteURL)
	}
}

func TestDetectReposUnknown(t *testing.T) {
	root := t.TempDir()
	mkRepo(t, filepath.Join(root, "stranger"), "git@github.com:org/stranger.git")

	got, err := DetectRepos(root, map[string]string{"unrelated": "git@x"})
	if err != nil {
		t.Fatalf("DetectRepos: %v", err)
	}
	if len(got) != 1 || got[0].Match != "" || got[0].RemoteURL == "" {
		t.Errorf("got = %+v, want unknown match with remote URL set", got)
	}
}

func TestDetectReposSkipsFilesAndDotfiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README"), []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".hidden"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	mkRepo(t, filepath.Join(root, "real"), "git@x")

	got, err := DetectRepos(root, nil)
	if err != nil {
		t.Fatalf("DetectRepos: %v", err)
	}
	if len(got) != 1 || got[0].Dir != "real" {
		t.Errorf("got = %+v, want [real]", got)
	}
}

func TestDetectReposIncludesPlainDirs(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "not-a-repo"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	got, err := DetectRepos(root, nil)
	if err != nil {
		t.Fatalf("DetectRepos: %v", err)
	}
	if len(got) != 1 || got[0].Dir != "not-a-repo" || got[0].RemoteURL != "" {
		t.Errorf("got = %+v, want one entry with no remote", got)
	}
}
