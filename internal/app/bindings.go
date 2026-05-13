// Package app is the Wails-facing layer of qail: the QailService bound
// to the JS runtime via v3 generated bindings, plus systray wiring lives
// in cmd/app.go. Every method here funnels into internal/actions, so the
// desktop app, the HTTP server, and the CLI all share the same business
// logic.
//
// Progress streaming uses Wails v3 custom events ("workspace:progress",
// "workspace:done", "workspace:error"). The frontend subscribes via
// `@wailsio/runtime` Events.On.
package app

import (
	"bufio"
	"context"
	"io"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/ubaniak/qail/internal/actions"
	"github.com/ubaniak/qail/internal/config"
	"github.com/ubaniak/qail/internal/scripts"
	"github.com/ubaniak/qail/internal/tmux"
)

// Bindings is the v3 service exposed to the frontend. wails3 generate
// bindings reads its method set and writes typed wrappers into
// frontend/bindings/. The struct holds the config store and the App so
// methods can fire custom events from any goroutine.
type Bindings struct {
	store config.Store
	app   *application.App
	ctx   context.Context
}

// New constructs a Bindings. The App is wired here (rather than via
// ServiceStartup) so the ctor stays callable from cmd/app.go before
// `app.Run()` blocks.
func New(s config.Store, app *application.App) *Bindings {
	return &Bindings{store: s, app: app, ctx: context.Background()}
}

// ServiceStartup is the v3 lifecycle hook — called once when the service
// registers, before the webview loads. We just stash ctx so handlers can
// honor cancellation if needed.
func (b *Bindings) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	b.ctx = ctx
	return nil
}

// ServiceName satisfies the optional ServiceName interface so the
// generated TS bindings live in a stable namespace regardless of pkg
// path. wails3 uses it as the namespace for the generated file.
func (*Bindings) ServiceName() string { return "Bindings" }

// --- config -----------------------------------------------------------------

// ConfigDTO mirrors the HTTP layer's configResponse so the same React
// types compile against either backend.
type ConfigDTO struct {
	Root       string                  `json:"root"`
	Editor     string                  `json:"editor"`
	Workspaces map[string]WorkspaceDTO `json:"workspaces"`
}

type WorkspaceDTO struct {
	Repos       []string  `json:"repos"`
	LastUsed    time.Time `json:"lastUsed"`
	PostInstall []string  `json:"postInstall,omitempty"`
}

type RepoDTO struct {
	URL         string   `json:"url"`
	PostInstall []string `json:"postInstall,omitempty"`
}

func (b *Bindings) GetConfig() (ConfigDTO, error) {
	cfg, err := b.store.Read()
	if err != nil {
		return ConfigDTO{}, err
	}
	out := ConfigDTO{
		Root:       cfg.Root,
		Editor:     cfg.Editor,
		Workspaces: make(map[string]WorkspaceDTO, len(cfg.Workspaces)),
	}
	for name, profile := range cfg.Workspaces {
		out.Workspaces[name] = WorkspaceDTO{
			Repos:       profile.Repos,
			LastUsed:    profile.LastUsed,
			PostInstall: cfg.PostInstallScripts.Workspace[name],
		}
	}
	return out, nil
}

func (b *Bindings) SetRoot(value string) error   { return actions.SetRoot(b.store, value) }
func (b *Bindings) SetEditor(value string) error { return actions.SetEditor(b.store, value) }

// --- repos ------------------------------------------------------------------

func (b *Bindings) ListRepos() (map[string]RepoDTO, error) {
	repos, postInstall, err := actions.ListRepos(b.store)
	if err != nil {
		return nil, err
	}
	out := make(map[string]RepoDTO, len(repos))
	for name, url := range repos {
		out[name] = RepoDTO{URL: url, PostInstall: postInstall[name]}
	}
	return out, nil
}

func (b *Bindings) AddRepo(name, url string) error {
	return actions.AddRepo(b.store, name, url)
}

func (b *Bindings) RemoveRepos(names []string) error {
	return actions.RemoveRepos(b.store, names)
}

func (b *Bindings) SetRepoPostInstall(repo string, scriptsList []string) error {
	return actions.SetRepoPostInstall(b.store, repo, scriptsList)
}

// --- workspaces -------------------------------------------------------------

func (b *Bindings) ListWorkspaces() (map[string]WorkspaceDTO, error) {
	workspaces, postInstall, err := actions.ListWorkspaces(b.store)
	if err != nil {
		return nil, err
	}
	out := make(map[string]WorkspaceDTO, len(workspaces))
	for name, profile := range workspaces {
		out[name] = WorkspaceDTO{
			Repos:       profile.Repos,
			LastUsed:    profile.LastUsed,
			PostInstall: postInstall[name],
		}
	}
	return out, nil
}

