package api

import (
	"errors"
	"net/http"

	"github.com/ubaniak/qail/internal/scripts"
)

// scopeFromQuery reads `?scope=workspace|repo` and returns the typed
// scope. Empty defaults to ScopeWorkspace so historical clients that
// predate the split continue to act on workspace scripts. Returns an
// error for any other value so the API never silently writes the wrong
// bucket.
func scopeFromQuery(r *http.Request) (scripts.Scope, error) {
	raw := r.URL.Query().Get("scope")
	if raw == "" {
		return scripts.ScopeWorkspace, nil
	}
	s := scripts.Scope(raw)
	if !s.Valid() {
		return "", errors.New("invalid scope: must be workspace or repo")
	}
	return s, nil
}

func (s *Server) handleListScripts(w http.ResponseWriter, r *http.Request) {
	scope, err := scopeFromQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	list, err := scripts.Default().ListScripts(scope)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"scripts": list, "scope": string(scope)})
}

type addScriptRequest struct {
	Name  string `json:"name"`
	Scope string `json:"scope"`
}

func (s *Server) handleAddScript(w http.ResponseWriter, r *http.Request) {
	var req addScriptRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, errors.New("name required"))
		return
	}
	scope := scripts.Scope(req.Scope)
	if req.Scope == "" {
		scope = scripts.ScopeWorkspace
	}
	if !scope.Valid() {
		writeError(w, http.StatusBadRequest, errors.New("invalid scope"))
		return
	}
	if err := scripts.Default().CreateBashScript(req.Name, scope); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"name": req.Name, "scope": string(scope)})
}

func (s *Server) handleRemoveScript(w http.ResponseWriter, r *http.Request) {
	name, err := pathParam(r, "name")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	scope, err := scopeFromQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	sc := scripts.Default()
	ok, err := sc.Has(scope, name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, errors.New("script not found"))
		return
	}
	if err := sc.RemoveScript(name, scope); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"removed": name, "scope": string(scope)})
}

// handleScriptsPath returns the on-disk scripts directory and the `cd`
// command the user can paste into a shell. Mirrors `qail scripts cd`
// without touching the server's clipboard (which it has no access to).
func (s *Server) handleScriptsPath(w http.ResponseWriter, _ *http.Request) {
	sc := scripts.Default()
	dir, err := sc.GetScriptDir()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	cmd, err := sc.CdCommand()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"path": dir, "command": cmd})
}
