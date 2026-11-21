package cmd

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/ubaniak/qail/internal/actions"
	"github.com/ubaniak/qail/internal/api"
	"github.com/ubaniak/qail/internal/config"
	"github.com/ubaniak/qail/internal/console"
)

// appAPIAddr is where the menubar app serves the qail HTTP API for the
// Agent Console. Same default as `qail serve` so either can back it.
const appAPIAddr = "127.0.0.1:8765"

// startEmbeddedAPI serves internal/api in-process for the Agent Console.
// If something healthy already owns the port (a `qail serve` in a
// terminal), it is reused and shutdown is a no-op.
func startEmbeddedAPI(s config.Store) (baseURL string, shutdown func()) {
	baseURL = "http://" + appAPIAddr
	noop := func() {}

	client := &http.Client{Timeout: 500 * time.Millisecond}
	if res, err := client.Get(baseURL + "/api/health"); err == nil {
		res.Body.Close()
		if res.StatusCode == http.StatusOK {
			return baseURL, noop
		}
	}

	ln, err := net.Listen("tcp", appAPIAddr)
	if err != nil {
		log.Printf("agent console: qail api not started: %v", err)
		return baseURL, noop
	}
	srv := &http.Server{
		Handler:           api.New(s).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("agent console: qail api: %v", err)
		}
	}()
	return baseURL, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}
}

// newAppConsole builds the console Manager for the menubar app. The app
// serves the API itself, so the console must not spawn `qail serve`.
func newAppConsole(apiURL string) *console.Manager {
	self, _ := os.Executable()
	return console.New(console.Options{
		Dir:     consoleDir(),
		QailURL: apiURL,
		QailBin: self,
		Log:     log.Writer(),
	})
}

// addConsoleSubmenu appends the "Agent Console" submenu:
//
//	Agent Console ▸ Open Agent Console
//	                ───────────
//	                <workspace> ▸ Whole workspace
//	                              ───────────
//	                              <repo> …
//	                ───────────
//	                Refresh workspaces
//	                Stop Agent Console
//
// Workspaces are read when the menu is built; rebuild re-creates the tray
// menu so new workspaces appear (also triggered after every open).
func addConsoleSubmenu(menu *application.Menu, s config.Store, mgr *console.Manager, rebuild func(), onErr func(error)) {
	sub := menu.AddSubmenu("Agent Console")

	open := func(path string) {
		go func() {
			if err := mgr.Open(context.Background(), path); err != nil {
				log.Printf("agent console: %v", err)
				onErr(err)
			}
			rebuild()
		}()
	}

	if !mgr.Installed() {
		sub.Add("Not installed — run `make console-install`").SetEnabled(false)
		sub.AddSeparator()
	}
	sub.Add("Open Agent Console").OnClick(func(*application.Context) { open("/") })

	if workspaces, _, err := actions.ListWorkspaces(s); err == nil && len(workspaces) > 0 {
		sub.AddSeparator()
		names := make([]string, 0, len(workspaces))
		for name := range workspaces {
			names = append(names, name)
		}
		// Most recently used first, matching the qail window.
		sort.Slice(names, func(i, j int) bool {
			return workspaces[names[i]].LastUsed.After(workspaces[names[j]].LastUsed)
		})
		for _, name := range names {
			wsMenu := sub.AddSubmenu(name)
			wsMenu.Add("Whole workspace").OnClick(func(*application.Context) { open(console.ScopePath(name, "")) })
			if repos := workspaces[name].Repos; len(repos) > 0 {
				wsMenu.AddSeparator()
				for _, repo := range repos {
					wsMenu.Add(repo).OnClick(func(*application.Context) { open(console.ScopePath(name, repo)) })
				}
			}
		}
	}

	sub.AddSeparator()
	sub.Add("Refresh workspaces").OnClick(func(*application.Context) { go rebuild() })
	stop := sub.Add("Stop Agent Console")
	stop.SetEnabled(mgr.Running())
	stop.OnClick(func(*application.Context) {
		go func() { mgr.Stop(); rebuild() }()
	})
}
