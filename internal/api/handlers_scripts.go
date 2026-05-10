package api

import (
	"errors"
	"net/http"

	"github.com/ubaniak/qail/internal/scripts"
)

func (s *Server) handleListScripts(w http.ResponseWriter, _ *http.Request) {
	list, err := scripts.Default().ListScripts()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"scripts": list})
}

type addScriptRequest struct {
	Name string `json:"name"`
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
	if err := scripts.Default().CreateBashScript(req.Name); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"name": req.Name})
}

func (s *Server) handleRemoveScript(w http.ResponseWriter, r *http.Request) {
	name, err := pathParam(r, "name")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	sc := scripts.Default()
	ok, err := sc.Has(name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, errors.New("script not found"))
		return
	}
	if err := sc.RemoveScript(name); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"removed": name})
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
