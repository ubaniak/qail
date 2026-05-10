package api

import (
	"errors"
	"net/http"

	"github.com/ubaniak/qail/internal/actions"
)

type repoResponse struct {
	URL         string   `json:"url"`
	PostInstall []string `json:"postInstall,omitempty"`
}

func (s *Server) handleListRepos(w http.ResponseWriter, _ *http.Request) {
	repos, postInstall, err := actions.ListRepos(s.store)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out := make(map[string]repoResponse, len(repos))
	for name, url := range repos {
		out[name] = repoResponse{URL: url, PostInstall: postInstall[name]}
	}
	writeJSON(w, http.StatusOK, out)
}

type addRepoRequest struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func (s *Server) handleAddRepo(w http.ResponseWriter, r *http.Request) {
	var req addRepoRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Name == "" || req.URL == "" {
		writeError(w, http.StatusBadRequest, errors.New("name and url required"))
		return
	}
	if err := actions.AddRepo(s.store, req.Name, req.URL); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"name": req.Name, "url": req.URL})
}

type removeReposRequest struct {
	Names []string `json:"names"`
}

func (s *Server) handleRemoveRepos(w http.ResponseWriter, r *http.Request) {
	var req removeReposRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(req.Names) == 0 {
		writeError(w, http.StatusBadRequest, errors.New("names must not be empty"))
		return
	}
	if err := actions.RemoveRepos(s.store, req.Names); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"removed": req.Names})
}

type setPostInstallRequest struct {
	Scripts []string `json:"scripts"`
}

func (s *Server) handleSetRepoPostInstall(w http.ResponseWriter, r *http.Request) {
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
	if err := actions.SetRepoPostInstall(s.store, name, req.Scripts); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"repo": name, "scripts": req.Scripts})
}
