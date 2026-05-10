package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ubaniak/qail/internal/config"
)

// newTestServer wires a Server against an in-memory store seeded with cfg.
// Returns the httptest.Server so tests use a real transport (covers the
// route mux + middleware) rather than calling handlers directly.
func newTestServer(t *testing.T, cfg config.Config) (*httptest.Server, *config.MemoryStore) {
	t.Helper()
	store := config.NewMemoryStoreFrom(cfg)
	s := New(store)
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return ts, store
}

func TestGetConfigReturnsRootEditorWorkspaces(t *testing.T) {
	ts, _ := newTestServer(t, config.Config{
		Root:   "/q",
		Editor: "code",
		Workspaces: config.Workspace{
			"alpha": config.WorkspaceProfile{Repos: []string{"a"}, LastUsed: time.Now()},
		},
	})

	resp, err := http.Get(ts.URL + "/api/config")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}

	var got configResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Root != "/q" || got.Editor != "code" {
		t.Errorf("got = %+v", got)
	}
	if _, ok := got.Workspaces["alpha"]; !ok {
		t.Errorf("alpha missing: %+v", got.Workspaces)
	}
}

func TestAddRepoPersistsToStore(t *testing.T) {
	ts, store := newTestServer(t, config.Config{})
	body := bytes.NewBufferString(`{"name":"svc-a","url":"git@host:foo/svc-a.git"}`)

	resp, err := http.Post(ts.URL+"/api/repos", "application/json", body)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, body = %s", resp.StatusCode, b)
	}

	cfg, _ := store.Read()
	if cfg.Repos["svc-a"] != "git@host:foo/svc-a.git" {
		t.Errorf("Repos = %+v", cfg.Repos)
	}
}

func TestAddRepoRejectsMissingFields(t *testing.T) {
	ts, _ := newTestServer(t, config.Config{})
	body := bytes.NewBufferString(`{"name":"svc-a"}`)

	resp, err := http.Post(ts.URL+"/api/repos", "application/json", body)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestRemoveWorkspaceRespondsAndPersists(t *testing.T) {
	ts, store := newTestServer(t, config.Config{
		Workspaces: config.Workspace{
			"alpha": config.WorkspaceProfile{Repos: []string{"a"}, LastUsed: time.Now()},
		},
	})

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/workspaces/alpha", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	cfg, _ := store.Read()
	if _, ok := cfg.Workspaces["alpha"]; ok {
		t.Errorf("alpha still present after DELETE: %+v", cfg.Workspaces)
	}
}

func TestListReposShapesPostInstall(t *testing.T) {
	ts, _ := newTestServer(t, config.Config{
		Repos: map[string]string{"svc-a": "git@a"},
		PostInstallScripts: config.PostInstallScripts{
			Repo: map[string][]string{"svc-a": {"setup.sh"}},
		},
	})

	resp, err := http.Get(ts.URL + "/api/repos")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var got map[string]repoResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	r, ok := got["svc-a"]
	if !ok {
		t.Fatalf("svc-a missing: %+v", got)
	}
	if r.URL != "git@a" || len(r.PostInstall) != 1 || r.PostInstall[0] != "setup.sh" {
		t.Errorf("repoResponse = %+v", r)
	}
}

func TestHealthEndpoint(t *testing.T) {
	ts, _ := newTestServer(t, config.Config{})
	resp, err := http.Get(ts.URL + "/api/health")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"status":"ok"`) {
		t.Errorf("body = %s", body)
	}
}