// AddWorkspace creates a workspace and streams progress as custom Wails
// events. The pipe + scanner mirrors what the SSE handler does for HTTP:
// each line becomes one "workspace:progress" event; the outcome is
// "workspace:done" or "workspace:error".
func (b *Bindings) AddWorkspace(name string, packages []string) error {
	return b.streamWorkspace(func(out io.Writer) error {
		return actions.AddWorkspace(b.ctx, b.store, name, packages, out)
	})
}

func (b *Bindings) EditWorkspace(name string, packages []string) error {
	return b.streamWorkspace(func(out io.Writer) error {
		return actions.EditWorkspace(b.ctx, b.store, name, packages, out)
	})
}

func (b *Bindings) CloneWorkspace(dst string, packages []string) error {
	return b.streamWorkspace(func(out io.Writer) error {
		return actions.CloneWorkspace(b.ctx, b.store, dst, packages, out)
	})
}

func (b *Bindings) CreateWorkspaceOnDisk(name string) error {
	return b.streamWorkspace(func(out io.Writer) error {
		return actions.CreateWorkspaceOnDisk(b.ctx, b.store, name, out)
	})
}

func (b *Bindings) RemoveWorkspace(name string) error {
	return actions.RemoveWorkspace(b.store, name)
}

func (b *Bindings) SetWorkspacePostInstall(name string, scriptsList []string) error {
	return actions.SetWorkspacePostInstall(b.store, name, scriptsList)
}

// CdWorkspace returns the workspace's on-disk path so JS can copy
// `cd <path>` to the clipboard.
func (b *Bindings) CdWorkspace(name string) (string, error) {
	return actions.CdWorkspace(b.store, name)
}

// AttachCommand returns the shell command to attach to (or launch) the
// workspace's tmux session.
func (b *Bindings) AttachCommand(name string) (string, error) {
	return actions.MuxWorkspace(b.ctx, b.store, name)
}

// OpenCommandDTO is returned to JS so the UI can display + copy the
// editor invocation — the app cannot exec an editor that owns a TTY.
type OpenCommandDTO struct {
	Editor  string `json:"editor"`
	Path    string `json:"path"`
	Command string `json:"command"`
}

func (b *Bindings) OpenCommand(name string) (OpenCommandDTO, error) {
	cmd, err := actions.OpenWorkspaceCommand(b.store, name)
	if err != nil {
		return OpenCommandDTO{}, err
	}
	return OpenCommandDTO{Editor: cmd.Editor, Path: cmd.Path, Command: cmd.String()}, nil
}

// OpenEditor spawns the configured editor against the workspace and
// returns immediately. The desktop app calls this on workspace click so
// the user gets a GUI launch instead of the legacy clipboard-copy flow.
// Use OpenCommand if the caller wants the invocation as data.
func (b *Bindings) OpenEditor(name string) error {
	return actions.LaunchEditor(b.store, name)
}

// ExplorePath returns the workspace path so JS can open it via
// app.Browser.OpenURL("file://"+path) in v3.
func (b *Bindings) ExplorePath(name string) (string, error) {
	return actions.ExploreWorkspacePath(b.store, name)
}

// --- scripts ----------------------------------------------------------------

func (b *Bindings) ListScripts() ([]string, error) {
	return scripts.Default().ListScripts()
}

func (b *Bindings) AddScript(name string) error {
	return scripts.Default().CreateBashScript(name)
}

func (b *Bindings) RemoveScript(name string) error {
	return scripts.Default().RemoveScript(name)
}

func (b *Bindings) ScriptsDir() (string, error) {
	return scripts.Default().GetScriptDir()
}

// --- tmux -------------------------------------------------------------------

func (b *Bindings) ListMuxSessions() ([]string, error) {
	return tmux.Default().ListSessions(b.ctx)
}

func (b *Bindings) RemoveMuxSession(name string) error {
	return tmux.Default().RemoveSession(b.ctx, name)
}

// --- helpers ----------------------------------------------------------------

// streamWorkspace runs fn against a pipe whose lines are emitted as
// "workspace:progress" v3 custom events. Final outcome events:
// "workspace:done" or "workspace:error". The scanner goroutine drains
// the pipe so workspace progress can never block on a full buffer.
func (b *Bindings) streamWorkspace(fn func(io.Writer) error) error {
	pr, pw := io.Pipe()
	doneScanning := make(chan struct{})
	go func() {
		defer close(doneScanning)
		sc := bufio.NewScanner(pr)
		sc.Buffer(make([]byte, 0, 1024*64), 1024*1024)
		for sc.Scan() {
			b.emit("workspace:progress", sc.Text())
		}
	}()

	err := fn(pw)
	pw.Close()
	<-doneScanning

	if err != nil {
		b.emit("workspace:error", err.Error())
		return err
	}
	b.emit("workspace:done", "ok")
	return nil
}

// emit is the v3 replacement for runtime.EventsEmit. We package the
// payload into a single Data field; the frontend handler treats the
// CustomEvent as `{ name, data, sender }` and reads `data`.
func (b *Bindings) emit(name string, data any) {
	if b.app == nil {
		return
	}
	b.app.Event.EmitEvent(&application.CustomEvent{Name: name, Data: data})
}
