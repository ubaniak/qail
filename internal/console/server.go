package console

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ubaniak/qail/internal/actions"
)

//go:embed web
var webFS embed.FS

// wsScope is the repo slot meaning "the whole workspace".
const wsScope = "_"

// readOnlyTools never prompt: they can't change anything.
var readOnlyTools = map[string]bool{
	"Read": true, "Grep": true, "Glob": true, "LS": true, "TodoWrite": true,
	"Skill": true, "WebSearch": true, "WebFetch": true, "NotebookRead": true,
}

// permissionTimeout denies a prompt nobody answers.
const permissionTimeout = 10 * time.Minute

type server struct {
	opts       Options
	threads    *threadStore
	homeClaude string

	mu      sync.Mutex
	live    map[string]context.CancelFunc // threadKey -> stop
	waiters map[string]*permWaiter        // permId -> waiter
	running sync.WaitGroup
}

type permWaiter struct {
	key string
	ch  chan Decision
}

func newServer(o Options) (*server, error) {
	ts, err := openThreadStore(o.DataDir)
	if err != nil {
		return nil, fmt.Errorf("agent console threads: %w", err)
	}
	home, _ := os.UserHomeDir()
	return &server{
		opts:       o,
		threads:    ts,
		homeClaude: filepath.Join(home, ".claude"),
		live:       map[string]context.CancelFunc{},
		waiters:    map[string]*permWaiter{},
	}, nil
}

func (s *server) handler() http.Handler {
	mux := http.NewServeMux()
	static, _ := fs.Sub(webFS, "web")
	mux.Handle("GET /", http.FileServerFS(static))

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "app": "qail-console"})
	})
	mux.HandleFunc("GET /api/workspaces", s.handleWorkspaces)
	mux.HandleFunc("POST /api/workspaces/{ws}/create", s.handleCreate)
	mux.HandleFunc("GET /api/scope/{ws}/{repo}/overview", s.handleOverview)
	mux.HandleFunc("GET /api/scope/{ws}/{repo}/runs/{run}/{file}", s.handleRunFile)
	mux.HandleFunc("GET /api/scope/{ws}/{repo}/threads/{agent}", s.handleGetThread)
	mux.HandleFunc("DELETE /api/scope/{ws}/{repo}/threads/{agent}", s.handleDeleteThread)
	mux.HandleFunc("POST /api/scope/{ws}/{repo}/threads/{agent}/stop", s.handleStop)
	mux.HandleFunc("POST /api/scope/{ws}/{repo}/threads/{agent}/chat", s.handleChat)
	mux.HandleFunc("POST /api/permissions/{permId}", s.handlePermission)
	return guard(mux)
}

var localOrigin = regexp.MustCompile(`^http://(127\.0\.0\.1|localhost):\d+$`)

// guard keeps other sites in the user's browser away from local agents:
// cross-origin API calls are refused, and so is any Host other than
// loopback (DNS rebinding).
func guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}
		if host != "127.0.0.1" && host != "localhost" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "host not allowed"})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			if o := r.Header.Get("Origin"); o != "" && !localOrigin.MatchString(o) {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "cross-origin blocked"})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// ---------- qail model ----------

type repoView struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Path        string `json:"path"`
	Cloned      bool   `json:"cloned"`
	HasClaude   bool   `json:"hasClaude"`
	HasClaudeMd bool   `json:"hasClaudeMd"`
	Busy        bool   `json:"busy"`
}

type workspaceView struct {
	Name        string     `json:"name"`
	Path        string     `json:"path"`
	Exists      bool       `json:"exists"`
	LastUsed    time.Time  `json:"lastUsed"`
	Editor      string     `json:"editor"`
	PostInstall []string   `json:"postInstall"`
	Repos       []repoView `json:"repos"`
	Busy        bool       `json:"busy"`
}

type model struct {
	Root       string          `json:"root"`
	Workspaces []workspaceView `json:"workspaces"`
}

