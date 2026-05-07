package config

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"
)

// storeFactory builds a fresh empty Store. Each adapter passes one in.
type storeFactory func(t *testing.T) Store

func adapters(t *testing.T) map[string]storeFactory {
	t.Helper()
	return map[string]storeFactory{
		"memory": func(_ *testing.T) Store { return NewMemoryStore() },
		"sqlite": func(t *testing.T) Store {
			dir := t.TempDir()
			s, err := NewSQLiteStore(filepath.Join(dir, "qail.db"))
			if err != nil {
				t.Fatalf("NewSQLiteStore: %v", err)
			}
			return s
		},
	}
}

// runParity runs fn against every Store adapter, sub-testing per name.
func runParity(t *testing.T, name string, fn func(t *testing.T, s Store)) {
	t.Helper()
	for adapterName, mk := range adapters(t) {
		t.Run(adapterName+"/"+name, func(t *testing.T) {
			fn(t, mk(t))
		})
	}
}

func TestStoreEmptyRead(t *testing.T) {
	runParity(t, "fresh store reads empty config", func(t *testing.T, s Store) {
		cfg, err := s.Read()
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if cfg.Root != "" || cfg.Editor != "" {
			t.Fatalf("Root/Editor not zero: %+v", cfg)
		}
		if len(cfg.Repos) != 0 || len(cfg.Workspaces) != 0 {
			t.Fatalf("Repos/Workspaces not empty: %+v", cfg)
		}
		// Post-install maps must be non-nil so callers can index into them.
		if cfg.PostInstallScripts.Repo == nil || cfg.PostInstallScripts.Workspace == nil {
			t.Fatalf("PostInstallScripts maps must be non-nil")
		}
	})
}

func TestStoreWriteThenRead(t *testing.T) {
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	want := Config{
		Root:   "/work",
		Editor: "code",
		Repos: map[string]string{
			"svc-a": "git@example.com:foo/svc-a.git",
			"svc-b": "git@example.com:foo/svc-b.git",
		},
		Workspaces: Workspace{
			"team-x": WorkspaceProfile{
				Repos:    []string{"svc-a", "svc-b"},
				LastUsed: now,
			},
		},
		PostInstallScripts: PostInstallScripts{
			Repo: map[string][]string{
				"svc-a": {"a.sh", "b.sh"},
			},
			Workspace: map[string][]string{
				"team-x": {"team.sh"},
			},
		},
	}

	runParity(t, "round-trips a populated config", func(t *testing.T, s Store) {
		if err := s.Write(want); err != nil {
			t.Fatalf("Write: %v", err)
		}
		got, err := s.Read()
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		assertConfigEqual(t, got, want)
	})
}

func TestStoreOverwriteReplacesAll(t *testing.T) {
	first := Config{
		Root:   "/old",
		Editor: "vim",
		Repos:  map[string]string{"a": "url-a", "b": "url-b"},
	}
	second := Config{
		Root:   "/new",
		Editor: "code",
		Repos:  map[string]string{"a": "url-a-updated", "c": "url-c"},
	}

	runParity(t, "second Write replaces stale rows", func(t *testing.T, s Store) {
		if err := s.Write(first); err != nil {
			t.Fatalf("first Write: %v", err)
		}
		if err := s.Write(second); err != nil {
			t.Fatalf("second Write: %v", err)
		}
		got, err := s.Read()
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		// b must be gone, a must be the updated URL, c must be present.
		if _, ok := got.Repos["b"]; ok {
			t.Fatalf("expected stale repo b to be gone, got %v", got.Repos)
		}
		if got.Repos["a"] != "url-a-updated" {
			t.Fatalf("expected a=url-a-updated, got %q", got.Repos["a"])
		}
		if got.Repos["c"] != "url-c" {
			t.Fatalf("expected c=url-c, got %q", got.Repos["c"])
		}
		if got.Root != "/new" || got.Editor != "code" {
			t.Fatalf("settings not overwritten: %+v", got)
		}
	})
}

func TestStorePostInstallScriptOrder(t *testing.T) {
	want := Config{
		PostInstallScripts: PostInstallScripts{
			Repo: map[string][]string{
				"svc-a": {"first.sh", "second.sh", "third.sh"},
			},
			Workspace: map[string][]string{
				"team-x": {"alpha.sh", "beta.sh"},
			},
		},
	}

	runParity(t, "post-install scripts retain insertion order", func(t *testing.T, s Store) {
		if err := s.Write(want); err != nil {
			t.Fatalf("Write: %v", err)
		}
		got, err := s.Read()
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if !reflect.DeepEqual(got.PostInstallScripts.Repo["svc-a"], want.PostInstallScripts.Repo["svc-a"]) {
			t.Fatalf("repo scripts order: got %v want %v", got.PostInstallScripts.Repo["svc-a"], want.PostInstallScripts.Repo["svc-a"])
		}
		if !reflect.DeepEqual(got.PostInstallScripts.Workspace["team-x"], want.PostInstallScripts.Workspace["team-x"]) {
			t.Fatalf("workspace scripts order: got %v want %v", got.PostInstallScripts.Workspace["team-x"], want.PostInstallScripts.Workspace["team-x"])
		}
	})
}

// assertConfigEqual compares two Configs ignoring map iteration order. Repo
// lists and post-install slices preserve order (matching SQLite Idx column);
// other maps are compared key-by-key.
func assertConfigEqual(t *testing.T, got, want Config) {
	t.Helper()
	if got.Root != want.Root {
		t.Fatalf("Root: got %q want %q", got.Root, want.Root)
	}
	if got.Editor != want.Editor {
		t.Fatalf("Editor: got %q want %q", got.Editor, want.Editor)
	}
	if !reflect.DeepEqual(got.Repos, want.Repos) {
		t.Fatalf("Repos:\n got: %v\nwant: %v", got.Repos, want.Repos)
	}
	if len(got.Workspaces) != len(want.Workspaces) {
		t.Fatalf("Workspaces len: got %d want %d", len(got.Workspaces), len(want.Workspaces))
	}
	for name, wantWS := range want.Workspaces {
		gotWS, ok := got.Workspaces[name]
		if !ok {
			t.Fatalf("missing workspace %q", name)
		}
		gr, wr := append([]string(nil), gotWS.Repos...), append([]string(nil), wantWS.Repos...)
		sort.Strings(gr)
		sort.Strings(wr)
		if !reflect.DeepEqual(gr, wr) {
			t.Fatalf("workspace %q repos: got %v want %v", name, gr, wr)
		}
		if !gotWS.LastUsed.Equal(wantWS.LastUsed) {
			t.Fatalf("workspace %q LastUsed: got %v want %v", name, gotWS.LastUsed, wantWS.LastUsed)
		}
	}
	if !reflect.DeepEqual(got.PostInstallScripts, want.PostInstallScripts) {
		t.Fatalf("PostInstallScripts:\n got: %+v\nwant: %+v", got.PostInstallScripts, want.PostInstallScripts)
	}
}
