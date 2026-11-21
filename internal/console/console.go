// Package console supervises the Agent Console: a local web UI for Claude
// Code agents scoped to qail workspaces and repos (ADR-0018).
//
// The console's agent runtime is the Claude Agent SDK, which ships for
// TypeScript and Python only, so the console runs as a Node sidecar
// (`node server.mjs`). This package locates that install, starts and stops
// the process, waits for it to become healthy, and opens it in the user's
// browser. The console reads workspaces and repos from qail's HTTP API
// (internal/api), so it never touches the SQLite store directly.
package console

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ubaniak/qail/internal/runner"
)

// DefaultPort is the console's loopback port.
const DefaultPort = 4317

// ErrNotInstalled means the console files or its node_modules are missing.
var ErrNotInstalled = errors.New("agent console is not installed (run `make console-install`)")

// ErrNoNode means no Node.js binary could be found.
var ErrNoNode = errors.New("node not found (install Node ≥ 22 or set QAIL_NODE)")

// Runner is the narrow subprocess seam used to open the browser.
type Runner interface {
	Run(ctx context.Context, cmd runner.Command) (runner.Result, error)
}

// Options configures a Manager. Zero values pick sensible defaults.
type Options struct {
	// Dir is the console install directory (contains server.mjs).
	Dir string
	// Port is the console's loopback port. Default DefaultPort.
	Port int
	// QailURL is the qail HTTP API the console reads workspaces from.
	QailURL string
	// QailBin is passed to the console so it can start `qail serve` itself
	// when QailURL is down and AutostartQail is true.
	QailBin string
	// AutostartQail lets the console spawn `qail serve`. The desktop app
	// serves the API in-process, so it leaves this false.
	AutostartQail bool
	// Node is the node binary. "" = auto-detect (see FindNode).
	Node string
	// Log receives the console's stdout/stderr. nil = discarded.
	Log io.Writer
	// Runner opens the browser. nil = runner.NewOS().
	Runner Runner
	// GOOS overrides runtime.GOOS (tests).
	GOOS string
}

// Manager owns at most one console process.
type Manager struct {
	opts   Options
	client *http.Client

	mu   sync.Mutex
	proc *exec.Cmd
	done chan struct{}
}

// New returns a Manager with defaults applied.
func New(o Options) *Manager {
	if o.Port == 0 {
		o.Port = DefaultPort
	}
	if o.QailURL == "" {
		o.QailURL = "http://127.0.0.1:8765"
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

// Installed reports whether Dir holds a runnable console.
func (m *Manager) Installed() bool {
	return fileExists(filepath.Join(m.opts.Dir, "server.mjs")) &&
		fileExists(filepath.Join(m.opts.Dir, "node_modules"))
}

// Healthy reports whether a console (ours or another process's) answers on Port.
func (m *Manager) Healthy(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.URL("/"), nil)
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

// Running reports whether this Manager owns a live console process.
func (m *Manager) Running() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.proc != nil
}

// Start launches the console unless one already answers on Port, then
// waits up to 15s for it to become healthy.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.proc != nil || m.Healthy(ctx) {
		return nil
	}
	if !m.Installed() {
		return fmt.Errorf("%w: looked in %s", ErrNotInstalled, m.opts.Dir)
	}
	node, err := FindNode(m.opts.Node)
	if err != nil {
		return err
	}

	autostart := "0"
	if m.opts.AutostartQail {
		autostart = "1"
	}
	cmd := exec.Command(node, "server.mjs")
	cmd.Dir = m.opts.Dir
	cmd.Stdout = m.opts.Log
	cmd.Stderr = m.opts.Log
	cmd.Env = append(os.Environ(),
		"PORT="+strconv.Itoa(m.opts.Port),
		"QAIL_URL="+m.opts.QailURL,
		"QAIL_AUTOSTART="+autostart,
		// GUI apps on macOS start with a minimal PATH; the agent's tools
		// (git, npm, go…) need the user's usual locations.
		"PATH="+augmentPath(filepath.Dir(node), os.Getenv("PATH")),
	)
	if m.opts.QailBin != "" {
		cmd.Env = append(cmd.Env, "QAIL_BIN="+m.opts.QailBin)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start agent console: %w", err)
	}

	done := make(chan struct{})
	m.proc, m.done = cmd, done
	go func() {
		_ = cmd.Wait()
		m.mu.Lock()
		if m.proc == cmd {
			m.proc = nil
		}
		m.mu.Unlock()
		close(done)
	}()

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-done:
			return errors.New("agent console exited during startup (see qail logs)")
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
		if m.Healthy(ctx) {
			return nil
		}
	}
	return errors.New("agent console did not become healthy within 15s")
}

// Stop terminates the console process this Manager started. A console
// started by someone else is left alone.
func (m *Manager) Stop() {
	m.mu.Lock()
	cmd, done := m.proc, m.done
	m.proc = nil
	m.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return
	}
	if m.opts.GOOS == "windows" {
		_ = cmd.Process.Kill()
	} else {
		_ = cmd.Process.Signal(os.Interrupt)
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		_ = cmd.Process.Kill()
	}
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

// FindNode resolves the node binary: explicit, $QAIL_NODE, PATH, then the
// usual install locations (Homebrew, Volta, nvm) since a menubar app does
// not inherit the shell's PATH.
func FindNode(explicit string) (string, error) {
	for _, c := range []string{explicit, os.Getenv("QAIL_NODE")} {
		if c != "" && fileExists(c) {
			return c, nil
		}
	}
	if p, err := exec.LookPath("node"); err == nil {
		return p, nil
	}
	home, _ := os.UserHomeDir()
	candidates := []string{
		"/opt/homebrew/bin/node",
		"/usr/local/bin/node",
		filepath.Join(home, ".homebrew", "bin", "node"),
		filepath.Join(home, ".volta", "bin", "node"),
	}
	candidates = append(candidates, nvmNodes(filepath.Join(home, ".nvm", "versions", "node"))...)
	for _, c := range candidates {
		if fileExists(c) {
			return c, nil
		}
	}
	return "", ErrNoNode
}

// nvmNodes lists nvm-installed node binaries, newest version first.
func nvmNodes(dir string) []string {
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
		out = append(out, filepath.Join(dir, v, "bin", "node"))
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

func augmentPath(nodeDir, current string) string {
	home, _ := os.UserHomeDir()
	parts := []string{nodeDir, "/opt/homebrew/bin", "/usr/local/bin", filepath.Join(home, ".qail", "bin")}
	if current != "" {
		parts = append(parts, current)
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
