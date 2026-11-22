package app

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ubaniak/qail/internal/config"
)

// The tray builds its Agent Console submenu from ConsoleStatus alone, so
// it must list workspaces most recently used first, and still answer when
// no console is wired up.
func TestConsoleStatusListsWorkspacesByLastUsed(t *testing.T) {
	now := time.Now()
	s := config.NewMemoryStoreFrom(config.Config{
		Repos: map[string]string{"api": "u1", "web": "u2"},
		Workspaces: config.Workspace{
			"old": config.WorkspaceProfile{Repos: []string{"api"}, LastUsed: now.Add(-time.Hour)},
			"new": config.WorkspaceProfile{Repos: []string{"api", "web"}, LastUsed: now},
		},
	})
	got := roundTrip(t, s, `{"id":1,"method":"ConsoleStatus","args":[]}`)
	if e := got[1]["error"]; e != nil {
		t.Fatalf("ConsoleStatus error: %s", e)
	}
	var st ConsoleStatusDTO
	if err := json.Unmarshal(got[1]["result"], &st); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if st.ClaudeFound || st.Running {
		t.Fatalf("no console wired, got %+v", st)
	}
	if len(st.Workspaces) != 2 || st.Workspaces[0].Name != "new" || len(st.Workspaces[0].Repos) != 2 {
		t.Fatalf("Workspaces = %+v", st.Workspaces)
	}
}

func TestOpenConsoleWithoutConsoleErrors(t *testing.T) {
	got := roundTrip(t, config.NewMemoryStore(), `{"id":1,"method":"OpenConsole","args":["",""]}`)
	if got[1]["error"] == nil {
		t.Fatal("OpenConsole succeeded with no console wired up")
	}
}
