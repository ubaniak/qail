// Package console is the Agent Console: a local web UI for Claude Code
// agents scoped to qail workspaces and repos (ADR-0018, ADR-0020).
//
// It runs inside the qail binary. The UI is embedded (web/), the HTTP
// handlers read workspaces and repos straight from the config Store, and
// chats drive the user's installed `claude` CLI over its stream-json
// protocol (claude.go), answering its permission prompts from the
// browser. There is no Node process and no separate qail API server.
package console

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ubaniak/qail/internal/config"
	"github.com/ubaniak/qail/internal/runner"
)

// DefaultPort is the console's loopback port.
const DefaultPort = 4317

// Runner is the narrow subprocess seam used to open the browser.
type Runner interface {
	Run(ctx context.Context, cmd runner.Command) (runner.Result, error)
}

// Options configures a Manager. Zero values pick sensible defaults.
type Options struct {
	// Store is qail's config: workspaces, repos and root. Required to Start.
	Store config.Store
	// DataDir is where chat threads are kept (store.json).
	DataDir string
	// Port is the console's loopback port. Default DefaultPort.
	Port int
	// Claude is the Claude Code CLI. "" = FindClaude at each chat, so
	// installing Claude Code while qail runs just works.
	Claude string
	// Log receives server errors. nil = discarded.
	Log io.Writer
	// Runner opens the browser. nil = runner.NewOS().
	Runner Runner
	// GOOS overrides runtime.GOOS (tests).
	GOOS string
}

// Manager owns at most one in-process console server.
type Manager struct {
	opts   Options
	client *http.Client

	mu  sync.Mutex
	srv *http.Server
	app *server
}

// New returns a Manager with defaults applied.
func New(o Options) *Manager {
	if o.Port == 0 {
		o.Port = DefaultPort
	}
	if o.Log == nil {
		o.Log = io.Discard
	}
	if o.Runner == nil {
		o.Runner = runner.NewOS()
	}
	if o.GOOS == "" {
		o.GOOS = runtime.GOOS
	}
	return &Manager{opts: o, client: &http.Client{Timeout: 700 * time.Millisecond}}
}

// URL returns the console URL for path (which should start with "/").
func (m *Manager) URL(path string) string {
	if path == "" {
		path = "/"
	}
	return fmt.Sprintf("http://127.0.0.1:%d%s", m.opts.Port, path)
}

// ScopePath returns the deep-link path for a workspace, or for one repo in
// it when repo is non-empty. The console reads it from the URL hash.
func ScopePath(workspace, repo string) string {
	if workspace == "" {
		return "/"
	}
	p := "/#/ws/" + url.PathEscape(workspace)
	if repo != "" {
		p += "/" + url.PathEscape(repo)
	}
	return p
}

// ClaudeFound reports whether the Claude Code CLI can be found. The
// console still opens without it; chats explain what is missing.
func (m *Manager) ClaudeFound() bool {
	_, err := FindClaude(m.opts.Claude)
	return err == nil
}

// Healthy reports whether a console (ours or another qail's) answers on Port.
func (m *Manager) Healthy(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.URL("/api/health"), nil)
	if err != nil {
		return false
	}
	res, err := m.client.Do(req)
	if err != nil {
		return false
	}
	res.Body.Close()
	return res.StatusCode == http.StatusOK
}

// Running reports whether this Manager is serving the console.
func (m *Manager) Running() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.srv != nil
}

// Start serves the console on 127.0.0.1:Port unless a console already
// answers there (e.g. the menubar app's, when `qail console` runs in a
// terminal), in which case that one is reused.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.srv != nil || m.Healthy(ctx) {
		return nil
	}
	if m.opts.Store == nil {
		return errors.New("agent console: no config store")
	}
	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(m.opts.Port)))
	if err != nil {
		return fmt.Errorf("agent console: %w", err)
	}
	app, err := newServer(m.opts)
	if err != nil {
		ln.Close()
		return err
	}
	srv := &http.Server{
		Handler:           app.handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ErrorLog:          log.New(m.opts.Log, "agent console: ", 0),
	}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(m.opts.Log, "agent console: %v\n", err)
		}
	}()
	m.srv, m.app = srv, app
	return nil
}

// Stop stops running chats and shuts the server down. A console served by
// another process is left alone.
func (m *Manager) Stop() {
	m.mu.Lock()
	srv, app := m.srv, m.app
	m.srv, m.app = nil, nil
	m.mu.Unlock()
	if srv == nil {
		return
	}
	app.stopAll()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		_ = srv.Close()
	}
	app.wait(3 * time.Second)
}

// Open starts the console if needed and opens path in the browser.
func (m *Manager) Open(ctx context.Context, path string) error {
	if err := m.Start(ctx); err != nil {
		return err
	}
	return m.OpenBrowser(ctx, m.URL(path))
}

// OpenBrowser opens u in the default browser via the platform opener.
func (m *Manager) OpenBrowser(ctx context.Context, u string) error {
	var c runner.Command
	switch m.opts.GOOS {
	case "darwin":
		c = runner.Command{Name: "open", Args: []string{u}}
	case "windows":
		c = runner.Command{Name: "rundll32", Args: []string{"url.dll,FileProtocolHandler", u}}
	default:
		c = runner.Command{Name: "xdg-open", Args: []string{u}}
	}
	res, err := m.opts.Runner.Run(ctx, c)
	if err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("open browser: %s exited %d: %s", c.Name, res.ExitCode, strings.TrimSpace(string(res.Stderr)))
	}
	return nil
}

// nvmBins lists nvm-installed Node bin directories, newest version first.
// Claude Code installed with `npm i -g` under nvm lives there.
func nvmBins(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var versions []string
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "v") {
			versions = append(versions, e.Name())
		}
	}
	sort.Slice(versions, func(i, j int) bool { return versionLess(versions[j], versions[i]) })
	out := make([]string, 0, len(versions))
	for _, v := range versions {
		out = append(out, filepath.Join(dir, v, "bin"))
	}
	return out
}

// versionLess compares "vX.Y.Z" strings numerically.
func versionLess(a, b string) bool {
	pa, pb := strings.Split(strings.TrimPrefix(a, "v"), "."), strings.Split(strings.TrimPrefix(b, "v"), ".")
	for i := 0; i < len(pa) && i < len(pb); i++ {
		na, _ := strconv.Atoi(pa[i])
		nb, _ := strconv.Atoi(pb[i])
		if na != nb {
			return na < nb
		}
	}
	return len(pa) < len(pb)
}

// augmentPath puts the claude CLI's directory and the usual tool
// locations on PATH. GUI apps on macOS start with a minimal PATH, and the
// agent's tools (git, npm, go…) need the user's usual locations.
func augmentPath(claudeDir, current string) string {
	home, _ := os.UserHomeDir()
	parts := []string{claudeDir, "/opt/homebrew/bin", "/usr/local/bin", filepath.Join(home, ".local", "bin"), filepath.Join(home, ".qail", "bin")}
	if current != "" {
		parts = append(parts, filepath.SplitList(current)...)
	}
	parts = append(parts, "/usr/bin", "/bin", "/usr/sbin", "/sbin")
	seen := map[string]bool{}
	out := parts[:0]
	for _, p := range parts {
		if p != "" && !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return strings.Join(out, string(os.PathListSeparator))
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
