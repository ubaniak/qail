package console

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/ubaniak/qail/internal/config"
	"github.com/ubaniak/qail/internal/runner"
)

func TestScopePath(t *testing.T) {
	cases := []struct{ ws, repo, want string }{
		{"", "", "/"},
		{"payments", "", "/#/ws/payments"},
		{"payments", "payments-api", "/#/ws/payments/payments-api"},
		{"my ws", "a/b", "/#/ws/my%20ws/a%2Fb"},
	}
	for _, c := range cases {
		if got := ScopePath(c.ws, c.repo); got != c.want {
			t.Errorf("ScopePath(%q,%q) = %q, want %q", c.ws, c.repo, got, c.want)
		}
	}
}

func TestOpenBrowserPerOS(t *testing.T) {
	cases := map[string]runner.Command{
		"darwin":  {Name: "open", Args: []string{"http://127.0.0.1:4317/"}},
		"linux":   {Name: "xdg-open", Args: []string{"http://127.0.0.1:4317/"}},
		"windows": {Name: "rundll32", Args: []string{"url.dll,FileProtocolHandler", "http://127.0.0.1:4317/"}},
	}
	for goos, want := range cases {
		rec := runner.NewRecorder()
		m := New(Options{Runner: rec, GOOS: goos})
		if err := m.OpenBrowser(context.Background(), m.URL("/")); err != nil {
			t.Fatalf("%s: %v", goos, err)
		}
		got := rec.LastCall()
		if got.Name != want.Name || len(got.Args) != len(want.Args) || got.Args[len(got.Args)-1] != want.Args[len(want.Args)-1] {
			t.Errorf("%s: got %+v, want %+v", goos, got, want)
		}
	}
}

func TestOpenBrowserNonZeroExit(t *testing.T) {
	rec := runner.NewRecorder().Respond(runner.Result{ExitCode: 1, Stderr: []byte("no display")}, nil)
	m := New(Options{Runner: rec, GOOS: "linux"})
	if err := m.OpenBrowser(context.Background(), "http://x"); err == nil {
		t.Fatal("expected error on non-zero exit")
	}
}

func TestStartReusesHealthyConsole(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })}
	go srv.Serve(ln)
	defer srv.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	// Something healthy already owns the port: Start is a no-op.
	m := New(Options{Store: config.NewMemoryStore(), Port: port})
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if m.Running() {
		t.Fatal("Manager should not own a console it didn't start")
	}
}

func TestStartServesAndStops(t *testing.T) {
	m := New(Options{Store: config.NewMemoryStore(), Port: freePort(t), DataDir: t.TempDir()})
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !m.Running() || !m.Healthy(context.Background()) {
		t.Fatal("console not serving after Start")
	}
	res, err := http.Get(m.URL("/"))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if !strings.Contains(string(body), "Agent Console") {
		t.Fatalf("index not embedded: %.200s", body)
	}
	m.Stop()
	if m.Running() || m.Healthy(context.Background()) {
		t.Fatal("console still serving after Stop")
	}
}

func TestVersionLess(t *testing.T) {
	if !versionLess("v20.11.0", "v22.1.0") || versionLess("v22.10.0", "v22.9.1") {
		t.Fatal("versionLess ordering wrong")
	}
}

func TestNvmBinsNewestFirst(t *testing.T) {
	dir := t.TempDir()
	for _, v := range []string{"v18.20.0", "v22.9.0", "v22.10.1"} {
		os.MkdirAll(filepath.Join(dir, v, "bin"), 0o755)
	}
	got := nvmBins(dir)
	if len(got) != 3 || filepath.Base(filepath.Dir(got[0])) != "v22.10.1" {
		t.Fatalf("got %v", got)
	}
}

func freePort(t *testing.T) int {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	p, _ := strconv.Atoi(ln.Addr().String()[len("127.0.0.1:"):])
	return p
}

