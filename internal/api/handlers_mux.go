package api

import (
	"net/http"

	"github.com/ubaniak/qail/internal/tmux"
)

func (s *Server) handleListMuxSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := tmux.Default().ListSessions(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
}

func (s *Server) handleRemoveMuxSession(w http.ResponseWriter, r *http.Request) {
	name, err := pathParam(r, "name")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := tmux.Default().RemoveSession(r.Context(), name); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"removed": name})
}
