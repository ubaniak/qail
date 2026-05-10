package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/ubaniak/qail/internal/actions"
)

// workspaceResponse is the JSON shape for a single workspace. LastUsed is
// time.Time so JSON marshalling produces an ISO-8601 string the browser
// can pass directly to `new Date()`.
type workspaceResponse struct {
	Repos       []string  `json:"repos"`
	LastUsed    time.Time `json:"lastUsed"`
	PostInstall []string  `json:"postInstall,omitempty"`
}

func (s *Server) handleListWorkspaces(w http.ResponseWriter, _ *http.Request) {
	workspaces, postInstall, err := actions.ListWorkspaces(s.store)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out := make(map[string]workspaceResponse, len(workspaces))
	for name, profile := range workspaces {
		out[name] = workspaceResponse{
			Repos:       profile.Repos,
			LastUsed:    profile.LastUsed,
			PostInstall: postInstall[name],
		}
	}
	writeJSON(w, http.StatusOK, out)
}

type addWorkspaceRequest struct {
	Name     string   `json:"name"`
	Packages []string `json:"packages"`
}

// handleAddWorkspace creates a new workspace and streams progress as SSE.
// Body is consumed before streaming starts so a malformed JSON request
// returns a normal 400, not a half-open SSE stream.
func (s *Server) handleAddWorkspace(w http.ResponseWriter, r *http.Request) {
	var req addWorkspaceRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, errors.New("name required"))
		return
	}
	sw, err := newSSE(w, r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	streamAction(r.Context(), sw, func(ctx context.Context, out io.Writer) error {
		return actions.AddWorkspace(ctx, s.store, req.Name, req.Packages, out)
	})
}

type editWorkspaceRequest struct {
	Packages []string `json:"packages"`
}

func (s *Server) handleEditWorkspace(w http.ResponseWriter, r *http.Request) {
	name, err := pathParam(r, "name")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var req editWorkspaceRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	sw, err := newSSE(w, r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	streamAction(r.Context(), sw, func(ctx context.Context, out io.Writer) error {
		return actions.EditWorkspace(ctx, s.store, name, req.Packages, out)
	})
}

func (s *Server) handleRemoveWorkspace(w http.ResponseWriter, r *http.Request) {
	name, err := pathParam(r, "name")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := actions.RemoveWorkspace(s.store, name); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"removed": name})
}

type cloneWorkspaceRequest struct {
	Dst      string   `json:"dst"`
	Packages []string `json:"packages"`
}

func (s *Server) handleCloneWorkspace(w http.ResponseWriter, r *http.Request) {
	src, err := pathParam(r, "name")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var req cloneWorkspaceRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Dst == "" {
		writeError(w, http.StatusBadRequest, errors.New("dst required"))
		return
	}
	// If packages omitted, inherit from source. Done before SSE so the
	// validation error is a normal 400.
	pkgs := req.Packages
	if len(pkgs) == 0 {
		ctx, err := actions.ReadWorkspaceContext(s.store)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		profile, ok := ctx.Workspaces[src]
		if !ok {
			writeError(w, http.StatusNotFound, errors.New("source workspace not found"))
			return
		}
		pkgs = profile.Repos
	}
	sw, err := newSSE(w, r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	streamAction(r.Context(), sw, func(ctx context.Context, out io.Writer) error {
		return actions.CloneWorkspace(ctx, s.store, req.Dst, pkgs, out)
	})
}

func (s *Server) handleCreateWorkspaceOnDisk(w http.ResponseWriter, r *http.Request) {
	name, err := pathParam(r, "name")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	sw, err := newSSE(w, r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	streamAction(r.Context(), sw, func(ctx context.Context, out io.Writer) error {
		return actions.CreateWorkspaceOnDisk(ctx, s.store, name, out)
	})
}

func (s *Server) handleSetWorkspacePostInstall(w http.ResponseWriter, r *http.Request) {
	name, err := pathParam(r, "name")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var req setPostInstallRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := actions.SetWorkspacePostInstall(s.store, name, req.Scripts); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"workspace": name, "scripts": req.Scripts})
}

func (s *Server) handleCdWorkspace(w http.ResponseWriter, r *http.Request) {
	name, err := pathParam(r, "name")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	wsPath, err := actions.CdWorkspace(s.store, name)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"path": wsPath})
}

func (s *Server) handleMuxWorkspace(w http.ResponseWriter, r *http.Request) {
	name, err := pathParam(r, "name")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	attachCmd, err := actions.MuxWorkspace(r.Context(), s.store, name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"attachCommand": attachCmd})
}

func (s *Server) handleOpenWorkspaceCmd(w http.ResponseWriter, r *http.Request) {
	name, err := pathParam(r, "name")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	cmd, err := actions.OpenWorkspaceCommand(s.store, name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"editor":  cmd.Editor,
		"path":    cmd.Path,
		"command": cmd.String(),
	})
}
