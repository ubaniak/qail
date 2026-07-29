// Package api exposes qail's actions over HTTP so a web UI can drive the
// same operations the Cobra CLI does. Every endpoint funnels into the
// internal/actions package — no business logic lives in this layer; it
// only translates HTTP requests into action calls and serialises results.
//
// Long-running mutations (workspace create, edit, clone, on-disk rebuild)
// stream progress as Server-Sent Events so a browser can show clone /
// post-install output as it happens. All other endpoints are plain JSON.
//
// Concurrency: every request goroutine ultimately calls actions.* which
// serialises through the package mutex (see actions/actions.go). SQLite
// is shared with the CLI via the same DB path, so a `qail repo add` from
// a terminal while the server is up is observed by the next /api/repos
// GET — but the lost-update race that would corrupt cross-process writes
// is not solved here. Single-user assumption stands; deploy with care.
package api

import (
	"net/http"

	"github.com/ubaniak/qail/internal/config"
)

// Server holds the dependencies shared by every handler. The Store is the
// only mutable seam; everything else (runners, sub-clients) is constructed
// per-request because they're cheap and stateless.
type Server struct {
	store config.Store
}

// New returns a Server wired to s. Callers construct s once at startup
// (e.g. config.NewSQLiteStore from the cmd/serve.go handler) and hand it
// in here.
func New(s config.Store) *Server {
	return &Server{store: s}
}

// Handler returns the HTTP mux with every route registered. Composed in a
// constructor (rather than method on Server) so tests can swap mux flavours
// (e.g. wrap with a logger middleware) without touching Server itself.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// config / settings
	mux.HandleFunc("GET /api/config", s.handleGetConfig)
	mux.HandleFunc("PUT /api/config/root", s.handlePutRoot)
	mux.HandleFunc("POST /api/config/editors", s.handleAddEditor)
	mux.HandleFunc("DELETE /api/config/editors/{name}", s.handleRemoveEditor)
	mux.HandleFunc("PUT /api/config/editors/default", s.handleSetDefaultEditor)
	mux.HandleFunc("POST /api/config/ais", s.handleAddAI)
	mux.HandleFunc("DELETE /api/config/ais/{name}", s.handleRemoveAI)
	mux.HandleFunc("PUT /api/config/ais/default", s.handleSetDefaultAI)

	// repos
	mux.HandleFunc("GET /api/repos", s.handleListRepos)
	mux.HandleFunc("POST /api/repos", s.handleAddRepo)
	mux.HandleFunc("DELETE /api/repos", s.handleRemoveRepos)
	mux.HandleFunc("PUT /api/repos/{name}/post-install", s.handleSetRepoPostInstall)

	// workspaces
	mux.HandleFunc("GET /api/workspaces", s.handleListWorkspaces)
	mux.HandleFunc("POST /api/workspaces", s.handleAddWorkspace)
	mux.HandleFunc("PUT /api/workspaces/{name}", s.handleEditWorkspace)
	mux.HandleFunc("DELETE /api/workspaces/{name}", s.handleRemoveWorkspace)
	mux.HandleFunc("POST /api/workspaces/{name}/clone", s.handleCloneWorkspace)
	mux.HandleFunc("POST /api/workspaces/{name}/create", s.handleCreateWorkspaceOnDisk)
	mux.HandleFunc("PUT /api/workspaces/{name}/post-install", s.handleSetWorkspacePostInstall)
	mux.HandleFunc("PUT /api/workspaces/{name}/editor", s.handleSetWorkspaceEditor)
	mux.HandleFunc("PUT /api/workspaces/{name}/ai", s.handleSetWorkspaceAI)
	mux.HandleFunc("GET /api/workspaces/{name}/path", s.handleCdWorkspace)
	mux.HandleFunc("GET /api/workspaces/{name}/mux", s.handleMuxWorkspace)
	mux.HandleFunc("GET /api/workspaces/{name}/open-cmd", s.handleOpenWorkspaceCmd)
	mux.HandleFunc("GET /api/workspaces/{name}/open-ai-cmd", s.handleOpenWorkspaceAICmd)

	// scripts
	mux.HandleFunc("GET /api/scripts", s.handleListScripts)
	mux.HandleFunc("POST /api/scripts", s.handleAddScript)
	mux.HandleFunc("DELETE /api/scripts/{name}", s.handleRemoveScript)
	mux.HandleFunc("GET /api/scripts/path", s.handleScriptsPath)

	// tmux sessions
	mux.HandleFunc("GET /api/mux/sessions", s.handleListMuxSessions)
	mux.HandleFunc("DELETE /api/mux/sessions/{name}", s.handleRemoveMuxSession)

	// healthcheck — useful for liveness probes / smoke tests.
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	return mux
}
