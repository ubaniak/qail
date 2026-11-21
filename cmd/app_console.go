package cmd

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/ubaniak/qail/internal/api"
	"github.com/ubaniak/qail/internal/config"
	"github.com/ubaniak/qail/internal/console"
)

// appAPIAddr is where the desktop backend serves the qail HTTP API for the
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

// newAppConsole builds the console Manager for the desktop backend. The
// backend serves the API itself, so the console must not spawn `qail serve`.
func newAppConsole(apiURL string) *console.Manager {
	self, _ := os.Executable()
	return console.New(console.Options{
		Dir:     consoleDir(self),
		DataDir: consoleDataDir(),
		QailURL: apiURL,
		QailBin: self,
		Log:     log.Writer(),
	})
}
