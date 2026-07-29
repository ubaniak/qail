package api

import (
	"errors"
	"net/http"

	"github.com/ubaniak/qail/internal/actions"
	"github.com/ubaniak/qail/internal/config"
)

// configResponse mirrors actions.WorkspaceContext but adds the per-repo
// post-install map so a single GET seeds the whole UI shell.
type configResponse struct {
	Root          string                       `json:"root"`
	Editors       []editorDTO                  `json:"editors"`
	DefaultEditor string                       `json:"defaultEditor"`
	AIs           []aiDTO                      `json:"ais"`
	DefaultAI     string                       `json:"defaultAI"`
	Workspaces    map[string]workspaceResponse `json:"workspaces"`
}

type editorDTO struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

type aiDTO struct {
	Name    string `json:"name"`
	Command string `json:"command"`
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
			Editor:      profile.Editor,
			AI:          profile.AI,
			PostInstall: cfg.PostInstallScripts.Workspace[name],
		}
	}
	writeJSON(w, http.StatusOK, configResponse{
		Root:          cfg.Root,
		Editors:       toEditorDTOs(cfg.Editors),
		DefaultEditor: cfg.DefaultEditor,
		AIs:           toAIDTOs(cfg.AIs),
		DefaultAI:     cfg.DefaultAI,
		Workspaces:    wsResp,
	})
}

func toEditorDTOs(in []config.Editor) []editorDTO {
	out := make([]editorDTO, len(in))
	for i, e := range in {
		out[i] = editorDTO{Name: e.Name, Command: e.Command}
	}
	return out
}

func toAIDTOs(in []config.AI) []aiDTO {
	out := make([]aiDTO, len(in))
	for i, a := range in {
		out[i] = aiDTO{Name: a.Name, Command: a.Command}
	}
	return out
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

type addEditorRequest struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

func (s *Server) handleAddEditor(w http.ResponseWriter, r *http.Request) {
	var req addEditorRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := actions.AddEditor(s.store, req.Name, req.Command); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, editorDTO{Name: req.Name, Command: req.Command})
}

func (s *Server) handleRemoveEditor(w http.ResponseWriter, r *http.Request) {
	name, err := pathParam(r, "name")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := actions.RemoveEditor(s.store, name); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"removed": name})
}

func (s *Server) handleSetDefaultEditor(w http.ResponseWriter, r *http.Request) {
	var req setStringRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := actions.SetDefaultEditor(s.store, req.Value); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"defaultEditor": req.Value})
}

func (s *Server) handleSetWorkspaceEditor(w http.ResponseWriter, r *http.Request) {
	name, err := pathParam(r, "name")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var req setStringRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := actions.SetWorkspaceEditor(s.store, name, req.Value); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"workspace": name, "editor": req.Value})
}

type addAIRequest struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

func (s *Server) handleAddAI(w http.ResponseWriter, r *http.Request) {
	var req addAIRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := actions.AddAI(s.store, req.Name, req.Command); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, aiDTO{Name: req.Name, Command: req.Command})
}

func (s *Server) handleRemoveAI(w http.ResponseWriter, r *http.Request) {
	name, err := pathParam(r, "name")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := actions.RemoveAI(s.store, name); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"removed": name})
}

func (s *Server) handleSetDefaultAI(w http.ResponseWriter, r *http.Request) {
	var req setStringRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := actions.SetDefaultAI(s.store, req.Value); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"defaultAI": req.Value})
}

func (s *Server) handleSetWorkspaceAI(w http.ResponseWriter, r *http.Request) {
	name, err := pathParam(r, "name")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var req setStringRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := actions.SetWorkspaceAI(s.store, name, req.Value); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"workspace": name, "ai": req.Value})
}