// loadModel builds the workspace tree from qail's config. Layout per
// CONTEXT.md: <Root>/<workspace>/<repo>.
func (s *server) loadModel() (*model, error) {
	cfg, err := s.opts.Store.Read()
	if err != nil {
		return nil, err
	}
	m := &model{Root: cfg.Root, Workspaces: []workspaceView{}}
	for name, p := range cfg.Workspaces {
		wsPath := filepath.Join(cfg.Root, name)
		w := workspaceView{
			Name: name, Path: wsPath, Exists: isDir(wsPath), LastUsed: p.LastUsed,
			Editor: p.Editor, PostInstall: nonNil(cfg.PostInstallScripts.Workspace[name]), Repos: []repoView{},
		}
		if w.Editor == "" {
			w.Editor = cfg.DefaultEditor
		}
		for _, r := range p.Repos {
			rp := filepath.Join(wsPath, r)
			w.Repos = append(w.Repos, repoView{
				Name: r, URL: cfg.Repos[r], Path: rp, Cloned: isDir(rp),
				HasClaude:   fileExists(filepath.Join(rp, ".claude")),
				HasClaudeMd: fileExists(filepath.Join(rp, "CLAUDE.md")) || fileExists(filepath.Join(rp, "AGENTS.md")),
			})
		}
		m.Workspaces = append(m.Workspaces, w)
	}
	sort.Slice(m.Workspaces, func(i, j int) bool { return m.Workspaces[i].LastUsed.After(m.Workspaces[j].LastUsed) })
	return m, nil
}

type scope struct {
	id    string
	ws    workspaceView
	repo  *repoView
	path  string
	label string
	ok    bool // on disk
}

type httpError struct {
	status int
	msg    string
}

func (e httpError) Error() string { return e.msg }

func (s *server) resolveScope(wsName, repoName string) (*scope, error) {
	m, err := s.loadModel()
	if err != nil {
		return nil, err
	}
	for _, w := range m.Workspaces {
		if w.Name != wsName {
			continue
		}
		id := wsName + "/" + repoName
		if repoName == wsScope {
			return &scope{id: id, ws: w, path: w.Path, label: wsName, ok: w.Exists}, nil
		}
		for i := range w.Repos {
			if w.Repos[i].Name == repoName {
				r := w.Repos[i]
				return &scope{id: id, ws: w, repo: &r, path: r.Path, label: wsName + " / " + repoName, ok: r.Cloned}, nil
			}
		}
		return nil, httpError{http.StatusNotFound, fmt.Sprintf("repo %q is not in workspace %q", repoName, wsName)}
	}
	return nil, httpError{http.StatusNotFound, fmt.Sprintf("no qail workspace %q", wsName)}
}

// ---------- handlers ----------

type claudeStatus struct {
	OK    bool   `json:"ok"`
	Path  string `json:"path,omitempty"`
	Error string `json:"error,omitempty"`
}

func (s *server) claude() claudeStatus {
	p, err := FindClaude(s.opts.Claude)
	if err != nil {
		return claudeStatus{Error: err.Error()}
	}
	return claudeStatus{OK: true, Path: p}
}

func (s *server) busyScopes() map[string]bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]bool{}
	for k := range s.live {
		out[strings.SplitN(k, "::", 2)[0]] = true
	}
	return out
}

func (s *server) handleWorkspaces(w http.ResponseWriter, _ *http.Request) {
	m, err := s.loadModel()
	if err != nil {
		writeErr(w, err)
		return
	}
	busy := s.busyScopes()
	for i := range m.Workspaces {
		ws := &m.Workspaces[i]
		ws.Busy = busy[ws.Name+"/"+wsScope]
		for j := range ws.Repos {
			ws.Repos[j].Busy = busy[ws.Name+"/"+ws.Repos[j].Name]
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"root": m.Root, "workspaces": m.Workspaces, "claude": s.claude()})
}

// handleCreate runs qail's "create on disk" (clone missing repos + run
// post-install scripts), streaming its output as SSE.
func (s *server) handleCreate(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("ws")
	if _, err := s.resolveScope(name, wsScope); err != nil {
		writeErr(w, err)
		return
	}
	sse, ok := newSSE(w)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming unsupported"})
		return
	}
	err := actions.CreateWorkspaceOnDisk(r.Context(), s.opts.Store, name, sseLines{sse})
	if err != nil {
		sse.raw("event: error\ndata: " + oneLine(err.Error()) + "\n\n")
		return
	}
	sse.raw("event: done\ndata: ok\n\n")
}

