package console

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"

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

func TestStartNotInstalled(t *testing.T) {
	m := New(Options{Dir: t.TempDir(), Port: freePort(t)})
	err := m.Start(context.Background())
	if !errors.Is(err, ErrNotInstalled) {
		t.Fatalf("got %v, want ErrNotInstalled", err)
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
	// Not installed, but something healthy already owns the port: Start is a no-op.
	m := New(Options{Dir: t.TempDir(), Port: port})
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if m.Running() {
		t.Fatal("Manager should not own a process it didn't start")
	}
}

func TestInstalled(t *testing.T) {
	dir := t.TempDir()
	m := New(Options{Dir: dir})
	if m.Installed() {
		t.Fatal("empty dir reported installed")
	}
	os.WriteFile(filepath.Join(dir, "server.mjs"), []byte(""), 0o644)
	os.Mkdir(filepath.Join(dir, "node_modules"), 0o755)
	if !m.Installed() {
		t.Fatal("expected installed")
	}
}

func TestVersionLess(t *testing.T) {
	if !versionLess("v20.11.0", "v22.1.0") || versionLess("v22.10.0", "v22.9.1") {
		t.Fatal("versionLess ordering wrong")
	}
}

func TestNvmNodesNewestFirst(t *testing.T) {
	dir := t.TempDir()
	for _, v := range []string{"v18.20.0", "v22.9.0", "v22.10.1"} {
		os.MkdirAll(filepath.Join(dir, v, "bin"), 0o755)
	}
	got := nvmNodes(dir)
	if len(got) != 3 || filepath.Base(filepath.Dir(filepath.Dir(got[0]))) != "v22.10.1" {
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
