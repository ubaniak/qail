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
	"fmt"
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
	Root          string                  `json:"root"`
	Editors       []EditorDTO             `json:"editors"`
	DefaultEditor string                  `json:"defaultEditor"`
	Workspaces    map[string]WorkspaceDTO `json:"workspaces"`
}

type EditorDTO struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

type WorkspaceDTO struct {
	Repos       []string  `json:"repos"`
	LastUsed    time.Time `json:"lastUsed"`
	Editor      string    `json:"editor,omitempty"`
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
	editors := make([]EditorDTO, len(cfg.Editors))
	for i, e := range cfg.Editors {
		editors[i] = EditorDTO{Name: e.Name, Command: e.Command}
	}
	out := ConfigDTO{
		Root:          cfg.Root,
		Editors:       editors,
		DefaultEditor: cfg.DefaultEditor,
		Workspaces:    make(map[string]WorkspaceDTO, len(cfg.Workspaces)),
	}
	for name, profile := range cfg.Workspaces {
		out.Workspaces[name] = WorkspaceDTO{
			Repos:       profile.Repos,
			LastUsed:    profile.LastUsed,
			Editor:      profile.Editor,
			PostInstall: cfg.PostInstallScripts.Workspace[name],
		}
	}
	return out, nil
}

func (b *Bindings) SetRoot(value string) error { return actions.SetRoot(b.store, value) }
func (b *Bindings) AddEditor(name, command string) error {
	return actions.AddEditor(b.store, name, command)
}
func (b *Bindings) RemoveEditor(name string) error    { return actions.RemoveEditor(b.store, name) }
func (b *Bindings) SetDefaultEditor(name string) error { return actions.SetDefaultEditor(b.store, name) }
func (b *Bindings) SetWorkspaceEditor(workspace, name string) error {
	return actions.SetWorkspaceEditor(b.store, workspace, name)
}

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
			Editor:      profile.Editor,
			PostInstall: postInstall[name],
		}
	}
	return out, nil
}

// AddWorkspace creates a workspace and streams progress as custom Wails
// events. The pipe + scanner mirrors what the SSE handler does for HTTP:
// each line becomes one "workspace:progress" event; the outcome is
// "workspace:done" or "workspace:error". postInstall optionally attaches
// workspace-scoped scripts to the new workspace before the build runs.
func (b *Bindings) AddWorkspace(name string, packages []string, postInstall []string) error {
	return b.streamWorkspace(func(out io.Writer) error {
		return actions.AddWorkspace(b.ctx, b.store, name, packages, postInstall, out)
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

// ListOrphanWorkspaces returns directory names under the configured root
// that are NOT registered workspaces — candidates for cleanup. Settings →
// Cleanup tab subscribes via callService and pairs this with
// RemoveOrphanWorkspaces.
func (b *Bindings) ListOrphanWorkspaces() ([]string, error) {
	return actions.ListOrphanWorkspaces(b.store)
}

// RemoveOrphanWorkspaces deletes the named directories from the root.
// The action refuses to delete registered workspaces even if the caller
// asks, so the UI doesn't need its own guard.
func (b *Bindings) RemoveOrphanWorkspaces(names []string) error {
	return actions.RemoveOrphanWorkspaces(b.store, names)
}

// OrphanPath returns the absolute path of an orphan directory under the
// configured workspace root. UI uses it for clipboard copy.
func (b *Bindings) OrphanPath(name string) (string, error) {
	return actions.OrphanPath(b.store, name)
}

// OpenOrphanEditor spawns the default editor against an orphan directory.
// Mirrors OpenEditor for unregistered dirs surfaced by Settings → Cleanup.
func (b *Bindings) OpenOrphanEditor(name string) error {
	return actions.LaunchEditorForOrphan(b.store, name, "")
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
	return b.OpenCommandWith(name, "")
}

func (b *Bindings) OpenCommandWith(name, editor string) (OpenCommandDTO, error) {
	cmd, err := actions.OpenWorkspaceCommandWith(b.store, name, editor)
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

// OpenEditorWith spawns a specific editor (by name) against the workspace.
func (b *Bindings) OpenEditorWith(name, editor string) error {
	return actions.LaunchEditorWith(b.store, name, editor)
}

// ExplorePath returns the workspace path so JS can open it via
// app.Browser.OpenURL("file://"+path) in v3.
func (b *Bindings) ExplorePath(name string) (string, error) {
	return actions.ExploreWorkspacePath(b.store, name)
}

// --- scripts ----------------------------------------------------------------

// parseScope turns the JS-side string into a scripts.Scope, erroring on
// any unrecognised value so the binding never silently writes the wrong
// bucket.
func parseScope(scope string) (scripts.Scope, error) {
	s := scripts.Scope(scope)
	if !s.Valid() {
		return "", fmt.Errorf("invalid scope %q (want %q or %q)", scope, scripts.ScopeWorkspace, scripts.ScopeRepo)
	}
	return s, nil
}

func (b *Bindings) ListScripts(scope string) ([]string, error) {
	s, err := parseScope(scope)
	if err != nil {
		return nil, err
	}
	return scripts.Default().ListScripts(s)
}

func (b *Bindings) AddScript(name, scope string) error {
	s, err := parseScope(scope)
	if err != nil {
		return err
	}
	return scripts.Default().CreateBashScript(name, s)
}

func (b *Bindings) RemoveScript(name, scope string) error {
	s, err := parseScope(scope)
	if err != nil {
		return err
	}
	return scripts.Default().RemoveScript(name, s)
}

// ReadScript returns the on-disk contents of the named script so the UI
// can preview it without delegating to an external editor.
func (b *Bindings) ReadScript(name, scope string) (string, error) {
	s, err := parseScope(scope)
	if err != nil {
		return "", err
	}
	return scripts.Default().ReadScript(name, s)
}

// WriteScript replaces the contents of an existing script so the UI can
// edit a script in place without shelling out to an external editor.
func (b *Bindings) WriteScript(name, scope, contents string) error {
	s, err := parseScope(scope)
	if err != nil {
		return err
	}
	return scripts.Default().WriteScript(name, s, contents)
}

func (b *Bindings) ScriptsDir() (string, error) {
	return scripts.Default().GetScriptDir()
}

// RunWorkspaceScript streams a workspace-scoped script execution against
// the named workspace's on-disk path. Progress flows through the same
// custom-event channel as workspace creation so the existing
// ProgressDrawer subscribes once.
func (b *Bindings) RunWorkspaceScript(workspace, script string) error {
	return b.streamWorkspace(func(out io.Writer) error {
		return actions.RunWorkspaceScript(b.ctx, b.store, workspace, script, out)
	})
}

// RunRepoScript streams a repo-scoped script execution against
// workspace/repo. workspace is required because the script runs inside
// the repo's clone path under that workspace's root.
func (b *Bindings) RunRepoScript(workspace, repo, script string) error {
	return b.streamWorkspace(func(out io.Writer) error {
		return actions.RunRepoScript(b.ctx, b.store, workspace, repo, script, out)
	})
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