func (s *server) handleOverview(w http.ResponseWriter, r *http.Request) {
	sc, err := s.resolveScope(r.PathValue("ws"), r.PathValue("repo"))
	if err != nil {
		writeErr(w, err)
		return
	}
	scopeOut := map[string]any{"id": sc.id, "label": sc.label, "path": sc.path, "repo": sc.repo}
	if !sc.ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"scope": scopeOut, "workspace": sc.ws, "agents": []Agent{}, "skills": []Skill{},
			"runs": []ShipRun{}, "hasClaudeMd": []string{}, "missing": true,
		})
		return
	}
	skills := discoverSkills(s.homeClaude, sc.path)
	agents := discoverAgents(s.homeClaude, sc.path, skills)
	s.mu.Lock()
	live := map[string]bool{}
	for k := range s.live {
		live[k] = true
	}
	s.mu.Unlock()
	for i := range agents {
		a := &agents[i]
		key := threadKey(sc.id, a.Name)
		if t := s.threads.get(key); t != nil {
			a.Thread = &threadBrief{Count: len(t.Messages), SkillsUsed: nonNil(t.SkillsUsed), Updated: t.Updated}
			if len(t.RuntimeSkills) > 0 && a.CanInvokeSkills {
				a.DiscoverableSkills = []string{}
				for _, n := range t.RuntimeSkills {
					if !contains(a.PreloadedSkills, n) {
						a.DiscoverableSkills = append(a.DiscoverableSkills, n)
					}
				}
			}
		}
		a.Busy = live[key]
	}
	for i := range skills {
		for _, a := range agents {
			if contains(a.PreloadedSkills, skills[i].Name) {
				skills[i].PreloadedBy = append(skills[i].PreloadedBy, a.Name)
			}
		}
	}
	hasMd := []string{}
	for _, f := range []string{"CLAUDE.md", "AGENTS.md"} {
		if fileExists(filepath.Join(sc.path, f)) {
			hasMd = append(hasMd, f)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"scope": scopeOut, "workspace": sc.ws, "agents": agents, "skills": skills,
		"runs": listRuns(sc.path), "hasClaudeMd": hasMd,
	})
}

func (s *server) handleRunFile(w http.ResponseWriter, r *http.Request) {
	sc, err := s.resolveScope(r.PathValue("ws"), r.PathValue("repo"))
	if err != nil {
		writeErr(w, err)
		return
	}
	base := filepath.Join(sc.path, ".agent", "runs")
	f := filepath.Join(base, r.PathValue("run"), r.PathValue("file"))
	if !strings.HasPrefix(f, base+string(filepath.Separator)) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	b, err := os.ReadFile(f)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write(b)
}

func keyFor(r *http.Request) string {
	return threadKey(r.PathValue("ws")+"/"+r.PathValue("repo"), r.PathValue("agent"))
}

func (s *server) handleGetThread(w http.ResponseWriter, r *http.Request) {
	t := s.threads.get(keyFor(r))
	if t == nil {
		t = &Thread{}
	}
	t.Messages = append([]ChatMessage{}, t.Messages...)
	t.SkillsUsed = nonNil(t.SkillsUsed)
	writeJSON(w, http.StatusOK, t)
}

func (s *server) handleDeleteThread(w http.ResponseWriter, r *http.Request) {
	if err := s.threads.delete(keyFor(r)); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *server) handleStop(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	stop := s.live[keyFor(r)]
	s.mu.Unlock()
	if stop != nil {
		stop()
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *server) handlePermission(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Allow   bool   `json:"allow"`
		Message string `json:"message"`
	}
	_ = json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body)
	id := r.PathValue("permId")
	s.mu.Lock()
	wt := s.waiters[id]
	delete(s.waiters, id)
	s.mu.Unlock()
	if wt == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "expired"})
		return
	}
	d := Decision{Allow: body.Allow, Message: body.Message}
	if !d.Allow && d.Message == "" {
		d.Message = "Denied by user in Agent Console"
	}
	wt.ch <- d // buffered
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// stopAll stops every running chat (server shutdown).
func (s *server) stopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, stop := range s.live {
		stop()
	}
}

