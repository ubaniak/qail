package actions

import (
	"reflect"
	"sort"
	"testing"

	"github.com/ubaniak/qail/internal/config"
)

func TestAddRepoNewEntry(t *testing.T) {
	s := config.NewMemoryStore()
	if err := AddRepo(s, "svc-a", "git@example.com:foo/svc-a.git"); err != nil {
		t.Fatalf("AddRepo: %v", err)
	}

	cfg, err := s.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if cfg.Repos["svc-a"] != "git@example.com:foo/svc-a.git" {
		t.Fatalf("Repos[svc-a] = %q", cfg.Repos["svc-a"])
	}
}

func TestAddRepoOverwritesExisting(t *testing.T) {
	s := config.NewMemoryStoreFrom(config.Config{
		Repos: map[string]string{"svc-a": "old"},
	})
	if err := AddRepo(s, "svc-a", "new"); err != nil {
		t.Fatalf("AddRepo: %v", err)
	}
	cfg, _ := s.Read()
	if cfg.Repos["svc-a"] != "new" {
		t.Fatalf("expected overwrite, got %q", cfg.Repos["svc-a"])
	}
}

func TestAddRepoEmptyName(t *testing.T) {
	s := config.NewMemoryStore()
	if err := AddRepo(s, "", "url"); err == nil {
		t.Fatalf("expected error for empty name")
	}
}

func TestRemoveReposCascadesIntoWorkspaces(t *testing.T) {
	s := config.NewMemoryStoreFrom(config.Config{
		Repos: map[string]string{
			"svc-a": "url-a",
			"svc-b": "url-b",
			"svc-c": "url-c",
		},
		Workspaces: config.Workspace{
			"team-x": config.WorkspaceProfile{Repos: []string{"svc-a", "svc-b"}},
			"team-y": config.WorkspaceProfile{Repos: []string{"svc-b", "svc-c"}},
		},
		PostInstallScripts: config.PostInstallScripts{
			Repo: map[string][]string{
				"svc-a": {"a.sh"},
				"svc-b": {"b.sh"},
			},
			Workspace: map[string][]string{},
		},
	})

	if err := RemoveRepos(s, []string{"svc-a", "svc-b"}); err != nil {
		t.Fatalf("RemoveRepos: %v", err)
	}

	cfg, _ := s.Read()
	if _, ok := cfg.Repos["svc-a"]; ok {
		t.Fatalf("svc-a still in Repos")
	}
	if _, ok := cfg.Repos["svc-b"]; ok {
		t.Fatalf("svc-b still in Repos")
	}
	if cfg.Repos["svc-c"] != "url-c" {
		t.Fatalf("svc-c lost, got %v", cfg.Repos)
	}

	// Workspaces stripped.
	if got := cfg.Workspaces["team-x"].Repos; len(got) != 0 {
		t.Fatalf("team-x repos = %v, want empty", got)
	}
	if got := cfg.Workspaces["team-y"].Repos; !reflect.DeepEqual(got, []string{"svc-c"}) {
		t.Fatalf("team-y repos = %v, want [svc-c]", got)
	}

	// Per-repo post-install scripts stripped for removed repos.
	if _, ok := cfg.PostInstallScripts.Repo["svc-a"]; ok {
		t.Fatalf("svc-a post-install scripts not stripped")
	}
	if _, ok := cfg.PostInstallScripts.Repo["svc-b"]; ok {
		t.Fatalf("svc-b post-install scripts not stripped")
	}
}

func TestRemoveReposEmptyIsNoop(t *testing.T) {
	s := config.NewMemoryStoreFrom(config.Config{
		Repos: map[string]string{"svc-a": "url"},
	})
	if err := RemoveRepos(s, nil); err != nil {
		t.Fatalf("RemoveRepos(nil): %v", err)
	}
	cfg, _ := s.Read()
	if cfg.Repos["svc-a"] != "url" {
		t.Fatalf("noop expected, got %v", cfg.Repos)
	}
}

func TestSetRepoPostInstallReplacesList(t *testing.T) {
	s := config.NewMemoryStoreFrom(config.Config{
		PostInstallScripts: config.PostInstallScripts{
			Repo: map[string][]string{
				"svc-a": {"old.sh"},
			},
		},
	})
	if err := SetRepoPostInstall(s, "svc-a", []string{"new1.sh", "new2.sh"}); err != nil {
		t.Fatalf("SetRepoPostInstall: %v", err)
	}
	cfg, _ := s.Read()
	got := cfg.PostInstallScripts.Repo["svc-a"]
	want := []string{"new1.sh", "new2.sh"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("scripts = %v, want %v", got, want)
	}
}

func TestSetRepoPostInstallEmptyClears(t *testing.T) {
	s := config.NewMemoryStoreFrom(config.Config{
		PostInstallScripts: config.PostInstallScripts{
			Repo: map[string][]string{
				"svc-a": {"old.sh"},
			},
		},
	})
	if err := SetRepoPostInstall(s, "svc-a", nil); err != nil {
		t.Fatalf("SetRepoPostInstall(nil): %v", err)
	}
	cfg, _ := s.Read()
	if _, ok := cfg.PostInstallScripts.Repo["svc-a"]; ok {
		t.Fatalf("expected entry cleared, got %v", cfg.PostInstallScripts.Repo)
	}
}

func TestListReposReturnsBoth(t *testing.T) {
	s := config.NewMemoryStoreFrom(config.Config{
		Repos: map[string]string{"svc-a": "url-a", "svc-b": "url-b"},
		PostInstallScripts: config.PostInstallScripts{
			Repo: map[string][]string{"svc-a": {"hook.sh"}},
		},
	})
	repos, postInstall, err := ListRepos(s)
	if err != nil {
		t.Fatalf("ListRepos: %v", err)
	}
	keys := make([]string, 0, len(repos))
	for k := range repos {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if !reflect.DeepEqual(keys, []string{"svc-a", "svc-b"}) {
		t.Fatalf("repo keys = %v", keys)
	}
	if postInstall["svc-a"][0] != "hook.sh" {
		t.Fatalf("post-install lost: %v", postInstall)
	}
}
