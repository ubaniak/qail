// Package tray is the system tray icon for `qail tray` (ADR-0021). It
// replaces the Electron tray: left-click opens the qail UI in the default
// browser, right-click shows the menu (Open qail, the Agent Console
// submenu, Quit). It uses fyne.io/systray, which works on macOS (cgo),
// Windows, and Linux desktops with a StatusNotifierItem tray (KDE, XFCE,
// Cinnamon, ...; GNOME needs the AppIndicator extension).
package tray

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"sync"
	"time"

	"fyne.io/systray"

	qailapp "github.com/ubaniak/qail/internal/app"
)

//go:embed icon.png
var iconPNG []byte

// Console is the slice of Bindings the menu needs. *app.Bindings
// satisfies it; tests can fake it.
type Console interface {
	ConsoleStatus() (qailapp.ConsoleStatusDTO, error)
	OpenConsole(workspace, repo string) error
	StopConsole() error
}

// Options configures the tray.
type Options struct {
	// UIURL is the page "Open qail" opens.
	UIURL string
	// OpenURL opens a URL in the default browser.
	OpenURL func(url string) error
	// Console drives the Agent Console submenu.
	Console Console
	// Log receives errors (menu actions have no window to report them in).
	Log io.Writer
	// OnReady runs once the icon is up; OnExit runs as the tray quits.
	OnReady, OnExit func()
	// RefreshEvery re-reads workspaces for the Agent Console submenu and
	// rebuilds the menu when they changed. Zero means 30s.
	RefreshEvery time.Duration
}

type tray struct {
	opts Options

	mu       sync.Mutex
	cancel   context.CancelFunc // stops the current menu's click loops
	lastMenu string             // status the current menu was built from
}

// Run shows the tray icon and blocks until Quit. It must be called from
// the main goroutine (macOS needs the main thread; systray locks it in
// its init).
func Run(o Options) {
	if o.RefreshEvery == 0 {
		o.RefreshEvery = 30 * time.Second
	}
	if o.Log == nil {
		o.Log = io.Discard
	}
	t := &tray{opts: o}
	systray.Run(t.onReady, func() {
		t.mu.Lock()
		if t.cancel != nil {
			t.cancel()
		}
		t.mu.Unlock()
		if o.OnExit != nil {
			o.OnExit()
		}
	})
}

// Quit removes the icon and makes Run return.
func Quit() { systray.Quit() }

func (t *tray) onReady() {
	setIcon()
	systray.SetTooltip("qail")
	// Left-click opens the UI; with a left-click handler set, right-click
	// shows the menu on every platform.
	systray.SetOnTapped(t.openUI)
	t.rebuild(true)

	go func() {
		tick := time.NewTicker(t.opts.RefreshEvery)
		defer tick.Stop()
		for range tick.C {
			t.rebuild(false)
		}
	}()
	if t.opts.OnReady != nil {
		t.opts.OnReady()
	}
}

func setIcon() {
	if runtime.GOOS == "windows" {
		systray.SetIcon(pngToICO(iconPNG))
		return
	}
	systray.SetIcon(iconPNG)
}

func (t *tray) openUI() {
	if err := t.opts.OpenURL(t.opts.UIURL); err != nil {
		t.logf("open qail: %v", err)
	}
}

func (t *tray) logf(format string, args ...any) {
	fmt.Fprintf(t.opts.Log, "tray: "+format+"\n", args...)
}

// rebuild recreates the menu from the current console status. Unless
// force is set it does nothing when the status hasn't changed, so the
// periodic refresh doesn't flicker an open menu.
func (t *tray) rebuild(force bool) {
	status, err := t.opts.Console.ConsoleStatus()
	if err != nil {
		t.logf("agent console status: %v", err)
	}
	key, _ := json.Marshal(status)

	t.mu.Lock()
	defer t.mu.Unlock()
	if !force && string(key) == t.lastMenu {
		return
	}
	t.lastMenu = string(key)
	if t.cancel != nil {
		t.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel

	systray.ResetMenu()
	on := func(item *systray.MenuItem, fn func()) {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-item.ClickedCh:
					fn()
				}
			}
		}()
	}

	on(systray.AddMenuItem("Open qail", "Open qail in your browser"), t.openUI)
	systray.AddSeparator()
	t.addConsoleMenu(systray.AddMenuItem("Agent Console", ""), status, on)
	systray.AddSeparator()
	on(systray.AddMenuItem("Quit", "Quit qail"), systray.Quit)
}

// addConsoleMenu fills the Agent Console submenu, matching the Electron
// tray (ADR-0018, ADR-0020):
//
//	Open Agent Console
//	───────────
//	<workspace> ▸ Whole workspace / ─── / <repo> …
//	───────────
//	Refresh workspaces
//	Stop Agent Console
func (t *tray) addConsoleMenu(parent *systray.MenuItem, status qailapp.ConsoleStatusDTO, on func(*systray.MenuItem, func())) {
	run := func(name string, fn func() error) func() {
		return func() {
			if err := fn(); err != nil {
				t.logf("%s: %v", name, err)
			}
			t.rebuild(false)
		}
	}
	open := func(ws, repo string) func() {
		return run("open agent console", func() error { return t.opts.Console.OpenConsole(ws, repo) })
	}

	if !status.ClaudeFound {
		parent.AddSubMenuItem("Claude Code not found — install it to run agents", "").Disable()
		parent.AddSeparator()
	}
	on(parent.AddSubMenuItem("Open Agent Console", ""), open("", ""))

	if len(status.Workspaces) > 0 {
		parent.AddSeparator()
		for _, ws := range status.Workspaces {
			wsItem := parent.AddSubMenuItem(ws.Name, "")
			on(wsItem.AddSubMenuItem("Whole workspace", ""), open(ws.Name, ""))
			if len(ws.Repos) > 0 {
				wsItem.AddSeparator()
			}
			for _, repo := range ws.Repos {
				on(wsItem.AddSubMenuItem(repo, ""), open(ws.Name, repo))
			}
		}
	}

	parent.AddSeparator()
	on(parent.AddSubMenuItem("Refresh workspaces", ""), func() { t.rebuild(true) })
	stop := parent.AddSubMenuItem("Stop Agent Console", "")
	if !status.Running {
		stop.Disable()
	}
	on(stop, run("stop agent console", t.opts.Console.StopConsole))
}