// wait blocks until running chats have saved their threads, or d passes.
func (s *server) wait(d time.Duration) {
	done := make(chan struct{})
	go func() { s.running.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(d):
	}
}

// ---------- chat ----------

func (s *server) handleChat(w http.ResponseWriter, r *http.Request) {
	sc, err := s.resolveScope(r.PathValue("ws"), r.PathValue("repo"))
	if err != nil {
		writeErr(w, err)
		return
	}
	if !sc.ok {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "Not cloned yet — create the workspace on disk first."})
		return
	}
	agent := r.PathValue("agent")
	var body struct {
		Message string `json:"message"`
		Mode    string `json:"mode"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 2<<20)).Decode(&body); err != nil || strings.TrimSpace(body.Message) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "message required"})
		return
	}
	mode := body.Mode
	if mode != "acceptEdits" && mode != "plan" {
		mode = "default"
	}
	key := threadKey(sc.id, agent)

	ctx, stop := context.WithCancel(r.Context())
	defer stop()
	s.mu.Lock()
	if _, busy := s.live[key]; busy {
		s.mu.Unlock()
		writeJSON(w, http.StatusConflict, map[string]string{"error": "agent is busy in this scope"})
		return
	}
	s.live[key] = stop
	s.running.Add(1)
	s.mu.Unlock()
	defer s.running.Done()

	var resume string
	s.threads.with(func(all map[string]*Thread) {
		t := all[key]
		if t == nil {
			t = &Thread{Messages: []ChatMessage{}, SkillsUsed: []string{}}
			all[key] = t
		}
		t.Messages = append(t.Messages, ChatMessage{Role: "user", Text: body.Message, Ts: nowISO()})
		resume = t.SessionID
	})

	sse, _ := newSSE(w)
	send := func(typ string, data map[string]any) {
		if data == nil {
			data = map[string]any{}
		}
		data["type"] = typ
		b, _ := json.Marshal(data)
		sse.raw("data: " + string(b) + "\n\n")
	}

	reply := ChatMessage{Role: "assistant", Tools: []ToolRef{}, Skills: []string{}}
	var skillsUsed, runtimeSkills []string
	var sessionID string

	claudeBin, findErr := FindClaude(s.opts.Claude)
	runErr := findErr
	if findErr == nil {
		opts := RunOptions{
			Claude:             claudeBin,
			Cwd:                sc.path,
			Prompt:             body.Message,
			Resume:             resume,
			PermissionMode:     mode,
			AppendSystemPrompt: workspaceContext(sc),
			Env:                append(os.Environ(), "PATH="+augmentPath(filepath.Dir(claudeBin), os.Getenv("PATH"))),
			CanUseTool: func(ctx context.Context, tool string, input json.RawMessage) Decision {
				if readOnlyTools[tool] {
					return Decision{Allow: true}
				}
				return s.askPermission(ctx, key, tool, input, send)
			},
			OnMessage: func(m Message) {
				translate(m, &reply, send, func(id string, skills []string) {
					sessionID, runtimeSkills = id, skills
				}, func(skill string) {
					if !contains(skillsUsed, skill) {
						skillsUsed = append(skillsUsed, skill)
					}
				})
			},
		}
		if agent != mainAgent {
			opts.Agent = agent
		}
		runErr = Run(ctx, opts)
	}

	if runErr != nil {
		aborted := ctx.Err() != nil
		msg := runErr.Error()
		if aborted {
			msg = "Stopped."
		}
		send("error", map[string]any{"message": msg})
		if reply.Text == "" {
			if aborted {
				reply.Text = "_Stopped._"
			} else {
				reply.Text = "**Error:** " + runErr.Error()
			}
		}
	}

	s.mu.Lock()
	for id, wt := range s.waiters {
		if wt.key == key {
			delete(s.waiters, id)
			wt.ch <- Decision{Message: "Session ended"}
		}
	}
	delete(s.live, key)
	s.mu.Unlock()

	reply.Ts = nowISO()
	s.threads.with(func(all map[string]*Thread) {
		t := all[key]
		if t == nil { // thread deleted mid-chat
			t = &Thread{Messages: []ChatMessage{}, SkillsUsed: []string{}}
			all[key] = t
		}
		if sessionID != "" {
			t.SessionID = sessionID
			t.RuntimeSkills = runtimeSkills
		}
		for _, sk := range skillsUsed {
			if !contains(t.SkillsUsed, sk) {
				t.SkillsUsed = append(t.SkillsUsed, sk)
			}
		}
		t.Messages = append(t.Messages, reply)
		t.Updated = reply.Ts
	})
	if err := s.threads.save(); err != nil {
		fmt.Fprintf(s.opts.Log, "agent console: save threads: %v\n", err)
	}
	send("done", nil)
}

// askPermission shows a permission card in the browser and waits for the
// user, the turn ending, or the timeout.
func (s *server) askPermission(ctx context.Context, key, tool string, input json.RawMessage, send func(string, map[string]any)) Decision {
	id := newID()
	wt := &permWaiter{key: key, ch: make(chan Decision, 1)}
	s.mu.Lock()
	s.waiters[id] = wt
	s.mu.Unlock()
	send("permission", map[string]any{"permId": id, "tool": tool, "input": input})

	timer := time.NewTimer(permissionTimeout)
	defer timer.Stop()
	select {
	case d := <-wt.ch:
		return d
	case <-timer.C:
		s.dropWaiter(id)
		return Decision{Message: "No response in 10 minutes"}
	case <-ctx.Done():
		s.dropWaiter(id)
		return Decision{Message: "Aborted", Interrupt: true}
	}
}

func (s *server) dropWaiter(id string) {
	s.mu.Lock()
	delete(s.waiters, id)
	s.mu.Unlock()
}

// workspaceContext tells an agent at a workspace root how it is laid out.
func workspaceContext(sc *scope) string {
	if sc.repo != nil {
		return ""
	}
	names := make([]string, 0, len(sc.ws.Repos))
	for _, r := range sc.ws.Repos {
		if r.Cloned {
			names = append(names, r.Name)
		} else {
			names = append(names, r.Name+" (not cloned)")
		}
	}
	return fmt.Sprintf("You are working at the root of the qail workspace %q (%s). "+
		"It contains these repos as subdirectories: %s. "+
		"Each repo has its own CLAUDE.md, .claude/skills and conventions — read the ones for the repo you touch before changing it.",
		sc.ws.Name, sc.ws.Path, strings.Join(names, ", "))
}

// translate turns one CLI message into UI events and updates reply.
func translate(m Message, reply *ChatMessage, send func(string, map[string]any), onInit func(string, []string), onSkill func(string)) {
	switch m.Type {
	case "system":
		if m.Subtype != "init" {
			return
		}
		var v struct {
			SessionID      string   `json:"session_id"`
			Model          string   `json:"model"`
			Skills         []string `json:"skills"`
			Tools          []string `json:"tools"`
			PermissionMode string   `json:"permissionMode"`
		}
		_ = json.Unmarshal(m.Raw, &v)
		onInit(v.SessionID, v.Skills)
		send("init", map[string]any{"model": v.Model, "skills": v.Skills, "tools": v.Tools, "permissionMode": v.PermissionMode, "sessionId": v.SessionID})

	case "stream_event":
		var v struct {
			Parent *string `json:"parent_tool_use_id"`
			Event  struct {
				Type  string `json:"type"`
				Delta struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"delta"`
			} `json:"event"`
		}
		_ = json.Unmarshal(m.Raw, &v)
		if v.Parent == nil && v.Event.Type == "content_block_delta" && v.Event.Delta.Type == "text_delta" {
			send("delta", map[string]any{"text": v.Event.Delta.Text})
		}

	case "assistant":
		var v struct {
			Parent  *string `json:"parent_tool_use_id"`
			Message struct {
				Content []struct {
					Type  string          `json:"type"`
					Text  string          `json:"text"`
					ID    string          `json:"id"`
					Name  string          `json:"name"`
					Input json.RawMessage `json:"input"`
				} `json:"content"`
			} `json:"message"`
		}
		_ = json.Unmarshal(m.Raw, &v)
		sub := v.Parent != nil
		for _, b := range v.Message.Content {
			switch b.Type {
			case "text":
				if sub {
					continue
				}
				if reply.Text != "" {
					reply.Text += "\n\n"
				}
				reply.Text += b.Text
				send("text_done", nil)
			case "tool_use":
				input := b.Input
				if len(input) == 0 {
					input = json.RawMessage("{}")
				}
				reply.Tools = append(reply.Tools, ToolRef{ID: b.ID, Name: b.Name, Sub: sub, Input: truncate(string(input), 400)})
				send("tool", map[string]any{"id": b.ID, "name": b.Name, "input": input, "sub": sub})
				if b.Name == "Skill" {
					var in map[string]any
					_ = json.Unmarshal(input, &in)
					skill := str(in["skill"])
					if skill == "" {
						skill = str(in["command"])
					}
					if skill == "" {
						skill = str(in["name"])
					}
					if skill != "" {
						if !contains(reply.Skills, skill) {
							reply.Skills = append(reply.Skills, skill)
						}
						onSkill(skill)
						send("skill", map[string]any{"skill": skill})
					}
				}
			}
		}

	case "user":
		var v struct {
			Message struct {
				Content json.RawMessage `json:"content"`
			} `json:"message"`
		}
		_ = json.Unmarshal(m.Raw, &v)
		var blocks []struct {
			Type      string          `json:"type"`
			ToolUseID string          `json:"tool_use_id"`
			IsError   bool            `json:"is_error"`
			Content   json.RawMessage `json:"content"`
		}
		if json.Unmarshal(v.Message.Content, &blocks) != nil {
			return
		}
		for _, b := range blocks {
			if b.Type != "tool_result" {
				continue
			}
			send("tool_result", map[string]any{"id": b.ToolUseID, "error": b.IsError, "preview": truncate(resultText(b.Content), 1500)})
		}

	case "result":
		var v struct {
			Subtype  string   `json:"subtype"`
			Cost     *float64 `json:"total_cost_usd"`
			Turns    int      `json:"num_turns"`
			Duration int64    `json:"duration_ms"`
			IsError  bool     `json:"is_error"`
		}
		_ = json.Unmarshal(m.Raw, &v)
		reply.Cost = v.Cost
		send("result", map[string]any{"subtype": v.Subtype, "cost": v.Cost, "turns": v.Turns, "durationMs": v.Duration, "isError": v.IsError})
	}
}

