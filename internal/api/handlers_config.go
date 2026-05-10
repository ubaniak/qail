package api

import (
	"errors"
	"net/http"

	"github.com/ubaniak/qail/internal/actions"
)

// configResponse mirrors actions.WorkspaceContext but adds the per-repo
// post-install map so a single GET seeds the whole UI shell.
type configResponse struct {
	Root       string                       `json:"root"`
	Editor     string                       `json:"editor"`
	Workspaces map[string]workspaceResponse `json:"workspaces"`
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.store.Read()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	wsResp := make(map[string]workspaceResponse, len(cfg.Workspaces))
	for name, profile := range cfg.Workspaces {
		wsResp[name] = workspaceResponse{
			Repos:       profile.Repos,
			LastUsed:    profile.LastUsed,
			PostInstall: cfg.PostInstallScripts.Workspace[name],
		}
	}
	writeJSON(w, http.StatusOK, configResponse{
		Root:       cfg.Root,
		Editor:     cfg.Editor,
		Workspaces: wsResp,
	})
}

type setStringRequest struct {
	Value string `json:"value"`
}

func (s *Server) handlePutRoot(w http.ResponseWriter, r *http.Request) {
	var req setStringRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Value == "" {
		writeError(w, http.StatusBadRequest, errors.New("value must not be empty"))
		return
	}
	if err := actions.SetRoot(s.store, req.Value); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"root": req.Value})
}

func (s *Server) handlePutEditor(w http.ResponseWriter, r *http.Request) {
	var req setStringRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Value == "" {
		writeError(w, http.StatusBadRequest, errors.New("value must not be empty"))
		return
	}
	if err := actions.SetEditor(s.store, req.Value); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"editor": req.Value})
}