// resultText flattens a tool_result's content (a string or text blocks).
func resultText(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var blocks []struct {
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &blocks) == nil {
		parts := make([]string, len(blocks))
		for i, b := range blocks {
			parts[i] = b.Text
		}
		return strings.Join(parts, "\n")
	}
	return string(raw)
}

// ---------- helpers ----------

type sseStream struct {
	mu sync.Mutex
	w  http.ResponseWriter
	f  http.Flusher
}

func newSSE(w http.ResponseWriter) (*sseStream, bool) {
	f, ok := w.(http.Flusher)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	if ok {
		f.Flush()
	}
	return &sseStream{w: w, f: f}, ok
}

func (s *sseStream) raw(frame string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, _ = io.WriteString(s.w, frame)
	if s.f != nil {
		s.f.Flush()
	}
}

// sseLines writes each line of progress output as one `data:` event.
type sseLines struct{ s *sseStream }

func (l sseLines) Write(b []byte) (int, error) {
	for _, line := range strings.Split(strings.TrimRight(string(b), "\n"), "\n") {
		l.s.raw("data: " + line + "\n\n")
	}
	return len(b), nil
}

func oneLine(s string) string { return strings.ReplaceAll(s, "\n", " ") }

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	// Don't cut a UTF-8 sequence in half.
	for n > 0 && n < len(s) && s[n]&0xC0 == 0x80 {
		n--
	}
	return s[:n]
}

func nowISO() string { return time.Now().UTC().Format("2006-01-02T15:04:05.000Z") }

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	var he httpError
	if errors.As(err, &he) {
		writeJSON(w, he.status, map[string]string{"error": he.msg})
		return
	}
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
}
